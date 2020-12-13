// Released under an MIT license. See LICENSE.

// Package engine provides an evaluator for parsed oh code.
package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/boolean"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/interface/scope"
	"github.com/michaelmacinnis/oh/internal/common/struct/frame"
	"github.com/michaelmacinnis/oh/internal/common/type/env"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/obj"
	"github.com/michaelmacinnis/oh/internal/common/type/pipe"
	"github.com/michaelmacinnis/oh/internal/common/type/status"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
	"github.com/michaelmacinnis/oh/internal/engine/boot"
	"github.com/michaelmacinnis/oh/internal/engine/task"
	"github.com/michaelmacinnis/oh/internal/reader"
	"github.com/michaelmacinnis/oh/internal/system/job"
)

func Boot(path string, arguments []string) {
	sym.Cache(true)

	job.Monitor()

	if path != "" {
		path = filepath.Dir(path)
	}

	path, err := filepath.Abs(path)
	if err != nil {
		panic(err.Error())
	}

	env0.Export("ORIGIN", sym.New(path))

	pwd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	env0.Export("OLDPWD", sym.New(pwd))
	env0.Export("PWD", sym.New(pwd))

	if len(arguments) > 0 {
		args := make([]cell.I, 0, len(arguments))

		for n, s := range arguments {
			v := str.New(s)
			args = append(args, v)
			env0.Export(strconv.Itoa(n), v)
		}

		env0.Export("@", list.New(args[1:]...))
	}

	j := job.Job(0)
	r := reader.New("boot.oh")

	for _, line := range strings.SplitAfter(boot.Script(), "\n") {
		c, err := r.Scan(line)
		if err != nil {
			panic(err.Error())
		}

		if c != nil {
			System(j, c)
		}
	}

	sym.Cache(false)
}

// Evaluate evaluates the command c.
func Evaluate(j *job.T, c cell.I) cell.I {
	r, exited := System(j, c)

	if exited {
		code, ok := exitcode(r)
		if !ok {
			code = success(r)
		}

		os.Exit(code)
	}

	scope0.Define("?", r)

	return r
}

func Resolve(k string) (v string) {
	defer func() {
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

// System evaluates the command c returning the result and if the task exited.
func System(j *job.T, c cell.I) (cell.I, bool) {
	t := task.New(j, c, frame0)

	t.PushOp(task.Action(task.EvalCommand))

	done := make(chan struct{})

	j.Spawn(nil, t, func() {
		close(done)
	})

	<-done

	return t.Result(), t.Exited()
}

//nolint:gochecknoglobals
var (
	env0   scope.I
	frame0 *frame.T
	scope0 scope.I
)

func bg(t *task.T) task.Op {
	v := validate.Fixed(t.Code(), 0, 1)

	n := 0
	if len(v) > 0 {
		n = int(integer.Value(v[0]))
	}

	// TODO: Convert this to a function that returns what a wrapper needs.
	bt := job.Bg(pipe.W(t.CellValue("stdout")), n)
	if bt == nil {
		panic("job does not exist")
	}

	return t.Return(bt)
}

func exitcode(c cell.I) (code int, ok bool) {
	defer func() {
		r := recover()
		if r != nil {
			ok = false
		}
	}()

	return int(integer.Value(c)), true
}

func fg(t *task.T) task.Op {
	v := validate.Fixed(t.Code(), 0, 1)

	n := 0
	if len(v) > 0 {
		n = int(integer.Value(v[0]))
	}

	// TODO: Convert this to a function that returns what a wrapper needs.
	if !job.Fg(pipe.W(t.CellValue("stdout")), n) {
		panic("job does not exist")
	}

	return t.Return(sym.True)
}

func init() { //nolint:gochecknoinits
	ee := &task.Syntax{Op: task.Action(task.EvalExport)}

	env0 = env.New(nil)

	env0.Export("export", ee)

	env0.Export("stdin", pipe.New(os.Stdin, nil))
	env0.Export("stdout", pipe.New(nil, os.Stdout))
	env0.Export("stderr", pipe.New(nil, os.Stderr))

	// Environment variables.
	for _, entry := range os.Environ() {
		kv := strings.SplitN(entry, "=", 2)
		env0.Export(kv[0], sym.New(kv[1]))
	}

	scope0 = env.New(nil)

	scope0.Export("export", ee)

	scope0.Define("str", task.StringScope())
	scope0.Define("sys", obj.New(env0))

	// Methods.
	scope0.Define("bg", &task.Method{Op: task.Action(bg)})
	scope0.Define("fg", &task.Method{Op: task.Action(fg)})
	scope0.Define("jobs", &task.Method{Op: task.Action(jobs)})

	task.Actions(scope0)

	frame0 = frame.New(scope0, frame.New(env0, nil))
}

func jobs(t *task.T) task.Op {
	// TODO: Convert this to a function that returns what a wrapper needs.
	job.Jobs(pipe.W(t.CellValue("stdout")))

	return t.Return(status.Int(0))
}

func success(c cell.I) (exitcode int) {
	defer func() {
		recover() //nolint:errcheck
	}()

	return map[bool]int{
		true:  0,
		false: 1,
	}[boolean.Value(c)]
}
