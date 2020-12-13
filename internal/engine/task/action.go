// Released under an MIT license. See LICENSE.

package task

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/michaelmacinnis/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/interface/reference"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/struct/frame"
	"github.com/michaelmacinnis/oh/internal/common/type/create"
	"github.com/michaelmacinnis/oh/internal/common/type/env"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/obj"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/pipe"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
	"github.com/michaelmacinnis/oh/internal/engine/commands"
)

// Action performs a single step of the machine and returns the next operation.
type Action func(*T) Op

// Perform is required for an action to be an operation.
func (a Action) Perform(t *T) Op {
	return a(t)
}

// Actions associates actions with names in the scope s.
func Actions(s scope.I) {
	// Base.
	s.Define("block", &Syntax{Op: Action(block)})
	s.Define("define", &Syntax{Op: Action(evalDefine)})
	s.Define("if", &Syntax{Op: Action(evalIf)})
	s.Define("while", &Syntax{Op: Action(evalWhile)})
	s.Define("set", &Syntax{Op: Action(evalSet)})
	s.Define("spawn", &Syntax{Op: Action(spawn)})

	s.Define("get", &Method{Op: Action(get)})
	s.Define("eval", &Method{Op: Action(eval)})
	s.Define("set?", &Method{Op: Action(isSet)})
	s.Define("unset", &Method{Op: Action(unset)})

	// Builtins.
	s.Define("cd", &Builtin{Op: Action(cd)})
	s.Define("command", &Builtin{Op: Action(cmd)})

	for k, v := range commands.Builtins() {
		s.Define(k, b(v))
	}

	// Methods.
	s.Define("builtin?", &Method{Op: Action(isBuiltin)})
	s.Define("continuation?", &Method{Op: Action(isContinuation)})
	s.Define("exit", &Method{Op: Action(exit)})
	s.Define("fatal", &Method{Op: Action(fatal)})
	s.Define("interpolate", &Method{Op: Action(interpolate)})
	s.Define("method?", &Method{Op: Action(isMethod)})
	s.Define("resolve", &Method{Op: Action(resolve)})
	s.Define("resolves?", &Method{Op: Action(resolves)})
	s.Define("splice", &Method{Op: Action(splice)})
	s.Define("syntax?", &Method{Op: Action(isSyntax)})
	s.Define("trace", &Method{Op: Action(trace)})
	s.Define("wait", &Method{Op: Action(wait)})

	// Syntax.
	s.Define("builtin", &Syntax{Op: Action(builtin)})
	s.Define("method", &Syntax{Op: Action(method)})
	s.Define("syntax", &Syntax{Op: Action(syntax)})

	// Functions.
	for k, v := range commands.Functions() {
		s.Define(k, f(v))
	}
}

// Actions.

// EvalCommand triggers the evaluation of the head of a command so that
// execCommand can determine the next step.
//
// Result:
//  code:  Head
//  stack: implicitLookup Restore(code: Arg_0 ... Arg_N) execCommand Previous ...
//
// Requires:
//  code:  Head Arg_0 ... Arg_N
//  stack: EvalCommand Previous ...
//
// As a special case the Null command evaluates to Null.
//
func EvalCommand(t *T) Op {
	if t.code == pair.Null {
		return t.Return(pair.Null)
	}

	t.ReplaceOp(Action(execCommand))
	t.PushOp(&registers{code: pair.Cdr(t.code)})

	t.code = pair.Car(t.code)

	t.PushOp(Action(implicitLookup))

	return t.PushOp(Action(evalElement))
}

func EvalExport(t *T) Op {
	t.ReplaceOp(Action(execExport))
	t.PushOp(Action(evalArg))
	t.PushOp(Action(sublistKey))
	t.PushOp(&registers{code: pair.Cdr(t.code)})

	t.code = pair.Car(t.code)

	return t.PushOp(Action(evalArg))
}

func StringScope() scope.I {
	s := env.New(nil)

	for k, v := range commands.StringFunctions() {
		s.Export(k, f(v))
	}

	return obj.New(s)
}

// All commands are bound to the scope in which they were found.
type binding struct {
	command
	self cell.I
}

// Builtin, method, and syntax types all conform to the command interface.
type command interface {
	cell.I

	Closure() *Closure
	Execute(*T) Op
}

//nolint:gochecknoglobals
var (
	conduitScope = makeConduitScope()
	listScope    = makeListScope()
)

// accessMember looks for a command named Name in the object Object.
//
// Result:
//  code:  <undefined>
//  dump:  Name Object ...
//  stack: Previous ...
//
// Requires:
//  code:  <undefined>
//  dump:  Binding ...
//  stack: ...
//
func accessMember(t *T) Op {
	m := t.PopResult()

	o := t.Result()

	if !sym.Is(m) {
		panic("member name must be a symbol not a " + m.Name())
	}

	n := literal.String(m)

	var r reference.I

	switch o := o.(type) {
	case scope.I:
		r = o.Lookup(n)
	case conduit.I:
		r = conduitScope.Lookup(n)
	case *pair.T:
		r = listScope.Lookup(n)
	default:
		panic(m.Name() + " is not an object")
	}

	if r == nil {
		panic("'" + n + "' not defined")
	}

	v := r.Get()

	c, ok := v.(command)
	if !ok {
		panic(n + " is not executable")
	}

	t.ReplaceResult(bind(c, o))

	return t.PreviousOp()
}

// apply is the action for any user-defined builtin, method, or syntax.
//
// Result:
//  code:  Body
//  dump:  Binding ...
//  frame: New scope (possibly in new frame) with bindings for:
//           [calling env,] [self,] parameters (to arguments), 'return'.
//  stack: evalBlock Restore(frame: Current) Previous ...
//
// Requires:
//  code:  Arg_0 ... Arg_N
//  dump:  Binding ...
//  frame: Current
//  stack: apply Previous ...
//
// If the action is syntax the arguments are unevaluated. Method, the
// arguments have been evaluated. Builtin, evaluated and expanded.
//
func apply(t *T) Op {
	s := t.frame.Scope()

	b := bound(t.Result())

	c := b.Closure()

	e := env.New(c.Scope)

	t.ReplaceOp(&registers{frame: t.frame})

	cc := &registers{dump: t.dump, stack: t.stack}

	// If the visibility differs, create a new frame.
	if !c.Scope.Visible(s) {
		t.frame = frame.New(e, t.frame)
	} else {
		t.frame = frame.Dup(e, t.frame)
	}

	if c.Labels.Env != pair.Null {
		elabel := literal.String(c.Labels.Env)
		e.Define(elabel, s)
	}

	if c.Labels.Self != pair.Null {
		slabel := literal.String(c.Labels.Self)
		e.Define(slabel, scope.To(b.self))
	}

	args := t.code
	plabels := c.Labels.Params

	actual := int(list.Length(args))
	expected := int(list.Length(plabels))

	for args != pair.Null && plabels != pair.Null {
		label := pair.Car(plabels)
		if !sym.Is(label) {
			break
		}

		e.Define(literal.String(label), pair.Car(args))
		args, plabels = pair.Cdr(args), pair.Cdr(plabels)
	}

	rest := pair.Car(plabels)
	if plabels != pair.Null && pair.Is(rest) && pair.Cdr(rest) == pair.Null {
		e.Define(literal.String(pair.Car(rest)), args)
	} else if actual != expected {
		panic("expected " + validate.Count(expected, "argument", "s") + ", passed " + strconv.Itoa(actual))
	}

	t.code = c.Body

	e.Define("return", cc)

	return t.PushOp(Action(evalBlock))
}

// block evaluates a block of code in a new scope.
//
// Result:
//  code:  Cmd_0 ... Cmd_N
//  dump:  Null ...
//  frame: New scope
//  stack: evalBlock Restore(frame: Current) Previous ...
//
// Requires:
//  code:  Cmd_0 ... Cmd_N
//  dump:  Binding ...
//  frame: Current
//  stack: block Previous ...
//
func block(t *T) Op {
	t.ReplaceOp(&registers{frame: t.frame})

	t.frame = frame.Dup(env.New(t.frame.Scope()), t.frame)

	t.ReplaceResult(pair.Null)

	return t.PushOp(Action(evalBlock))
}

func continuation(t *T) Op {
	r := t.PopResult()

	t.Result().(*registers).restoreOver(t.registers)

	t.ReplaceResult(r)

	return t.Op()
}

// eval evaluates its argument in the scope provided by self.
//
// Result:
//  code:  <undefined>
//  dump:  Value ...
//  stack: Previous ...
//
// Requires:
//  code:  Expression ...
//  dump:  Binding ...
//  stack: eval Previous ...
//
func eval(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	b := bound(t.Result())

	t.ReplaceOp(&registers{frame: t.frame})

	t.code = v[0]

	// If the visibility differs, create a new frame.
	e := scope.To(b.self)
	if !e.Visible(t.frame.Scope()) {
		t.frame = frame.New(e, t.frame)
	} else {
		t.frame = frame.Dup(e, t.frame)
	}

	return t.PushOp(Action(evalElement))
}

// evalArg pushes the value pointed to by code as a result. If code points
// to a command (a pair), this value is replaced by the return value from
// evaluating the command.
//
// Result:
//  code:  Element
//  dump:  Element ...
//  stack: [EvalCommand if Element is a command] Previous ...
//
// Requires:
//  code:  Element
//  dump:  ...
//  stack: evalArg Previous ...
//
func evalArg(t *T) Op {
	t.PushResult(t.code)

	if pair.Is(t.code) {
		return t.ReplaceOp(Action(EvalCommand))
	}

	return t.PreviousOp()
}

// evalArgs evaluates arguments in the list pointed to by code.
//
// Result:
//  code:  <undefined>
//  dump:  EvaluatedArg_N ... EvaluatedArg_0 nil ...
//  stack: Previous ...
//
// Requires:
//  code:  Arg_i ...
//  dump:  EvaluatedArg_i-1 ... EvaluatedArg_0 nil ...
//  stack: evalArgs Previous ...
//
//
// When evalArgs is first invoked the registers look like this:
//
//  code:  Arg_0 ... Arg_N
//  dump:  nil ...
//  stack: evalArgs Previous ...
//
// While there are arguments to be evaluated, the next operation for evalArgs
// is EvalElement, evalArgs sets code to the current argument; and pushes a
// restore operation with code pointing to the rest of the argument list.
// If the current argument is the last argument evalArgs removes itself and
// does not push a restore operation. This allows the final EvalElement to
// return directly to the previous op. On each iteration, before EvalElement
// the registers look like this:
//
//  code:  Arg_i
//  dump:  EvaluatedArg_i-1 ... EvaluatedArg_0 nil ...
//  stack: EvalElement [Restore(code: Arg_i+1 ...) evalArgs] Previous ...
//
func evalArgs(t *T) Op {
	if t.code == pair.Null {
		return t.PreviousOp()
	}

	next := pair.Cdr(t.code)
	if next == pair.Null {
		t.RemoveOp()
	} else {
		t.PushOp(&registers{code: next})
	}

	t.code = pair.Car(t.code)

	return t.PushOp(Action(evalArg))
}

// evalBlock evaluates each command in the list pointed to by code.
//
// Result:
//  code:  <undefined> | else* ...
//  stack: Previous ...
//
// Requires:
//  code:  Cmd_i ...
//  dump:  RVal_i-1 ...
//  stack: evalBlock Previous ...
//
//
// When evalBlock is first invoked the registers look like this:
//
//  code:  Cmd_0 ... Cmd_N
//  dump:  ReplaceableValue ...
//  stack: evalArgs Previous ...
//
// While there are commands to be evaluated, the next operation for evalBlock
// is EvalCommand, evalBlock sets code to the current command; and pushes a
// restore operation with code pointing to the rest of the commands in the
// block. If the current command is the last command evalBlock removes itself
// and does not push a restore operation. This allows the final EvalCommand to
// return directly to the previous op. On each iteration, before EvalCommand
// the registers look like this:
//
//  code:  Cmd_i
//  dump:  ReplaceableValue|ReturnValue_i-1 ...
//  stack: EvalCommand [Restore(code: Cmd_i+1 ...) evalBlock] Previous ...
//
// To avoid "unnatural" looking conditionals because of the way lists are
// represented in oh, when the condition in an if-statement is false, oh has
// to scan through the list of commands looking for one that is the symbol
// 'else', not a list (which would be a command), and not Null (which would
// mean there is no else-clause). Conversely, if the condition is true the
// list of commands is executed until exhausted or until the 'else' symbol
// is encountered.
//
func evalBlock(t *T) Op {
	if t.code == pair.Null {
		// Empty block.
		return t.PreviousOp()
	}

	current := pair.Car(t.code)
	if !pair.Is(current) {
		// The most likely reason for this is an else-clause.
		// Or this is the foreground task and this is the spot
		// that will be filled in when evaluation of the block
		// resumes.
		return t.PreviousOp()
	}

	next := pair.Cdr(t.code)
	if next == pair.Null {
		t.RemoveOp()
	} else {
		t.PushOp(&registers{code: next})
	}

	t.code = current

	return t.PushOp(Action(EvalCommand))
}

// evalDefine associates name and value, privately, in the scope provided by self.
//
// Result:
//  code:  <undefined>
//  dump:  Value ...
//  stack: Previous ...
//
// Requires:
//  code:  Name Value ...
//  dump:  Binding ...
//  stack: define Previous ...
//
func evalDefine(t *T) Op {
	t.ReplaceOp(Action(execDefine))
	t.PushOp(Action(evalArg))
	t.PushOp(Action(sublistKey))
	t.PushOp(&registers{code: pair.Cdr(t.code)})

	t.code = pair.Car(t.code)

	return t.PushOp(Action(evalArg))
}

// TODO: Document.
func evalElement(t *T) Op {
	t.PopResult()

	return evalArg(t)
}

// evalIf creates new scope in which to execute an if-statement.
//
// Result:
//  code:  Condition (Consequence) [else (Alternative)]
//  dump:  Binding ...
//  frame: New scope
//  stack: execIfTest Restore(frame: Current) Previous ...
//
// Requires:
//  code:  Condition (Consequence) [else (Alternative)]
//  dump:  Binding ...
//  frame: Current
//  stack: evalIf Previous ...
//
func evalIf(t *T) Op {
	t.ReplaceOp(&registers{frame: t.frame})

	t.frame = frame.Dup(env.New(t.frame.Scope()), t.frame)

	return t.PushOp(Action(execIfTest))
}

func evalSet(t *T) Op {
	t.ReplaceOp(Action(execSet))
	t.PushOp(Action(evalArg))
	t.PushOp(Action(sublistKey))
	t.PushOp(&registers{code: pair.Cdr(t.code)})

	t.code = pair.Car(t.code)

	return t.PushOp(Action(evalArg))
}

// evalWhile creates new scope in which to execute a while-loop.
//
// Result:
//  code:  Condition (Block)
//  dump:  Binding ...
//  frame: New scope
//  stack: execWhileTest Restore(frame: Current) Previous ...
//
// Requires:
//  code:  Condition (Block)
//  dump:  Binding ...
//  frame: Current
//  stack: evalWhile Previous ...
//
func evalWhile(t *T) Op {
	t.ReplaceOp(&registers{frame: t.frame})

	t.frame = frame.Dup(env.New(t.frame.Scope()), t.frame)

	return t.PushOp(Action(execWhileTest))
}

// execBuiltin expands the evaluated arguments and then executes the builtin.
//
// Result:
//  code:  ExpandedEvaluatedArg_1 ... ExpandedEvaluatedArg_N
//  dump:  Binding ...
//  stack: Builtin ...
//
// Requires:
//  code:  <undefined>
//  dump:  EvaluatedArg_N ... EvaluatedArg_0 nil Binding ...
//  stack: execBuiltin Builtin ...
//
func execBuiltin(t *T) Op {
	t.code = t.expand(t.arguments())

	return t.PreviousOp()
}

// execCommand uses the value from evaluating the head of the command to
// determine how to execute the command.
//
// If the head of the command is a symbol or string, produces.
//  code:  Arg_0 ... Arg_N
//  dump:  EvaluatedHead nil EvaluatedHead (this will be replaced) ...
//  stack: evalArgs execBuiltin external resume Previous ...
//
// Otherwise the Execute method of the closure sets up operations. For
// Builtins and Methods the operations look similar: evalArgs, execBuiltin
// or execMethod, followed by the action for the closure and then the
// previous operation. For Syntax the only operation before Previous is
// the action for the closure.
//
// Requires:
//  code:  Arg_0 ... Arg_N
//  dump:  EvaluatedHead ...
//  stack: execCommand Previous ...
//
func execCommand(t *T) Op {
	r := t.Result()

	switch v := r.(type) {
	case scope.I, conduit.I, *pair.T:
		t.PushOp(&registers{code: pair.Cdr(t.code)})
		t.code = pair.Car(t.code)
		t.PushOp(Action(accessMember))

		return t.PushOp(Action(evalArg))

	case *str.T, *sym.Plus, *sym.T:
		t.ReplaceOp(Action(resume))
		t.PushOp(Action(external))
		t.PushOp(Action(execBuiltin))
		t.PushResult(nil)
		t.PushResult(v) // First arg is command name.

		return t.PushOp(Action(evalArgs))

	case *binding:
		return v.Execute(t)

	case *registers:
		t.ReplaceOp(Action(continuation))
		t.code = pair.Car(t.code)

		return t.PushOp(Action(evalArg))

	default:
		panic("unexpected problem evaluating command " + fmt.Sprintf("%T", r))
	}
}

func execDefine(t *T) Op {
	v := t.PopResult()
	k := literal.String(t.PopResult())
	b := bound(t.Result())

	scope.To(b.self).Define(k, v)

	return t.Return(v)
}

func execExport(t *T) Op {
	v := t.PopResult()
	k := literal.String(t.PopResult())
	b := bound(t.Result())

	scope.To(b.self).Export(k, v)

	return t.Return(v)
}

// execIfBody executes the branch of an if-statement indicated by condition.
//
// Result:
//  code:  Consequence | Alternative
//  dump:  EvaluatedCondition ...
//  stack: evalBlock Previous ...
//
// Requires:
//  code:  Condition (Consequence) [else (Alternative)]
//  dump:  EvaluatedCondition ...
//  stack: execIfBody Previous ...
//
func execIfBody(t *T) Op {
	alternate := !boolean.Value(t.Result())
	if alternate {
		t.code = pair.Cdr(t.code)
		c := pair.Car(t.code)

		for pair.Is(c) && c != pair.Null {
			t.code = pair.Cdr(t.code)
			c = pair.Car(t.code)
		}

		if c != pair.Null && literal.String(c) != "else" {
			panic("expected else")
		}
	}

	t.code = pair.Cdr(t.code)
	if t.code == pair.Null {
		return t.PreviousOp()
	}

	next := pair.Car(t.code)
	if alternate && !pair.Is(next) {
		if literal.String(next) != "if" {
			panic("expected if")
		}

		return t.ReplaceOp(Action(EvalCommand))
	}

	return t.ReplaceOp(Action(evalBlock))
}

// execIfTest evaluates an if-statment's condition and triggers a decision.
//
// Result:
//  code:  Condition
//  dump:  ...
//  stack: evalElement Restore(code: Condition ...) execIfBody Previous ...
//
// Requires:
//  code:  Condition (Consequence) [else (Alternative)]
//  dump:  Binding ...
//  stack: execIfTest Previous ...
//
func execIfTest(t *T) Op {
	t.ReplaceOp(Action(execIfBody))

	return execTest(t)
}

// execMethod executes the method.
//
// Result:
//  code:  EvaluatedArg_1 ... EvaluatedArg_N
//  dump:  Binding ...
//  stack: Method ...
//
// Requires:
//  code:  <undefined>
//  dump:  EvaluatedArg_N ... EvaluatedArg_0 nil Binding ...
//  stack: execMethod Method ...
//
func execMethod(t *T) Op {
	t.code = t.arguments()

	return t.PreviousOp()
}

func execSet(t *T) Op {
	v := t.PopResult()
	k := literal.String(t.PopResult())

	b := bound(t.Result())

	s := scope.To(b.self)
	r := s.Lookup(k)

	if r == nil {
		panic("'" + k + "' not defined")
	}

	r.Set(v)

	return t.Return(v)
}

// execTest is used by both execIfTest and execWhileTest.
func execTest(t *T) Op {
	t.PushOp(&registers{code: t.code})

	t.code = pair.Car(t.code)

	return t.PushOp(Action(evalElement))
}

// execWhileBody executes the body of a while-loop while condition is true.
//
// Result:
//  code:  Block
//  dump:  EvaluatedCondition ...
//  stack: evalBlock Restore(code: Condition ...) execWhileTest Previous ...
//
// Requires:
//  code:  Condition (Block)
//  dump:  EvaluatedCondition ...
//  stack: execWhileBody Previous ...
//
func execWhileBody(t *T) Op {
	if !boolean.Value(t.Result()) {
		return t.PreviousOp()
	}

	t.ReplaceOp(Action(execWhileTest))
	t.PushOp(&registers{code: t.code})

	t.code = pair.Cdr(t.code)

	return t.PushOp(Action(evalBlock))
}

// execWhileTest evaluates a while-loop's condition and triggers a decision.
//
// Result:
//  code:  Condition
//  dump:  ...
//  stack: evalElement Restore(code: Condition ...) execWhileBody Previous ...
//
// Requires:
//  code:  Condition (Block)
//  dump:  Binding ...
//  stack: execWhileTest Previous ...
//
func execWhileTest(t *T) Op {
	t.ReplaceOp(Action(execWhileBody))

	return execTest(t)
}

// TODO: Change this so that it sets up everything and then triggers another
// operation that can be restarted if necessary.
func external(t *T) Op {
	name := t.tildeExpand(common.String(pair.Car(t.code)))

	arg0, executable, err := adapted.LookPath(name, t.stringValue("PATH"))
	if err != nil {
		panic(err.Error())
	}

	if !executable {
		return t.Return(t.Chdir(name))
	}

	argv := []string{name}
	for args := pair.Cdr(t.code); args != pair.Null; args = pair.Cdr(args) {
		argv = append(argv, common.String(pair.Car(args)))
	}

	dir := t.stringValue("PWD")
	stdin := t.value(nil, "stdin")
	stdout := t.value(nil, "stdout")
	stderr := t.value(nil, "stderr")

	files := []*os.File{pipe.R(stdin), pipe.W(stdout), pipe.W(stderr)}

	attr := &os.ProcAttr{Dir: dir, Env: t.Environ(), Files: files}

	err = t.job.Execute(t, arg0, argv, attr)
	if err != nil {
		panic(err.Error())
	}

	return t.PreviousOp()
}

// implicitLookup runs after the first element in a command is evaluated.
// The first element is unique because symbols are implicitly resolved. If
// it is a symbol and there is a resolution for the symbol the symbol is
// replaced by the resolution. Otherwise it is left unchanged.
//
// Result:
//  code:  Head
//  dump:  Value | Head ...
//  stack: Previous ...
//
// Requires:
//  code:  Head
//  dump:  ReplaceableResult ...
//  stack: implicitLookup Previous ...
//
// If code points to a command (a pair), the current result is replaced by
// the return value from evaluating the command:
//
//  code:  Head
//  dump:  ReplaceableResult ...
//  stack: EvalCommand Previous ...
//
// If the symbol is a symbol plus, that is a symbol with contextual
// information (buffer label, line number, column), the current frame is
// updated with this information.
//
func implicitLookup(t *T) Op {
	c := t.Result()

	if plus, ok := c.(*sym.Plus); ok {
		t.frame.Update(plus.Source())
	}

	if sym.Is(c) {
		v := t.value(nil, common.String(c))
		if v != nil {
			c = v
		}
	}

	t.ReplaceResult(c)

	return t.PreviousOp()
}

// resume returns the result from an external command.
//
// Result:
//  code:  <undefined>
//  dump:  Result ...
//  stack: Previous ...
//
// Requires:
//  code:  <undefined>
//  dump:  ...
//  stack: resume Previous ...
//
func resume(t *T) Op {
	return t.Return(t.state.Value())
}

func spawn(t *T) Op {
	child := New(t.job, t.code, frame.Dup(env.New(t.frame.Scope()), t.frame))

	child.PushOp(Action(evalBlock))

	t.job.Spawn(t, child, nil)

	return t.Return(child)
}

func splice(t *T) Op {
	t.PopResult()

	v := validate.Fixed(t.code, 1, 1)

	for c := v[0]; c != pair.Null; c = pair.Cdr(c) {
		t.PushResult(pair.Car(c))
	}

	return t.PreviousOp()
}

func sublistKey(t *T) Op {
	s := literal.String(t.Result())

	if strings.HasSuffix(s, ":") {
		t.ReplaceResult(sym.New(strings.TrimSuffix(s, ":")))
	} else {
		t.code = pair.Car(t.code)
	}

	return t.PreviousOp()
}

// Adapters.

func b(do func(args cell.I) cell.I) *Builtin {
	return &Builtin{Op: Action(func(t *T) Op {
		return t.Return(do(t.code))
	})}
}

func f(do func(args cell.I) cell.I) *Method {
	return &Method{Op: Action(func(t *T) Op {
		return t.Return(do(t.code))
	})}
}

func m(do func(s cell.I, args cell.I) cell.I) *Method {
	return &Method{Op: Action(func(t *T) Op {
		return t.Return(do(bound(t.Result()).self, t.code))
	})}
}

// Helpers.

// Bind a command to a scope.
func bind(a command, self cell.I) *binding {
	return &binding{a, self}
}

// Convert a cell to a binding. Panic if not possible.
func bound(a cell.I) *binding {
	if b, ok := a.(*binding); ok {
		return b
	}

	panic(a.Name() + " is not a command")
}

func makeConduitScope() scope.I {
	s := env.New(nil)

	for k, v := range commands.ConduitMethods() {
		s.Export(k, m(v))
	}

	return obj.New(s)
}

func makeListScope() scope.I {
	s := env.New(nil)

	for k, v := range commands.ListMethods() {
		s.Export(k, m(v))
	}

	return obj.New(s)
}

// Builtins.

func cd(t *T) Op {
	dir := ""

	if t.code == pair.Null {
		_, r := t.frame.Resolve("HOME")
		dir = common.String(r.Get())
	} else {
		dir = t.tildeExpand(common.String(pair.Car(t.code)))
	}

	if dir == "-" {
		_, r := t.frame.Resolve("OLDPWD")
		dir = common.String(r.Get())
	}

	return t.Return(t.Chdir(dir))
}

func cmd(t *T) Op {
	validate.Variadic(t.code, 1, 1)

	t.ReplaceOp(Action(resume))

	return t.PushOp(Action(external))
}

// Methods.

func exit(t *T) Op {
	t.Exit()

	return fatal(t)
}

func fatal(t *T) Op {
	v := validate.Fixed(t.code, 0, 1)

	c := sym.True

	if len(v) > 0 {
		c = v[0]
	}

	t.stack = done

	t.ReplaceResult(c)

	return t.Op()
}

func get(t *T) Op {
	k := literal.String(pair.Car(t.code))

	b := bound(t.Result())

	s := scope.To(b.self)

	r := s.Lookup(k)
	if r == nil {
		panic("'" + k + "' not defined")
	}

	v := r.Get()
	if c, ok := v.(command); ok {
		v = bind(c, s)
	}

	return t.Return(v)
}

func interpolate(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	b := bound(t.Result())

	s := scope.To(b.self)

	cb := func(ref string) string {
		if ref == "$$" {
			return "$"
		}

		name := ref[1:]
		if name[0] == '{' {
			name = name[1 : len(name)-1]
		}

		return common.String(t.resolve(s, name))
	}
	// '!', '%', '*', '+', '-',
	// '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	// '?', '@',
	// 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
	// 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	// '[', ']', '^', '_',
	// 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
	// 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	r := regexp.MustCompile(`(?:\$\$)|(?:\${.+?})|(?:\$[!%*+\-0-9?@A-Z\[\]^_a-z]+)`)

	return t.Return(str.New(r.ReplaceAllStringFunc(common.String(v[0]), cb)))
}

func isBuiltin(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	b, ok := v[0].(*binding)
	if ok {
		_, ok = b.command.(*Builtin)
	}

	return t.Return(create.Bool(ok))
}

func isContinuation(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	_, ok := v[0].(*registers)

	return t.Return(create.Bool(ok))
}

func isMethod(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	b, ok := v[0].(*binding)
	if ok {
		_, ok = b.command.(*Method)
	}

	return t.Return(create.Bool(ok))
}

func isSet(t *T) Op {
	k := literal.String(pair.Car(t.code))

	b := bound(t.Result())

	s := scope.To(b.self)

	r := s.Lookup(k)

	return t.Return(create.Bool(r != nil))
}

func isSyntax(t *T) Op {
	v := validate.Fixed(t.code, 1, 1)

	b, ok := v[0].(*binding)
	if ok {
		_, ok = b.command.(*Syntax)
	}

	return t.Return(create.Bool(ok))
}

func resolve(t *T) Op {
	k := literal.String(pair.Car(t.code))

	b := bound(t.Result())

	s := scope.To(b.self)

	return t.Return(t.resolve(s, k))
}

func resolves(t *T) Op {
	k := literal.String(pair.Car(t.code))

	b := bound(t.Result())

	s := scope.To(b.self)

	v := t.value(s, k)

	return t.Return(create.Bool(v != nil))
}

func trace(t *T) Op {
	dup := *t.registers

	l := dup.frame.Loc()
	trace := []cell.I{}

	for dup.stack != done {
		r, ok := dup.Op().(*registers)
		if !ok {
			dup.PreviousOp()

			continue
		}

		n := dup.frame.Loc()
		if n != l {
			l = n
			if !strings.HasSuffix(l.Text, "# oh:omit-from-trace") {
				s := str.New(l.String() + ": " + l.Text)
				trace = append(trace, s)
			}
		}

		r.restoreOver(&dup)

		dup.PreviousOp()
	}

	return t.Return(list.Reverse(list.New(trace...)))
}

func unset(t *T) Op {
	k := literal.String(pair.Car(t.code))

	b := bound(t.Result())

	s := scope.To(b.self)

	return t.Return(create.Bool(s.Remove(k)))
}

func wait(t *T) Op {
	v := []*T{}

	var last *T

	for args := t.code; args != pair.Null; args = pair.Cdr(args) {
		c := pair.Car(args)

		t, ok := c.(*T)
		if !ok {
			panic("can't wait on " + c.Name())
		}

		v = append(v, t)
	}

	t.Wait()

	t.job.Await(func() {
		res := pair.Null
		if last != nil {
			res = last.Result()
		}

		t.Notify(res)
	}, t, v...)

	return t.ReplaceOp(Action(resume))
}

// Syntax.

func builtin(t *T) Op {
	return t.Return(bind((*Builtin)(t.Closure()), obj.New(t.frame.Scope())))
}

func method(t *T) Op {
	return t.Return(bind((*Method)(t.Closure()), obj.New(t.frame.Scope())))
}

func syntax(t *T) Op {
	return t.Return(bind((*Syntax)(t.Closure()), obj.New(t.frame.Scope())))
}
