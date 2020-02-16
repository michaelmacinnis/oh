// Released under an MIT license. See LICENSE.

// Package secd provides the stack-based abstract machine used by oh tasks.
package secd

import (
	"github.com/michaelmacinnis/oh/internal/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/interface/literal"
	"github.com/michaelmacinnis/oh/internal/interface/scope"
	"github.com/michaelmacinnis/oh/internal/type/env"
	"github.com/michaelmacinnis/oh/internal/type/errnum"
	"github.com/michaelmacinnis/oh/internal/type/frame"
	"github.com/michaelmacinnis/oh/internal/type/list"
	"github.com/michaelmacinnis/oh/internal/type/pair"
	"github.com/michaelmacinnis/oh/internal/type/str"
	"github.com/michaelmacinnis/oh/internal/type/sym"

	"reflect"
	"runtime"
	"strings"
)

// Action performs a single step of the machine and returns the next state.
type Action func(*Machine) State

// Closure underlies the builtin, method, and syntax types.
type Closure struct {
	Body cell.T // Body of the routine.
	Labels
	Scope scope.T
	State
}

// Labels hold the labels for a user-defined routine.
type Labels struct {
	Env    cell.T   // Calling env label.
	Params cell.T   // Param labels.
	Self   cell.T   // Label for the env where this routine was found.
}

// Machine is a stack-based abstract machine.
type Machine struct {
	// TODO: Put lexical scope.T back.
	// TODO: Some of these need to be grouped together as a
	// contintuation but we'll do that later as it is easier
	// for now just to have everything available.
	*stack          // (S)tack
	frame  frame.T // dynamic and lexical (E)nvironment
	code   cell.T   // (C)ode a.k.a. (C)ontrol
	dump   cell.T   // (D)ump
}

// State is a single step of the machine.
type State interface {
	Do(*Machine) State
}

// Actions.
var (
	// NOTE: It would be nice to define all actions here rather
	// than using explicit Action(...) in the functions below
	// but the compiler complains about initialization loops.
	EvalArgs    = Action(evalArgs)
	EvalBlock   = Action(evalBlock)
	ExecBuiltin = Action(execBuiltin)
	ExecMethod  = Action(execMethod)
)

// New creates a new abstract machine.
func New(c cell.T) *Machine {
	return &Machine{
		code:  c,
		dump:  pair.Cons(errnum.New("0"), pair.Null),
		frame: *frame.New(scope0),
		stack: done,
	}
}

// Do is required for an action to be a state.
func (a Action) Do(m *Machine) State {
	return a(m)
}

func (m *Machine) Closure() *Closure {
	slabel := pair.Car(m.code)
	m.code = pair.Cdr(m.code)

	plabels := slabel
	if sym.Is(slabel) {
		plabels = pair.Car(m.code)
		m.code = pair.Cdr(m.code)
	} else {
		slabel = pair.Null
	}

	equals := pair.Car(m.code)
	m.code = pair.Cdr(m.code)

	elabel := pair.Null
	if literal.String(equals) != "=" {
		elabel = equals
		equals = pair.Car(m.code)
		m.code = pair.Cdr(m.code)
	}

	if literal.String(equals) != "=" {
		panic("expected '='")
	}

	return &Closure{
		Body: m.code,
		Labels: Labels{
			Env: elabel,
			Params: plabels,
			Self: slabel,
		},
		Scope: m.frame.Scope(),
		State: Action(apply),
	}
}

// Do copies non-nil fields from saved to m.
func (saved *Machine) Do(m *Machine) State {
	if saved.code != nil {
		m.code = saved.code
	}

	if saved.frame != frame0 {
		m.frame = saved.frame
	}

	return m.PreviousState()
}

// NewState pushes a new state onto the stack.
func (m *Machine) NewState(s State) State {
	// TODO: Condense restore states.
	/*
	if a, ok := s.(Action); ok {
		println("NewState("+funcName(a)+")")
	} else {
		println("NewState(SaveState)")
	}
	*/
	m.stack = &stack{m.stack, s}
	return s
}

// PreviousState pops the current state and returns the previous state.
func (m *Machine) PreviousState() State {
	m.RemoveState()
	return m.State()
}

// PushResult adds the result r to dump.
func (m *Machine) PushResult(r cell.T) {
	//println("push result")
	m.dump = pair.Cons(r, m.dump)
}

// RemoveState pops the current state off the stack.
func (m *Machine) RemoveState() {
	/*
	if a, ok := m.stack.state.(Action); ok {
		println("RemoveState("+funcName(a)+")")
	} else {
		println("RemoveState(SaveState)")
	}
	*/
	m.stack = m.stack.stack
}

// ReplaceResult replaced the current result.
func (m *Machine) ReplaceResult(r cell.T) {
	//println("replace result")
	m.dump = pair.Cons(r, pair.Cdr(m.dump))
}

// Result returns the current result.
func (m *Machine) Result() cell.T {
	return pair.Car(m.dump)
}

// ReplaceState replaces the state at the top of the stack.
func (m *Machine) ReplaceState(s State) State {
	m.RemoveState()
	return m.NewState(s)
}

// State returns the abstract machine's current state.
func (m *Machine) State() State {
	return m.stack.state
}

func StateString(s State) string {
	if s == nil {
		return "<nil>"
	}
	if a, ok := s.(Action); ok {
		return funcName(a)
	}
	return "Save"
}

// Step performs a single action and determine the next action.
func (m *Machine) Step(s State) State {
	/*
	println("Performing:", StateString(s))
	println("Stack:")
	for p := m.stack; p != nil && p.state != nil; p = p.stack {
		println(StateString(p.state))
	}
	println("---")
	println("")
	*/

	return s.Do(m)
}

// All commands are bound to the scope in which they were found.
type binding struct {
	command
	self scope.T
}

// Builtin, method, and syntax types all conform to the command interface.
type command interface {
	cell.T

	Closure() *Closure
	Execute(*Machine) State
}

// The stack type is a machine's execution stack.
type stack struct {
	*stack
	state State
}

var (
	done = &stack{}
	frame0 frame.T
	scope0 scope.T
)

func (m *Machine) MoreArguments() bool {
	return m.dump != pair.Null
}

func (m *Machine) PopResult() cell.T {
	//println("pop result")
	r := pair.Car(m.dump)
	m.dump = pair.Cdr(m.dump)

	return r
}

func (m *Machine) arguments() cell.T {
	e := m.PopResult()
	l := pair.Null

	for e != nil && m.MoreArguments() {
		l = pair.Cons(e, l)

		e = m.PopResult()
	}

	return l
}

func (m *Machine) expand(l cell.T) cell.T {
	// TODO: Actually do expansion.
	return l
}

func (m *Machine) selectBranch() bool {
	if !boolean.Value(m.Result()) {
		m.code = pair.Cdr(m.code)

		c := pair.Car(m.code)
		for pair.Is(c) && c != pair.Null {
			m.code = pair.Cdr(m.code)
			c = pair.Car(m.code)
		}

		if c != pair.Null && literal.String(c) != "else" {
			panic("expected else")
		}
	}

	return pair.Cdr(m.code) != pair.Null
}


// Action functions.

func _lookup_(m *Machine) State {
	b := bound(m.Result())

	s := literal.String(pair.Car(m.code))
	r := b.self.Lookup(s)
	if r == nil {
		panic(s + "not defined")
	}

	m.ReplaceResult(r.Get())

	return m.PreviousState()
}

func _join_(m *Machine) State {
	var create func(string) cell.T = sym.New
	var joined strings.Builder

	for code := m.code; code != pair.Null; code = pair.Cdr(code) {
		c := pair.Car(code)

		switch c := c.(type) {
		case *str.T:
			create = str.New
			joined.WriteString(c.String())
		case *sym.T:
			joined.WriteString(c.String())
		default:
			panic("only strings and symbols can be joined")
		}
	}

	m.ReplaceResult(create(joined.String()))

	return m.PreviousState()
}

func apply(m *Machine) State {
	s := m.frame.Scope()

	b := bound(m.Result())

	c := b.Closure()

	m.ReplaceState(&Machine{frame: m.frame})

	if !c.Scope.Visible(s) {
		m.frame.New()
	}
	m.frame.SetScope(c.Scope)

	if c.Labels.Env != pair.Null {
		m.frame.Scope().Define(literal.String(c.Labels.Env), s)
	}

	if c.Labels.Self != pair.Null {
		m.frame.Scope().Define(literal.String(c.Labels.Self), b.self.Expose())
	}

	args := m.code
	plabels := c.Labels.Params

	actual := list.Length(args)
	expected := list.Length(plabels)

	for args != pair.Null && plabels != pair.Null {
		label := pair.Car(plabels)
		if !sym.Is(label) {
			break
		}

		m.frame.Scope().Define(literal.String(label), pair.Car(args))
		args, plabels = pair.Cdr(args), pair.Cdr(plabels)
	}

	rest := pair.Car(plabels)
	if plabels != pair.Null && pair.Is(rest) && pair.Cdr(rest) == pair.Null {
		m.frame.Scope().Define(literal.String(pair.Car(rest)), args)
	} else if actual != expected {
		panic("wrong number of arguments") // TODO: Better message.
	}

	m.code = c.Body

	// m.scope.Define("return", m.CurrentContinuation())

	return m.NewState(Action(evalBlock))
}

func block(m *Machine) State {
	m.ReplaceState(&Machine{frame: m.frame})

	m.frame.SetScope(env.New(m.frame.Scope()))

	m.ReplaceResult(pair.Null)

	return m.NewState(EvalBlock)
}

func builtin(m *Machine) State {
	v := (*Builtin)(m.Closure())

	m.ReplaceResult(v)

	return m.PreviousState()
}

func debug(m *Machine) State {
	println("debug:", literal.String(m.code))

	return m.PreviousState()
}

func define(m *Machine) State {
	b := bound(m.Result())

	s := literal.String(pair.Car(m.code))
	v := pair.Cadr(m.code)

	b.self.Define(s, v)

	m.ReplaceResult(v)

	return m.PreviousState()
}

func evalArgs(m *Machine) State {
	if m.code == pair.Null {
		return m.PreviousState()
	}

	m.NewState(&Machine{code: pair.Cdr(m.code)})
	m.code = pair.Car(m.code)

	return m.NewState(Action(evalElement))
}

func evalBlock(m *Machine) State {
	if m.code == pair.Null {
		// Empty block.
		return m.PreviousState()
	}

	current := pair.Car(m.code)
	if current == nil {
		// This should only happen for the foreground task.
		// This is the spot that will be filled in with the next
		// instruction when evaluation of the block resumes.
		return m.PreviousState()
	}

	if !pair.Is(current) {
		// Most likely reason for this is an else-clause.
		return m.PreviousState()
	}

	next := pair.Cdr(m.code)
	if next != pair.Null {
		m.NewState(&Machine{code: next})
	} else {
		m.RemoveState()
	}

	//println(list.Length(m.dump))

	m.code = current

	return m.NewState(Action(evalCommand))
}

func evalCommand(m *Machine) State {
	if m.code == pair.Null {
		m.ReplaceResult(pair.Null)
		return m.PreviousState()
	}

	if plus, ok := m.code.(*pair.Plus); ok {
		m.frame.Update(plus.Source())
	}

	m.ReplaceState(Action(execCommand))
	m.NewState(&Machine{code: pair.Cdr(m.code)})

	m.code = pair.Car(m.code)

	return m.NewState(Action(evalHead))
}

func evalElement(m *Machine) State {
	m.PushResult(m.code)

	if pair.Is(m.code) {
		return m.ReplaceState(Action(evalCommand))
	}

	return m.PreviousState()
}

func evalHead(m *Machine) State {
	if pair.Is(m.code) {
		return m.ReplaceState(Action(evalCommand))
	}

	v := m.code
	if sym.Is(v) {
		s, r := m.frame.Resolve(literal.String(v))
		if r != nil {
			v = r.Get()
			if c, ok := v.(command); ok {
				v = bind(c, s)
			}
		}
	}

	m.ReplaceResult(v)

	return m.PreviousState()
}

func evalIf(m *Machine) State {
	m.ReplaceState(&Machine{frame: m.frame})

	m.frame.SetScope(env.New(m.frame.Scope()))

	return m.NewState(Action(execIfTest))
}

func evalWhile(m *Machine) State {
	m.ReplaceState(&Machine{frame: m.frame})

	m.frame.SetScope(env.New(m.frame.Scope()))

	return m.NewState(Action(execWhileTest))
}

func execBuiltin(m *Machine) State {
	m.code = m.expand(m.arguments())

	return m.PreviousState()
}

func execCommand(m *Machine) State {
	c := m.Result()

	switch c := c.(type) {
	case *str.T, *sym.T:
		m.ReplaceState(Action(external))
		m.NewState(ExecBuiltin)
		m.PushResult(nil)
		m.PushResult(c) // First arg is command name.
		m.NewState(Action(evalArgs))
	case *binding:
		c.Execute(m)
	default:
		panic("unexpected problem evaluating command")
	}

	return m.State()
}

func execIfBody(m *Machine) State {
	if !m.selectBranch() {
		return m.PreviousState()
	}

	m.code = pair.Cdr(m.code)

	return m.ReplaceState(EvalBlock)
}

func execIfTest(m *Machine) State {
	m.ReplaceState(Action(execIfBody))

	return execTest(m)
}

func execMethod(m *Machine) State {
	m.code = m.arguments()

	return m.PreviousState()
}

func execTest(m *Machine) State {
	m.NewState(&Machine{code: m.code})

	m.code = pair.Car(m.code)
	m.PopResult() // Make room for evalElement to push its result.

	return m.NewState(Action(evalElement))
}

func execWhileBody(m *Machine) State {
	if !m.selectBranch() {
		return m.PreviousState()
	}

	m.ReplaceState(Action(execWhileTest))
	m.NewState(&Machine{code: m.code})

	m.code = pair.Cdr(m.code)

	return m.NewState(EvalBlock)
}

func execWhileTest(m *Machine) State {
	m.ReplaceState(Action(execWhileBody))

	return execTest(m)
}

func export(m *Machine) State {
	b := bound(m.Result())

	s := literal.String(pair.Car(m.code))
	v := pair.Cadr(m.code)

	b.self.Export(s, v)

	m.ReplaceResult(v)

	return m.PreviousState()
}

func external(m *Machine) State {
	println("external:", literal.String(m.code))

	return m.PreviousState()
}

func method(m *Machine) State {
	v := (*Method)(m.Closure())

	m.ReplaceResult(v)

	return m.PreviousState()
}

func set(m *Machine) State {
	b := bound(m.Result())

	s := literal.String(pair.Car(m.code))
	v := pair.Cadr(m.code)

	r := b.self.Lookup(s)
	if r == nil {
		panic(s + "not defined")
	}

	r.Set(v)

	m.ReplaceResult(v)

	return m.PreviousState()
}

func syntax(m *Machine) State {
	v := (*Syntax)(m.Closure())

	m.ReplaceResult(v)

	return m.PreviousState()
}


// Helpers.

// Bind a command to a scope.
func bind(a command, self scope.T) *binding {
	return &binding{a, self}
}

// Convert a cell to a binding. Panic if not possible.
func bound(a cell.T) *binding {
	if b, ok := a.(*binding); ok {
		return b
	}
	panic(a.Name() + " is not a command")
}

// Get the function i's name. Useful for debugging.
func funcName(i interface{}) string {
	n := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	a := strings.Split(n, ".")

	l := len(a)
	if l == 0 {
		return n
	}

	return a[l-1]
}

func init() {
	done.stack = done

	// Create top-level environment
	s := env.New(nil)

	s.Define("debug", &Method{State: Action(debug)})

	s.Define("_lookup_", &Method{State: Action(_lookup_)})
	s.Define("_join_", &Method{State: Action(_join_)})
	s.Define("define", &Method{State: Action(define)})
	s.Define("export", &Method{State: Action(export)})
	s.Define("set", &Method{State: Action(set)})

	s.Define("block", &Syntax{State: Action(block)})
	s.Define("builtin", &Syntax{State: Action(builtin)})
	s.Define("if", &Syntax{State: Action(evalIf)})
	s.Define("method", &Syntax{State: Action(method)})
	s.Define("syntax", &Syntax{State: Action(syntax)})
	s.Define("while", &Syntax{State: Action(evalWhile)})

	s.Define("success", errnum.New("0"))
	s.Define("failure", errnum.New("1"))

	scope0 = s
}
