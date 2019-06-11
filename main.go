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
    rm -r junk || echo 'rm failed!'

For more detail, see: https://github.com/michaelmacinnis/oh

Oh is released under an MIT license.
*/

package main

import (
	"flag"
	"fmt"
	"github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/system"
	"github.com/michaelmacinnis/oh/pkg/task"
	"github.com/peterh/liner"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type ui struct {
	*liner.State
}

var (
	cooked            liner.ModeApplier
	parser0           cell.Parser
	pathListSeparator = string(os.PathListSeparator)
	pathSeparator     = string(os.PathSeparator)
	uncooked          liner.ModeApplier
)

func (cli ui) Close() error {
	if hpath, err := system.GetHistoryFilePath(); err == nil {
		if f, err := os.Create(hpath); err == nil {
			cli.WriteHistory(f)
			f.Close()
		} else {
			println("Error writing history: " + err.Error())
		}
	}
	return cli.State.Close()
}

func (cli ui) ReadString(delim byte) (line string, err error) {
	system.SetForegroundGroup(system.Pgid())

	uncooked.ApplyMode()
	defer cooked.ApplyMode()

	command := cell.List(
		cell.Cons(
			cell.NewSymbol("$_sys_"),
			cell.NewSymbol("get-prompt"),
		),
		cell.List(cell.NewSymbol("quote"), cell.NewSymbol("$ ")),
	)
	prompt := cell.Raw(task.Call(command))

	if line, err = cli.State.Prompt(prompt); err == nil {
		cli.AppendHistory(line)
		task.ForegroundTask().Job.SetCommand(line)
		line += "\n"
	}

	if err == liner.ErrPromptAborted {
		return line, cell.ErrCtrlCPressed
	}

	return
}

func clean(s string) string {
	if s == "." || s == pathSeparator+"." {
		return s
	}

	head, tail := split(s)
	if tail == s {
		head, tail = tail, head
	}

	return filepath.Clean(head) + tail
}

func complete(line string, pos int) (head string, completions []string, tail string) {
	first, state, completing := parser0.State(line[:pos])

	head = line[:pos]
	tail = line[pos:]

	defer func() {
		r := recover()
		if r == nil {
			return
		}

		completions = []string{}
	}()

	if state == "SkipWhitespace" {
		return head, []string{"    "}, tail
	}

	if !strings.HasSuffix(head, completing) {
		return head, []string{}, tail
	}

	// Ensure line == prefix + completing + tail
	prefix := head[0 : len(head)-len(completing)]

	ft := task.ForegroundTask()

	cwd := lookup(ft, "PWD")
	home := lookup(ft, "HOME")

	if first == "" {
		completions = files(cwd, home, lookup(ft, "PATH"), completing)
	} else {
		completions = files(cwd, home, lookup(ft, "PWD"), completing)
	}

	completions = append(
		completions,
		ft.Complete(first, completing)...,
	)

	clist := task.Call(cell.List(
		cell.Cons(
			cell.NewSymbol("$_sys_"),
			cell.NewSymbol("get-completions"),
		),
		cell.NewString(first), cell.NewString(completing),
	))

	length := 0
	if cell.IsPair(clist) {
		length = int(cell.Length(clist))
	}
	if length > 0 {
		carray := make([]string, length)
		for i := 0; i < length; i++ {
			carray[i] = cell.Raw(cell.Car(clist))
			clist = cell.Cdr(clist)
		}
		completions = append(completions, carray...)
	}

	if len(completions) == 0 {
		return prefix, []string{completing}, tail
	}

	unique := make(map[string]bool)
	for _, completion := range completions {
		unique[completion] = true
	}

	completions = make([]string, 0, len(unique))
	for completion := range unique {
		completions = append(completions, completion)
	}

	sort.Strings(completions)

	return prefix, completions, tail
}

func directories(s string) []string {
	dirs := []string{}
	for _, dir := range strings.Split(s, pathListSeparator) {
		if dir == "" {
			dir = "."
		} else {
			dir = filepath.Clean(dir)
		}

		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			continue
		}

		dirs = append(dirs, dir)
	}

	return dirs
}

func files(cwd, home, paths, word string) []string {
	candidate := word

	dotdir := false
	prefix := word + "   "
	if prefix[0:2] == "./" || prefix[0:3] == "../" {
		candidate = join(cwd, candidate)
		dotdir = true
	} else if prefix[0:1] == "~" {
		candidate = join(home, candidate[1:])
	} else {
		candidate = clean(candidate)
	}

	candidates := []string{candidate}
	if !path.IsAbs(candidate) && !dotdir {
		candidates = directories(paths)
		for k, v := range candidates {
			candidates[k] = v + pathSeparator + candidate
		}
	}

	completions := []string{}
	for _, candidate := range candidates {
		dirname, basename := filepath.Split(candidate)

		stat, err := os.Stat(dirname)
		if err != nil {
			continue
		} else if len(basename) == 0 && !stat.IsDir() {
			continue
		}

		max := strings.Count(dirname, pathSeparator)

		filepath.Walk(dirname, func(p string, i os.FileInfo, err error) error {
			depth := strings.Count(p, pathSeparator)
			if depth > max {
				if i.IsDir() {
					return filepath.SkipDir
				}
				return nil
			} else if depth < max {
				return nil
			}

			if candidate != pathSeparator && len(basename) == 0 {
				if p == dirname {
					return nil
				}
			} else if !strings.HasPrefix(p, candidate) {
				return nil
			}

			if p != pathSeparator && i.IsDir() {
				p += pathSeparator
			}

			if len(candidate) > len(p) {
				return nil
			}

			s := strings.Index(p, candidate) + len(candidate)
			completion := word + p[s:]
			completions = append(completions, completion)

			return nil
		})
	}

	return completions
}

func join(s ...string) string {
	last := len(s) - 1
	head, tail := split(s[last])
	s[last] = head
	return filepath.Join(s...) + tail
}

func lookup(ft *task.Task, name string) string {
	ref, _ := task.Resolve(ft.Lexical, ft.Frame, name)
	if ref != nil {
		return cell.Raw(ref.Get())
	}
	return ""
}

func split(s string) (head, tail string) {
	head = s
	tail = ""

	index := strings.LastIndex(s, pathSeparator)
	if index > -1 {
		head = s[:index]
		tail = s[index:]
	}

	return
}

func main() {
	defer task.Exit()

	interactive := flag.Bool("i", true, "enable interactive mode")
	command := flag.String("c", "", "parse next argument instead of stdin")

	flag.Parse()

	if *command != "" {
		task.StartNonInteractive(*command, flag.Args())
		return
	}

	if !*interactive && *command == "" {
		fmt.Println("Non interactive session needs either a command (-c) or a file as argument")
		flag.Usage()
		return
	}

	if flag.NArg() > 0 {
		args := flag.Args()
		task.StartFile(filepath.Dir(args[1]), args[1:])
		return
	}

	// We assume the terminal starts in cooked mode.
	cooked, _ = liner.TerminalMode()
	if cooked == nil {
		task.StartFile("", []string{"/dev/stdin"})
		return
	}

	cli := ui{liner.NewLiner()}

	if hpath, err := system.GetHistoryFilePath(); err == nil {
		if f, err := os.Open(hpath); err == nil {
			cli.ReadHistory(f)
			f.Close()
		}
	}

	uncooked, _ = liner.TerminalMode()

	cli.SetCtrlCAborts(true)
	cli.SetTabCompletionStyle(liner.TabPrints)
	cli.SetShouldRestart(system.ResetForegroundGroup)
	cli.SetWordCompleter(complete)

	parser0 = task.MakeParser(cli.ReadString)

	task.StartInteractive(parser0)

	cli.Close()
}

//go:generate bin/test.oh
//go:generate bin/generate.oh
//go:generate bin/doc.oh manual ../doc/manual.md
//go:generate bin/doc.oh readme ../README.md
//go:generate go generate oh/pkg/boot oh/pkg/parser oh/pkg/task
