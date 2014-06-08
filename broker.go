package main

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

var done0 chan Cell
var eval0 chan Cell
var incoming chan os.Signal
var task0 *Task

func broker(pid int, env0 *Env, scope0 *Scope) {
        var c Cell = nil
        for c == nil && task0.Stack != Null {
                for c == nil {
                        select {
                        case <-incoming:
                                // Ignore signals.
                        case c = <-eval0:
                        }
                }
                task0.Eval <- c
                for c != nil {
                        prev := task0
                        select {
                        case sig := <-incoming:
                                // Handle signals.
                                switch sig {
                                case syscall.SIGTSTP:
                                        if !interactive {
                                                syscall.Kill(pid, syscall.SIGSTOP)
                                                continue
                                        }
                                        task0.Suspend()
                                        last := 0
                                        for k, _ := range jobs {
                                                if k > last {
                                                        last = k
                                                }
                                        }
                                        last++

                                        jobs[last] = task0

                                        fallthrough
                                case syscall.SIGINT:
                                        if !interactive {
                                                os.Exit(130)
                                        }
                                        if sig == syscall.SIGINT {
                                                task0.Stop()
                                        }
                                        fmt.Printf("\n")

                                        task0 = listen(env0, scope0)
                                        c = nil
                                }

                        case c = <-task0.Done:
                                if task0 != prev {
                                        c = Null
                                        continue
                                }
                        }
                }
                done0 <- c
        }
        os.Exit(status(Car(task0.Scratch)))
}

func files(line, prefix string) []string {
	completions := []string{}

	prfx := path.Clean(prefix)
	if !path.IsAbs(prfx) {
	        ref := Resolve(task0.Lexical, task0.Dynamic, NewSymbol("$cwd"))
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
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
	done0 = make(chan Cell)
	eval0 = make(chan Cell)
	incoming = make(chan os.Signal, len(signals))
	signal.Notify(incoming, signals...)
}

func Complete(line string) []string {
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
	completions = append(completions, task0.Complete(trimmed, prefix)...)

	if len(completions) == 0 {
	        return []string{line}
	}

	return completions
}

func Evaluate(c Cell) {
	eval0 <- c
	<-done0
	task0.Job.command = ""
	task0.Job.group = 0
}

func ForegroundTask() *Task {
	return task0
}

func InjectSignal(s os.Signal) {
	incoming <- s
}

func SetCommand(command string) {
	if task0.Job.command == "" {
		task0.Job.command = command
	}
}

func SetForegroundTask(t *Task) {
	task0 = t
	task0.Continue()
}

func StartBroker(pid int, env0 *Env, scope0 *Scope) {
	task0 = listen(env0, scope0)
	go broker(pid, env0, scope0)
}

