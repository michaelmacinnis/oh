package main

import (
	"fmt"
	"github.com/peterh/liner"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

type Liner struct {
	*liner.State
}

func (cli *Liner) ReadString(delim byte) (line string, err error) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&group)))
	raw.ApplyMode()
	defer cooked.ApplyMode()

	if line, err = cli.State.Prompt("> "); err == nil {
		cli.AppendHistory(line)
		SetCommand(line)
		line += "\n"
	}
	return
}

var cli *Liner
var cooked liner.ModeApplier
var done0 chan Cell
var eval0 chan Cell
var incoming chan os.Signal
var raw liner.ModeApplier

func broker(pid int) {
	task := ForegroundTask()

	var c Cell = nil
	for c == nil && task.Stack != Null {
		for c == nil {
			select {
			case <-incoming:
				// Ignore signals.
			case c = <-eval0:
			}
		}
		task.Eval <- c
		for c != nil {
			prev := task
			select {
			case sig := <-incoming:
				// Handle signals.
				switch sig {
				case syscall.SIGTSTP:
					if !IsInteractive() {
						syscall.Kill(pid, syscall.SIGSTOP)
						continue
					}
					task.Suspend()
					last := 0
					for k, _ := range jobs {
						if k > last {
							last = k
						}
					}
					last++

					jobs[last] = task

					fallthrough
				case syscall.SIGINT:
					if !IsInteractive() {
						os.Exit(130)
					}
					if sig == syscall.SIGINT {
						task.Stop()
					}
					fmt.Printf("\n")

					task = NewTask0()
					SetForegroundTask(task)
					listen()
					c = nil
				}

			case c = <-task.Done:
				if task != prev {
					c = Null
					continue
				}
			}
		}
		done0 <- c
	}
	os.Exit(status(Car(task.Scratch)))
}

func complete(line string) []string {
	task := ForegroundTask()

	fields := strings.Fields(line)

	if len(fields) == 0 {
		return []string{"    " + line}
	}

	prefix := fields[len(fields)-1]
	if !strings.HasSuffix(line, prefix) {
		return []string{line}
	}

	trimmed := line[0 : len(line)-len(prefix)]

	completions := files(trimmed, prefix)
	completions = append(completions, task.Complete(trimmed, prefix)...)

	if len(completions) == 0 {
		return []string{line}
	}

	return completions
}

func files(line, prefix string) []string {
	task := ForegroundTask()

	completions := []string{}

	prfx := path.Clean(prefix)
	if !path.IsAbs(prfx) {
		ref := Resolve(task.Lexical, task.Dynamic, NewSymbol("$cwd"))
		cwd := ref.Get().String()

		prfx = path.Join(cwd, prfx)
	}

	root, prfx := filepath.Split(prfx)
	if strings.HasSuffix(prefix, "/") {
		root, prfx = path.Join(root, prfx)+"/", ""
	}
	max := strings.Count(root, "/")

	filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		depth := strings.Count(p, "/")
		if depth > max {
			if i.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		} else if depth == max {
			full := path.Join(root, prfx)
			if len(prfx) == 0 {
				full += "/"
			} else if !strings.HasPrefix(p, full) {
				return nil
			}

			completion := line + prefix + p[len(full):]
			completions = append(completions, completion)
		}

		return nil
	})

	return completions
}

func init() {
	done0 = make(chan Cell)
	eval0 = make(chan Cell)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	incoming = make(chan os.Signal, len(signals))
	signal.Notify(incoming, signals...)
}

func listen() *Task {
	task := ForegroundTask()

	go func() {
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
	}()

	return task
}

func Evaluate(c Cell) {
	task := ForegroundTask()

	eval0 <- c
	<-done0

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

func StartBroker(pid int) {
	scope0 = RootScope()
	scope0.DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !IsInteractive() || t != ForegroundTask() {
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

		t.Stop()

		if found.Job.group != 0 {
			foreground := found.Job.group
			syscall.Syscall(syscall.SYS_IOCTL,
				uintptr(syscall.Stdin),
				syscall.TIOCSPGRP,
				uintptr(unsafe.Pointer(&foreground)))
			found.Job.mode.ApplyMode()
		}

		SetForegroundTask(found)
		found.Continue()

		delete(jobs, index)

		return true
	})
	scope0.DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !IsInteractive() || t != ForegroundTask() || len(jobs) == 0 {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k, _ := range jobs {
			i = append(i, k)
		}
		sort.Ints(i)
		for k, v := range i {
			if k != len(jobs)-1 {
				fmt.Printf("[%d] \t%s\n", v, jobs[v].Job.command)
			} else {
				fmt.Printf("[%d]+\t%s\n", v, jobs[v].Job.command)
			}
		}
		return false
	})

	listen()
	go broker(pid)
}

func StartInterface() {
	if IsInteractive() {
		// We assume the terminal starts in cooked mode.
		cooked, _ = liner.TerminalMode()

		cli = &Liner{liner.NewLiner()}

		raw, _ = liner.TerminalMode()

		cli.SetCompleter(complete)

		Parse(cli, Evaluate)

		cli.Close()
		fmt.Printf("\n")
	} else {
		Evaluate(List(NewSymbol("source"), NewString(os.Args[1])))
	}

	os.Exit(0)
}
