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
	"github.com/michaelmacinnis/oh/src/boot"
	. "github.com/michaelmacinnis/oh/src/cell"
	"github.com/michaelmacinnis/oh/src/parser"
	"github.com/michaelmacinnis/oh/src/task"
	"os"
	"sort"
)

var (
	done0 chan Cell
	eval0 chan Cell
	jobs  = map[int]*task.Task{}
)

func broker() {
	irq := task.Incoming()

	task.LaunchForegroundTask()

	var c Cell
	for c == nil && task.ForegroundTask().Stack != Null {
		for c == nil {
			select {
			case <-irq:
			case c = <-eval0:
			}
		}
		task.ForegroundTask().Eval <- c
		for c != nil {
			prev := task.ForegroundTask()
			select {
			case sig := <-irq:
				// Handle signals.
				switch sig {
				case task.StopRequest:
					task.ForegroundTask().Suspend()
					last := 0
					for k := range jobs {
						if k > last {
							last = k
						}
					}
					last++

					jobs[last] = task.ForegroundTask()

					fallthrough
				case task.InterruptRequest:
					if sig == task.InterruptRequest {
						task.ForegroundTask().Stop()
					}
					fmt.Printf("\n")

					task.LaunchForegroundTask()
					c = nil
				}

			case c = <-task.ForegroundTask().Done:
				if task.ForegroundTask() != prev {
					c = Null
					continue
				}
			}
		}
		done0 <- c
	}
	os.Exit(status(Car(task.ForegroundTask().Scratch)))
}

func evaluate(c Cell) {
	eval0 <- c
	<-done0

	task := task.ForegroundTask()
	task.Job.Command = ""
	task.Job.Group = 0
}

func init() {
	done0 = make(chan Cell)
	eval0 = make(chan Cell)
}

func main() {
	go broker()

	task.DefineBuiltin("fg", func(t *task.Task, args Cell) bool {
		if !task.JobControlEnabled() || t != task.ForegroundTask() {
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

		task.SetForegroundTask(found)

		return true
	})

	task.DefineBuiltin("jobs", func(t *task.Task, args Cell) bool {
		if !task.JobControlEnabled() || t != task.ForegroundTask() ||
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
				fmt.Printf("[%d] \t%d\t%s\n", v, jobs[v].Job.Group, jobs[v].Job.Command)
			} else {
				fmt.Printf("[%d]+\t%d\t%s\n", v, jobs[v].Job.Group, jobs[v].Job.Command)
			}
		}
		return false
	})

	task.Start(boot.Script, evaluate, parser.Parse)
}

func status(c Cell) int {
	a, ok := c.(Atom)
	if !ok {
		return 0
	}
	return int(a.Status())
}

//go:generate bin/test.oh
//go:generate bin/doc.oh manual ../MANUAL.md
//go:generate bin/doc.oh readme ../README.md
