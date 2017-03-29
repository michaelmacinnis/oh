// Released under an MIT license. See LICENSE.

package ui

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

type cli struct {
	*liner.State
}

var (
	cooked   liner.ModeApplier
	uncooked liner.ModeApplier
	zero     cli
)

func New(args []string) cell.Interface {
	if len(args) > 1 {
		return &zero
	}

	// We assume the terminal starts in cooked mode.
	cooked, _ = liner.TerminalMode()
	if cooked == nil {
		return &zero
	}

	i := &cli{liner.NewLiner()}

	if hpath, err := system.GetHistoryFilePath(); err == nil {
		if f, err := os.Open(hpath); err == nil {
			i.ReadHistory(f)
			f.Close()
		}
	}

	uncooked, _ = liner.TerminalMode()

	i.SetCtrlCAborts(true)
	i.SetTabCompletionStyle(liner.TabPrints)
	i.SetShouldRestart(system.ResetForegroundGroup)
	i.SetWordCompleter(complete)

	return i
}

func (i *cli) Close() error {
	if i.Exists() {
		if hpath, err := system.GetHistoryFilePath(); err == nil {
			if f, err := os.Create(hpath); err == nil {
				i.WriteHistory(f)
				f.Close()
			} else {
				println("Error writing history: " + err.Error())
			}
		}
	}
	return i.State.Close()
}

func (i *cli) Exists() bool {
	return i != &zero
}

func (i *cli) ReadString(delim byte) (line string, err error) {
	system.SetForegroundGroup(system.Pgid())

	uncooked.ApplyMode()
	defer cooked.ApplyMode()

	command := cell.List(
		cell.Cons(
			cell.NewSymbol("_sys_"),
			cell.NewSymbol("get-prompt"),
		),
		cell.NewSymbol("> "),
	)
	prompt := task.Call(nil, command)

	if line, err = i.State.Prompt(prompt); err == nil {
		i.AppendHistory(line)
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
	first, state, completing := task.GlobalParser().State(line[:pos])

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
		ref, _ := task.Resolve(ft.Lexical, ft.Frame, "$PWD")
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
