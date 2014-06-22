/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"github.com/peterh/liner"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//#include <signal.h>
//#include <unistd.h>
//void ignore(void) {
//      signal(SIGTTOU, SIG_IGN);
//      signal(SIGTTIN, SIG_IGN);
//}
import "C"

type Liner struct {
	*liner.State
}

func (cli *Liner) ReadString(delim byte) (line string, err error) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(Pgid())))
	raw.ApplyMode()
	defer cooked.ApplyMode()

	if line, err = cli.State.Prompt("> "); err == nil {
		cli.AppendHistory(line)
        	if task0.Job.command == "" {
                	task0.Job.command = line
        	}
		line += "\n"
	}
	return
}

var cli *Liner
var cooked liner.ModeApplier
var interactive bool
var pgid int
var pid int
var raw liner.ModeApplier
var task0 *Task

func complete(line string) []string {
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
	interactive = len(os.Args) <= 1 && C.isatty(C.int(0)) != 0
	if interactive {
		// We assume the terminal starts in cooked mode.
		cooked, _ = liner.TerminalMode()

		cli = &Liner{liner.NewLiner()}

		raw, _ = liner.TerminalMode()

		cli.SetCompleter(complete)
	}
	pid = syscall.Getpid()
	pgid = syscall.Getpgrp()
	if pid != pgid {
		syscall.Setpgid(0, 0)
	}
	pgid = pid

	C.ignore()
}

func ForegroundTask() *Task {
	return task0
}

func Interactive() bool {
	return interactive
}

func Interface() *Liner {
	return cli
}

func NewForegroundTask() *Task {
	if task0 != nil {
		mode, _ := liner.TerminalMode()
		task0.Job.mode = mode
	}
	task0 = NewTask(Cons(nil, Null), nil, nil, nil)
	return task0
}

func Pid() int {
	return pid
}

func Pgid() *int {
	return &pgid
}

func SetForegroundGroup(group int) {
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&group)))
}

func SetForegroundTask(t *Task) {
	task0 = t
}
