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

func New(group int) *T {
	foreground = &T{
		group:   group,
		initial: group,
		running: map[int]*task.T{},
		stopped: map[int]*task.T{},
	}

	return foreground
}

func (j *T) Append(line string) {
	j.lines = append(j.lines, line)
}

func (j *T) Launch(t *task.T, path string, argv []string, attr *os.ProcAttr) error {
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

func (j *T) Spawn(p, c *task.T, w ...func()) {
	done := make(chan struct{})

	requestq <- func() {
		parent[c] = p
		if p != nil {
			children[p] = append(children[p], c)
		} else {
			j.main = c
		}

		if w != nil {
			if len(w) > 1 {
				panic("more than one next function")
			}
			next = w[0]
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
		if fs, found := waiting[t]; found {
			for _, f := range fs {
				f()
			}
			delete(waiting, t)
		}

		if foreground.main == t {
			next()
		}

		if !t.Completed() {
			if foreground.main == t {
				number := 1
				for n := range jobs {
					if n >= number {
						number = n + 1
					}
				}
				jobs[number] = foreground

				printJobs(os.Stdout, number)
			}
		} else {
			if p, found := parent[t]; found {
				delete(parent, t)

				cs := children[p]

				n := 0
				for i, c := range cs {
					if c == t {
						n = i
						break
					}
				}

				last := len(cs) - 1
				if last >= 0 {
					cs[n] = cs[last]
					cs[last] = nil
					children[p] = cs[:last]
				}
			}
		}
	}
}

func (j *T) Wait(t *task.T, f func()) {
	requestq <- func() {
		if _, found := parent[t]; !found {
			f()
		} else {
			waiting[t] = append(waiting[t], f)
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

//nolint:gochecknoglobals
var (
	foreground *T
	next       func()

	requestq chan func()
	signalq  chan os.Signal

	active   = map[int]*T{}
	children = map[*task.T][]*task.T{}
	jobs     = map[int]*T{}
	parent   = map[*task.T]*task.T{}
	waiting  = map[*task.T][]func(){}
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

	if status.Exited() {
		code = status.ExitStatus()
	} else if status.Signaled() {
		code += 128
	} else {
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

func init() { //nolint:gochecknoinits
	signal.Ignore(unix.SIGQUIT, unix.SIGTTIN, unix.SIGTTOU)

	signals := []os.Signal{
		unix.SIGCHLD, unix.SIGINT, unix.SIGTSTP,
	}

	requestq = make(chan func(), 1)
	signalq = make(chan os.Signal, len(signals)+1)

	signal.Notify(signalq, signals...)

	go monitor()
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
