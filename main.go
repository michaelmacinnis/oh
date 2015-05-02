/*
Oh is a Unix shell.  It is similar in spirit but different in detail from
other Unix shells. The following commands behave as expected:

    date
    cat /usr/share/dict/words
    who >user.names
    who >>user.names
    wc <file
    echo [a-f]*.c
    who | wc
    who; date
    cc *.c &
    mkdir junk && cd junk
    cd ..
    rm -r junk || echo "rm failed!"

For more detail, see: https://github.com/michaelmacinnis/oh

Oh is released under an MIT-style license.
*/
package main

import (
	"fmt"
	"os"
	"sort"
)

var (
	done0 chan Cell
	eval0 chan Cell
	jobs  = map[int]*Task{}
)

func broker() {
	irq := Incoming()

	LaunchForegroundTask()

	var c Cell
	for c == nil && ForegroundTask().Stack != Null {
		for c == nil {
			select {
			case <-irq:
			case c = <-eval0:
			}
		}
		ForegroundTask().Eval <- c
		for c != nil {
			prev := ForegroundTask()
			select {
			case sig := <-irq:
				// Handle signals.
				switch sig {
				case StopRequest:
					ForegroundTask().Suspend()
					last := 0
					for k := range jobs {
						if k > last {
							last = k
						}
					}
					last++

					jobs[last] = ForegroundTask()

					fallthrough
				case InterruptRequest:
					if sig == InterruptRequest {
						ForegroundTask().Stop()
					}
					fmt.Printf("\n")

					LaunchForegroundTask()
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

func evaluate(c Cell) {
	eval0 <- c
	<-done0

	task := ForegroundTask()
	task.Job.command = ""
	task.Job.group = 0
}

func init() {
	done0 = make(chan Cell)
	eval0 = make(chan Cell)
}

func main() {
	go broker()

	DefineBuiltin("fg", func(t *Task, args Cell) bool {
		if !JobControlEnabled() || t != ForegroundTask() {
			return false
		}

		index := 0
		if args != Null {
			if a, ok := Car(args).(Atom); ok {
				index = int(a.Int())
			}
		} else {
			for k := range jobs {
				if k > index {
					index = k
				}
			}
		}

		found, ok := jobs[index]

		if !ok {
			return false
		}

		delete(jobs, index)

		SetForegroundTask(found)

		return true
	})

	DefineBuiltin("jobs", func(t *Task, args Cell) bool {
		if !JobControlEnabled() || t != ForegroundTask() ||
			len(jobs) == 0 {
			return false
		}

		i := make([]int, 0, len(jobs))
		for k := range jobs {
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

	Start(boot, evaluate)
}

//go:generate generators/go.oh
//go:generate go fmt predicates.go

//go:generate scripts/test.oh
//go:generate generators/doc.oh manual MANUAL.md
//go:generate generators/doc.oh readme README.md
