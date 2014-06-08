package main

import (
        "fmt"
        "os"
        "os/signal"
        "syscall"
)

var done0 chan Cell
var eval0 chan Cell
var incoming chan os.Signal
var task0 *Task

func init() {
        signals := []os.Signal{syscall.SIGINT, syscall.SIGTSTP}
        done0 = make(chan Cell)
        eval0 = make(chan Cell)
        incoming = make(chan os.Signal, len(signals))
        signal.Notify(incoming, signals...)
}

func Evaluate(c Cell) {
        eval0 <- c
        <-done0
        task0.Job.command = ""
        task0.Job.group = 0
}

func InjectSignal(s os.Signal) {
	incoming <- s
}

func StartBroker(pid int, env0 *Env, scope0 *Scope) {
	task0 = listen(env0, scope0)
        go func() {

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
        }()
}

