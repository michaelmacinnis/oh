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
	"github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/system"
	"github.com/michaelmacinnis/oh/pkg/task"
	"github.com/peterh/liner"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ui struct {
	*liner.State
}

var (
	cooked   liner.ModeApplier
	parser0  cell.Parser
	uncooked liner.ModeApplier
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
		cell.List(cell.NewSymbol("quote"), cell.NewSymbol("> ")),
	)
	prompt := cell.Raw(task.Call(command))

	if line, err = cli.State.Prompt(prompt); err == nil {
		cli.AppendHistory(line)
		if task.ForegroundTask().Job.Command == "" {
			task.ForegroundTask().Job.Command = line
		}
		line += "\n"
	}

	if err == liner.ErrPromptAborted {
		return line, cell.ErrCtrlCPressed
	}

	return
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

	if first == "" {
		completions = executables(completing)
	} else {
		completions = files(completing)
	}

	completions = append(
		completions,
		task.ForegroundTask().Complete(first, completing)...,
	)

	clist := task.Call(cell.List(
		cell.Cons(
			cell.NewSymbol("$_sys_"),
			cell.NewSymbol("get-completions"),
		),
		cell.NewSymbol(first), cell.NewSymbol(completing),
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

	return prefix, completions, tail
}

func executables(word string) []string {
	completions := []string{}

	if strings.Contains(word, string(os.PathSeparator)) {
		return completions
	}

	pathenv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathenv, string(os.PathListSeparator)) {
		if dir == "" {
			dir = "."
		} else {
			dir = path.Clean(dir)
		}

		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			continue
		}

		max := strings.Count(dir, "/") + 1
		filepath.Walk(dir, func(p string, i os.FileInfo, err error) error {
			depth := strings.Count(p, "/")
			if depth > max {
				if i.IsDir() {
					return filepath.SkipDir
				}
				return nil
			} else if depth < max {
				return nil
			}

			_, basename := filepath.Split(p)

			if strings.HasPrefix(basename, word) {
				completions = append(completions, basename)
			}

			return nil
		})
	}

	return completions
}

func files(word string) []string {
	completions := []string{}

	candidate := word
	if candidate[:1] == "~" {
		candidate = filepath.Join(os.Getenv("HOME"), candidate[1:])
	}

	candidate = path.Clean(candidate)
	if !path.IsAbs(candidate) {
		ft := task.ForegroundTask()
		ref, _ := task.Resolve(ft.Lexical, ft.Frame, "PWD")
		cwd := ref.Get().String()

		candidate = path.Join(cwd, candidate)
	}

	dirname, basename := filepath.Split(candidate)
	if candidate != "/" && strings.HasSuffix(word, "/") {
		dirname, basename = path.Join(dirname, basename)+"/", ""
	}

	stat, err := os.Stat(dirname)
	if err != nil {
		return completions
	} else if len(basename) == 0 && !stat.IsDir() {
		return completions
	}

	max := strings.Count(dirname, "/")

	filepath.Walk(dirname, func(p string, i os.FileInfo, err error) error {
		depth := strings.Count(p, "/")
		if depth > max {
			if i.IsDir() {
				return filepath.SkipDir
			}
			return nil
		} else if depth < max {
			return nil
		}

		full := path.Join(dirname, basename)
		if candidate != "/" && len(basename) == 0 {
			if p == dirname {
				return nil
			}
			full += "/"
		} else if !strings.HasPrefix(p, full) {
			return nil
		}

		if p != "/" && i.IsDir() {
			p += "/"
		}

		if len(full) >= len(p) {
			return nil
		}

		completion := word + p[len(full):]
		completions = append(completions, completion)

		return nil
	})

	return completions
}

func main() {
	defer task.Exit()

	if len(os.Args) > 1 {
		task.StartNonInteractive()
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
