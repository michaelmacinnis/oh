// Released under an MIT license. See LICENSE.

// Package engine provides an evaluator for parsed oh code.
package engine

import (
	"os"
	"strconv"
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/interface/truth"
	"github.com/michaelmacinnis/oh/internal/common/struct/frame"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/env"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/common/type/obj"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/pipe"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/engine/boot"
	"github.com/michaelmacinnis/oh/internal/engine/task"
	"github.com/michaelmacinnis/oh/internal/reader"
	"github.com/michaelmacinnis/oh/internal/system/job"
)

func Boot(arguments []string) {
	if len(arguments) > 0 {
		args := make([]cell.I, 0, len(arguments))

		for n, s := range arguments {
			v := str.New(s)
			args = append(args, v)
			env0.Export(strconv.Itoa(n), v)
		}

		env0.Export("_args_", list.New(args[1:]...))
	}

	j := job.New(0)
	r := reader.New("boot.oh")

	for _, line := range strings.SplitAfter(boot.Script(), "\n") {
		c := r.Scan(line)
		if c != nil {
			Evaluate(j, c)
		}
	}
}

// TODO: Evaluate should call System.

// Evaluate evaluates the command c.
func Evaluate(j *job.T, c cell.I) cell.I {
	task0 := task.New(j, c, frame0)

	task0.PushOp(task.Action(task.EvalCommand))

	done := make(chan struct{})

	j.Spawn(nil, task0, func() {
		close(done)
	})

	<-done

	r := task0.Result()

	if task0.Exited() {
		exitcode, ok := status(r)
		if !ok {
			exitcode = success(r)
		}

		os.Exit(exitcode)
	}

	scope0.Define("?", r)

	return r
}

func Resolve(k string) (v string) {
	defer func () {
		r := recover()
		if r != nil {
			v = ""
		}
	}()

	_, r := frame0.Resolve(k)
	if r == nil {
		return
	}

	v = common.String(r.Get())

    return
}

// System evaluates the command c and returns the result.
func System(j *job.T, c cell.I) cell.I {
	t := task.New(j, c, frame0)

	t.PushOp(task.Action(task.EvalCommand))

	done := make(chan struct{})

	j.Spawn(nil, t, func() {
		close(done)
	})

	<-done

	return t.Result()
}

//nolint:gochecknoglobals
var (
	env0   scope.I
	frame0 *frame.T
	scope0 scope.I
)

func fg(t *task.T) task.Op {
	// TODO: Add error checking.
	n := 0

	args := t.Code()
	if args != pair.Null {
		s := common.String(pair.Car(t.Code()))
		n, _ = strconv.Atoi(s)
	}

	return t.Return(num.Int(job.Fg(pipe.W(t.CellValue("_stdout_")), n)))
}

func init() { //nolint:gochecknoinits
	ee := &task.Syntax{Op: task.Action(task.EvalExport)}

	env0 = env.New(nil)

	env0.Export("export", ee)

	env0.Export("_stdin_", pipe.New(os.Stdin, nil))
	env0.Export("_stdout_", pipe.New(nil, os.Stdout))
	env0.Export("_stderr_", pipe.New(nil, os.Stderr))

	// Environment variables.
	for _, entry := range os.Environ() {
		kv := strings.SplitN(entry, "=", 2)
		env0.Export(kv[0], sym.New(kv[1]))
	}

	scope0 = env.New(nil)

	scope0.Export("export", ee)

	scope0.Define("sys", obj.New(env0))

	// Methods.
	scope0.Define("fg", &task.Method{Op: task.Action(fg)})
	scope0.Define("jobs", &task.Method{Op: task.Action(jobs)})

	// Values.
	scope0.Define("False", boolean.False)
	scope0.Define("True", boolean.True)

	task.Actions(scope0)

	frame0 = frame.New(scope0, frame.New(env0, nil))
}

func jobs(t *task.T) task.Op {
	job.Jobs(pipe.W(t.CellValue("_stdout_")))

	return t.Return(num.Int(0))
}

func status(c cell.I) (exitcode int, ok bool) {
	defer func() {
		r := recover()
		if r != nil {
			ok = false
		}
	}()

	return int(integer.Value(c)), true
}

func success(c cell.I) (exitcode int) {
	defer func() {
		recover() //nolint:errcheck
	}()

	return map[bool]int{
		true:  0,
		false: 1,
	}[truth.Value(c)]
}
