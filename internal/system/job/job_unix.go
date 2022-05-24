// Released under an MIT license. See LICENSE.

// WCONTINUED is missing on NetBSD.

//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package job

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"

	"github.com/michaelmacinnis/oh/internal/common/type/status"
	"github.com/michaelmacinnis/oh/internal/engine/task"
	"github.com/michaelmacinnis/oh/internal/system/options"
	"github.com/michaelmacinnis/oh/internal/system/process"
	"golang.org/x/sys/unix"
)

// T (job) corresponds to a command entered by the user.
type T struct {
	group   int
	initial int
	lines   []string
	main    *task.T

	running map[int]*task.T
	stopped map[int]*task.T
}

type job = T

type wg struct {
	fn func()
	on map[*task.T]struct{}
}

// Job create a new (non-foreground) job.
func Job(group int) *T {
	r := make(chan *job)

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

// New creates a new foreground job.
func New(group int) *T {
	r := make(chan *job)

	requestq <- func() {
		foreground = &job{
			group:   group,
			initial: group,
			running: map[int]*task.T{},
			stopped: map[int]*task.T{},
		}

		r <- foreground
	}

	return <-r
}

func (j *job) Append(line string) {
	j.lines = append(j.lines, line)
}

func (j *job) Await(fn func(), p *task.T, ts ...*task.T) {
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

func (j *job) Execute(t *task.T, path string, argv []string, attr *os.ProcAttr) error {
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

func (j *job) Spawn(p, c *task.T, fn func()) {
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

func (j *job) Stopped(t *task.T) {
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

// Bg allows the paused job to continue and returns a reference to its main task.T.
func Bg(w io.Writer, n int) *task.T {
	r := make(chan *task.T)

	requestq <- func() {
		// TODO: Factor out the bits below that are common to this and the func in Fg.
		if len(jobs) == 0 {
			close(r)

			return
		}

		if n == 0 {
			n = jobNumbers()[len(jobs)-1]
		}

		j, found := jobs[n]
		if !found {
			close(r)

			return
		}

		delete(jobs, n)

		for _, line := range j.lines {
			fmt.Fprintf(w, "%s\n", line)
		}

		for pid, t := range j.stopped {
			process.Continue(pid)
			j.running[pid] = t
		}

		go j.main.Run()

		r <- j.main

		close(r)
	}

	return <-r
}

// Fg replaces the current foreground job with the selected job.
func Fg(w io.Writer, n int) bool {
	r := make(chan bool)

	requestq <- func() {
		// TODO: Factor out the bits below that are common to this and the func in Bg.
		if len(jobs) == 0 {
			close(r)

			return
		}

		if n == 0 {
			n = jobNumbers()[len(jobs)-1]
		}

		j, found := jobs[n]
		if !found {
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

		r <- true

		close(r)
	}

	return <-r
}

// Jobs prints a list of stopped jobs.
func Jobs(w io.Writer) {
	r := make(chan struct{})

	requestq <- func() {
		printJobs(w, 0)

		close(r)
	}

	<-r
}

// Monitor launches the goroutine responsible for monitoring jobs/tasks.
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
	foreground *job

	requestq chan func()
	signalq  chan os.Signal

	active     = map[int]*job{}
	background = []*task.T{}
	children   = map[*task.T][]*task.T{}
	jobs       = map[int]*job{}
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

func (j *job) interrupt() {
	interrupt(j.main)

	for pid := range j.running {
		process.Interrupt(pid)
	}
}

func (j *job) notify(pid int, ws unix.WaitStatus) {
	if ws.Continued() {
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

	if ws.Stopped() {
		if len(j.running) == 1 {
			// The last running process is stopping. Stop the task.
			j.stop()
		}

		j.stopped[pid] = t
		delete(j.running, pid)

		return
	}

	code := int(ws)

	switch {
	case ws.Exited():
		code = ws.ExitStatus()

	case ws.Signaled():
		code += 128

	default:
		return
	}

	delete(j.running, pid)
	delete(active, pid)

	if len(j.running) == 0 && len(j.stopped) == 0 {
		j.group = j.initial
	}

	t.Notify(status.Int(code))
}

func (j *job) resume() {
	resume(j.main)

	for pid := range j.running {
		process.Continue(pid)
	}
}

func (j *job) stop() {
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
