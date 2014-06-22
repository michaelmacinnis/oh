package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
)

var done0 chan Cell
var eval0 chan Cell
var incoming chan os.Signal

var jobs = map[int]*Task{}

func broker() {
	pid := Pid()
	var c Cell = nil
	for c == nil && ForegroundTask().Stack != Null {
		for c == nil {
			select {
			case <-incoming:
				// Ignore signals.
			case c = <-eval0:
			}
		}
		ForegroundTask().Eval <- c
		for c != nil {
			prev := ForegroundTask()
			select {
			case sig := <-incoming:
				// Handle signals.
				switch sig {
				case syscall.SIGTSTP:
					if !Interactive() {
						syscall.Kill(pid, syscall.SIGSTOP)
						continue
					}
					ForegroundTask().Suspend()
					last := 0
					for k, _ := range jobs {
						if k > last {
							last = k
						}
					}
					last++

					jobs[last] = ForegroundTask()

					fallthrough
				case syscall.SIGINT:
					if !Interactive() {
						os.Exit(130)
					}
					if sig == syscall.SIGINT {
						ForegroundTask().Stop()
					}
					fmt.Printf("\n")

					go listen(NewForegroundTask())
					c = nil
				}

			case c = <-ForegroundTask().Done:
				if ForegroundTask() != prev {
					c = Null
					continue
				}
			}
		}
		done0 <- c
	}
	os.Exit(status(Car(ForegroundTask().Scratch)))
}

func init() {
	done0 = make(chan Cell)
	eval0 = make(chan Cell)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	incoming = make(chan os.Signal, len(signals))
	signal.Notify(incoming, signals...)
}

func listen(task *Task) {
	for c := range task.Eval {
		saved := *(task.Registers)

		end := Cons(nil, Null)

		SetCar(task.Code, c)
		SetCdr(task.Code, end)

		task.Code = end
		task.NewStates(SaveCode, psEvalCommand)

		task.Code = c
		if !task.Run(end) {
			*(task.Registers) = saved

			SetCar(task.Code, nil)
			SetCdr(task.Code, Null)
		}

		task.Done <- nil
	}
}

func Evaluate(c Cell) {
	eval0 <- c
	<-done0

	task := ForegroundTask()
	task.Job.command = ""
	task.Job.group = 0
}

func InjectSignal(s os.Signal) {
	incoming <- s
}

func SetCommand(command string) {
	task := ForegroundTask()

	if task.Job.command == "" {
		task.Job.command = command
	}
}

func StartBroker() {
	scope0 = RootScope()
	scope0.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !Interactive() || t != ForegroundTask() {
			return false
		}

		index := 0
		if args != Null {
			if a, ok := Car(args).(Atom); ok {
				index = int(a.Int())
			}
		} else {
			for k, _ := range jobs {
				if k > index {
					index = k
				}
			}
		}

		found, ok := jobs[index]

		if !ok {
			return false
		}

		if found.Job.group != 0 {
			SetForegroundGroup(found.Job.group)
			found.Job.mode.ApplyMode()
		}

		delete(jobs, index)

		SetForegroundTask(found)

		t.Stop()
		found.Continue()

		return true
	})
	scope0.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !Interactive() || t != ForegroundTask() || len(jobs) == 0 {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k, _ := range jobs {
			i = append(i, k)
		}
		sort.Ints(i)
		for k, v := range i {
			if k != len(jobs)-1 {
				fmt.Printf("[%d] \t%d\t%s\n", v, jobs[v].Job.group, jobs[v].Job.command)
			} else {
				fmt.Printf("[%d]+\t%d\t%s\n", v, jobs[v].Job.group, jobs[v].Job.command)
			}
		}
		return false
	})

	go listen(NewForegroundTask())
	go broker()
}

func StartInterface() {
	if Interactive() {
		cli := Interface()

		Parse(cli, Evaluate)

		cli.Close()
		fmt.Printf("\n")
	} else if len(os.Args) > 1 {
		Evaluate(List(NewSymbol("source"), NewString(os.Args[1])))
	} else {
		Evaluate(List(NewSymbol("source"), NewString("/dev/stdin")))
	}

	os.Exit(0)
}
