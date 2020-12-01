// Released under an MIT license. See LICENSE.

// WCONTINUED is missing on NetBSD.

// +build aix darwin dragonfly freebsd linux openbsd solaris

package job

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"

	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/engine/task"
	"github.com/michaelmacinnis/oh/internal/system/options"
	"github.com/michaelmacinnis/oh/internal/system/process"
	"golang.org/x/sys/unix"
)

type T struct {
	group   int
	initial int
	lines   []string
	main    *task.T

	running map[int]*task.T
	stopped map[int]*task.T
}

type wg struct {
	fn func()
	on map[*task.T]struct{}
}

func Job(group int) *T {
	r := make(chan *T)

	requestq <- func() {
		r <- &T{
			group:   group,
			initial: group,
			running: map[int]*task.T{},
			stopped: map[int]*task.T{},
		}
	}

	return <-r
}

func New(group int) *T {
	r := make(chan *T)

	requestq <- func() {
		foreground = &T{
			group:   group,
			initial: group,
			running: map[int]*task.T{},
			stopped: map[int]*task.T{},
		}

		r <- foreground
	}

	return <-r
}

func (j *T) Append(line string) {
	j.lines = append(j.lines, line)
}

func (j *T) Await(fn func(), p *task.T, ts ...*task.T) {
	requestq <- func() {
		w := &wg{
			fn: fn,
			on: map[*task.T]struct{}{},
		}

		if len(ts) == 0 {
			if foreground != nil && p == foreground.main {
				ts = background
			} else {
				ts = children[p]
			}
		}

		for _, t := range ts {
			if _, found := parent[t]; !found {
				continue
			}
			w.on[t] = struct{}{}

			wgs, ok := waiting[t]
			if !ok {
				wgs = map[*wg]struct{}{}
				waiting[t] = wgs
			}
			wgs[w] = struct{}{}
		}

		if len(w.on) == 0 {
			fn()
		}
	}
}

func (j *T) Execute(t *task.T, path string, argv []string, attr *os.ProcAttr) error {
	errq := make(chan error)

	requestq <- func() {
		reap()

		foregroundProcess := options.Monitor() && j == foreground

		attr.Sys = process.SysProcAttr(foregroundProcess, j.group)

		p, err := os.StartProcess(path, argv, attr)
		if err == nil {
			t.Wait()

			if j.group == 0 {
				j.group = p.Pid
			}

			j.running[p.Pid] = t
			active[p.Pid] = j
		}

		// Ack with error.
		errq <- err

		close(errq)
	}

	// Wait for an ack.
	return <-errq
}

func (j *T) Spawn(p, c *task.T, fn func()) {
	done := make(chan struct{})

	requestq <- func() {
		parent[c] = p
		if p != nil {
			children[p] = append(children[p], c)
			if foreground != nil && p == foreground.main {
				background = append(background, c)
			}
		} else {
			j.main = c
		}

		if fn != nil {
			w := &wg{
				fn: fn,
				on: map[*task.T]struct{}{c: {}},
			}

			wgs, ok := waiting[c]
			if !ok {
				wgs = map[*wg]struct{}{}
				waiting[c] = wgs
			}
			wgs[w] = struct{}{}
		}

		go c.Run()

		// Ack.
		close(done)
	}

	// Wait for an ack.
	<-done
}

func (j *T) Stopped(t *task.T) {
	requestq <- func() {
		t.Stopped()

		if !t.Completed() {
			if foreground != nil && foreground.main == t {
				tell(t)

				number := 1
				for n := range jobs {
					if n >= number {
						number = n + 1
					}
				}
				jobs[number] = foreground

				printJobs(os.Stdout, number)
			}

			return
		}

		tell(t)

		if p, found := parent[t]; found {
			delete(parent, t)

			children[p] = remove(children[p], t)
			background = remove(background, t)
		}
	}
}

func Fg(w io.Writer, n int) int {
	r := make(chan int)

	requestq <- func() {
		if len(jobs) == 0 {
			r <- 1
			close(r)

			return
		}

		if n == 0 {
			n = jobNumbers()[len(jobs)-1]
		}

		j, found := jobs[n]
		if !found {
			r <- 1
			close(r)

			return
		}

		wgs := waiting[foreground.main]
		for wg := range wgs {
			wg.on[j.main] = wg.on[foreground.main]
			delete(wg.on, foreground.main)
		}
		waiting[j.main] = waiting[foreground.main]
		delete(waiting, foreground.main)

		foreground.main = j.main

		delete(jobs, n)

		for _, line := range j.lines {
			fmt.Fprintf(w, "%s\n", line)
		}

		if j.group > 0 {
			process.SetForegroundGroup(j.group)
		} else {
			process.SetForegroundGroup(process.Group())
		}

		for pid, t := range j.stopped {
			process.Continue(pid)
			j.running[pid] = t
		}

		go foreground.main.Run()

		close(r)
	}

	return <-r
}

func Jobs(w io.Writer) {
	r := make(chan struct{})

	requestq <- func() {
		printJobs(w, 0)

		close(r)
	}

	<-r
}

func Monitor() {
	signals := []os.Signal{unix.SIGCHLD}

	if options.Monitor() {
		signal.Ignore(unix.SIGQUIT, unix.SIGTTIN, unix.SIGTTOU)

		signals = append(signals, unix.SIGINT, unix.SIGTSTP)
	}

	requestq = make(chan func(), 1)
	signalq = make(chan os.Signal, len(signals)+1)

	signal.Notify(signalq, signals...)

	go monitor()
}

//nolint:gochecknoglobals
var (
	foreground *T

	requestq chan func()
	signalq  chan os.Signal

	active     = map[int]*T{}
	background = []*task.T{}
	children   = map[*task.T][]*task.T{}
	jobs       = map[int]*T{}
	parent     = map[*task.T]*task.T{}

	// Waiting is a map from a task to a set of "wait groups".
	// Each "wait group" is a set of tasks being waited "on"
	// and a function "fn" to call when the set becomes empty.
	// If we wanted to wait on tasks 1 and 3 we would create
	// a wait group containing 1 and 3 and a callback function
	// and then add this same wait group to the set of wait
	// groups for both tasks 1 and 3.
	waiting = map[*task.T]map[*wg]struct{}{}
)

func (j *T) interrupt() {
	interrupt(j.main)

	for pid := range j.running {
		process.Interrupt(pid)
	}
}

func (j *T) notify(pid int, status unix.WaitStatus) {
	if status.Continued() {
		t, found := j.stopped[pid]
		if !found {
			println("UNKNOWN PID CONTINUED", pid)

			return
		}

		if len(j.stopped) == 1 {
			// The last stopped process is running. Resume the job.
			j.resume()
		}

		j.running[pid] = t
		delete(j.stopped, pid)

		return
	}

	t, found := j.running[pid]
	if !found {
		println("UNKNOWN PID STATUS CHANGE", pid)

		return
	}

	if status.Stopped() {
		if len(j.running) == 1 {
			// The last running process is stopping. Stop the task.
			j.stop()
		}

		j.stopped[pid] = t
		delete(j.running, pid)

		return
	}

	code := int(status)

	switch {
	case status.Exited():
		code = status.ExitStatus()

	case status.Signaled():
		code += 128

	default:
		return
	}

	delete(j.running, pid)
	delete(active, pid)

	if len(j.running) == 0 && len(j.stopped) == 0 {
		j.group = j.initial
	}

	t.Notify(num.Int(code))
}

func (j *T) resume() {
	resume(j.main)

	for pid := range j.running {
		process.Continue(pid)
	}
}

func (j *T) stop() {
	stop(j.main)

	for pid := range j.running {
		process.Stop(pid)
	}
}

func interrupt(t *task.T) {
	for _, child := range children[t] {
		interrupt(child)
	}

	t.Interrupt()
}

func jobNumbers() []int {
	i := make([]int, 0, len(jobs))
	for k := range jobs {
		i = append(i, k)
	}

	sort.Ints(i)

	return i
}

func monitor() {
	for {
		select {
		case f := <-requestq:
			f()

		// If this process receives a SIGINT or SIGTSTP signal
		// then it must be in the foreground. Which means that
		// there are currently no other foreground processes.
		// So we can interrupt/stop execution of the current
		// task and any subtasks and don't need to worry about
		// forwarding the SIGINT/SIGTSTP to any child processes.
		//
		// TODO: What do we do for processes that would have
		// been started but we processed this signal first?
		// For a stopped job we could queue requests.
		// For a cancelled one we could return an exit status
		// of SIGINT + 130, immediately.
		case s := <-signalq:
			switch s {
			case unix.SIGCHLD:
				reap()

			case unix.SIGINT:
				foreground.interrupt()

			case unix.SIGTSTP:
				foreground.stop()
			}
		}
	}
}

func printJobs(w io.Writer, n int) {
	for k, v := range jobNumbers() {
		if n != 0 && v != n {
			continue
		}

		label := fmt.Sprintf("[%d]", v)
		if k == len(jobs)-1 {
			label += "+"
		}

		for _, line := range jobs[v].lines {
			fmt.Fprintf(w, "%s\t%s\n", label, line)
			label = "    "
		}
	}
}

func reap() {
	var (
		rusage unix.Rusage
		status unix.WaitStatus
	)

	options := unix.WNOHANG | unix.WUNTRACED | unix.WCONTINUED

	for {
		pid, _ := unix.Wait4(-1, &status, options, &rusage)
		if pid <= 0 {
			break
		}

		j, ok := active[pid]
		if ok {
			j.notify(pid, status)
		} else {
			println("UNKNOWN PID", pid)
		}
	}
}

func remove(cs []*task.T, t *task.T) []*task.T {
	n := -1

	for i, c := range cs {
		if c == t {
			n = i

			break
		}
	}

	if n == -1 {
		return cs
	}

	// We know len(cs) is at least 1.
	last := len(cs) - 1

	cs[n] = cs[last]
	cs[last] = nil

	return cs[:last]
}

func resume(t *task.T) {
	for _, child := range children[t] {
		resume(child)

		go child.Run()
	}
}

func stop(t *task.T) {
	for _, child := range children[t] {
		stop(child)
	}

	t.Stop()
}

func tell(t *task.T) {
	wgs, found := waiting[t]
	if !found {
		return
	}

	for w := range wgs {
		delete(w.on, t)

		if len(w.on) == 0 {
			w.fn()
			delete(wgs, w)
		}
	}

	if len(wgs) == 0 {
		delete(waiting, t)
	}
}
