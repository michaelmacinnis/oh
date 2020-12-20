package main

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/engine"
	"github.com/michaelmacinnis/oh/internal/reader"
	"github.com/michaelmacinnis/oh/internal/system/cache"
	"github.com/michaelmacinnis/oh/internal/system/history"
	"github.com/michaelmacinnis/oh/internal/system/job"
	"github.com/michaelmacinnis/oh/internal/system/options"
	"github.com/michaelmacinnis/oh/internal/system/process"
	"github.com/peterh/liner"
)

//nolint:gochecknoglobals
var (
	pathListSeparator = string(os.PathListSeparator)
	pathSeparator     = string(os.PathSeparator)
)

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

func command() bool {
	if options.Command() == "" {
		return false
	}

	r := reader.New(os.Args[0])

	c, err := r.Scan(options.Command() + "\n")
	if err != nil {
		println("problem parsing command:", err.Error())
		os.Exit(1)
	}

	if c == nil {
		println("incomplete command")
		os.Exit(1)
	}

	engine.Evaluate(job.New(process.Group()), c)

	return true
}

func completer(r **reader.T) func(s string, n int) (h string, cs []string, t string) {
	return func(s string, n int) (h string, cs []string, t string) {
		h = s[:n]
		t = s[n:]

		completing := h[strings.LastIndex(h, " ")+1:]

		defer func() {
			r := recover()
			if r == nil {
				return
			}

			cs = []string{}
		}()

		lc := (*r).Lexer().Copy()

		lc.Scan(h)

		lp := (*r).Parser().Copy(func(_ cell.I) {}, lc.Token)

		_ = lp.Parse()

		cs = lc.Expected()
		if len(cs) != 0 {
			return
		}

		// Ensure line == prefix + completing + tail
		prefix := h[0 : len(h)-len(completing)]

		cwd := engine.Resolve("PWD")
		home := engine.Resolve("HOME")

		cmd := lp.Current()
		if cmd == pair.Null {
			if completing == "" {
				cs = []string{"    "}

				return
			}

			cs = files(cache.Executables, cwd, home, engine.Resolve("PATH"), completing)
		} else {
			cs = files(cache.Files, cwd, home, engine.Resolve("PWD"), completing)
		}

		if len(cs) == 0 {
			return prefix, []string{completing}, t
		}

		unique := make(map[string]bool)
		for _, completion := range cs {
			unique[completion] = true
		}

		cs = make([]string, 0, len(unique))
		for completion := range unique {
			cs = append(cs, completion)
		}

		sort.Strings(cs)

		if len(cs) == 1 && !strings.HasSuffix(cs[0], "/") {
			cs[0] = cs[0] + " "
		}

		return prefix, cs, t
	}
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

func expand(cwd, home, candidate string) (string, bool) {
	dotdir := false
	prefix := candidate + "   "

	switch {
	case candidate == "":
		// Leave it as is.

	case prefix[0:2] == "./" || prefix[0:3] == "../":
		candidate = join(cwd, candidate)
		dotdir = true

	case prefix[0:1] == "~":
		candidate = join(home, candidate[1:])

	default:
		candidate = clean(candidate)
	}

	return candidate, dotdir
}

func files(cached func(string) []string, cwd, home, paths, word string) []string {
	candidate, dotdir := expand(cwd, home, word)

	candidates := []string{candidate}
	if !path.IsAbs(candidate) && !dotdir {
		candidates = directories(paths)
		for k, v := range candidates {
			candidates[k] = v + pathSeparator + candidate
		}
	}

	return matches(cached, candidates, word)
}

func interactive() bool {
	if !options.Interactive() {
		return false
	}

	name := options.Args()[0]

	err := process.BecomeForegroundGroup()
	if err != nil {
		println(err.Error())

		return false
	}

	// We assume the terminal starts in cooked mode.
	cooked, err := liner.TerminalMode()
	if err != nil {
		println(err.Error())

		return false
	}

	// Restore terminal state when we exit.
	defer func() {
		err := cooked.ApplyMode()
		if err != nil {
			println(err.Error())
		}
	}()

	cli := liner.NewLiner()

	cli.SetCtrlCAborts(true)

	uncooked, err := liner.TerminalMode()
	if err != nil {
		println(err.Error())

		return false
	}

	err = history.Load(cli.ReadHistory)
	if err != nil {
		println(err.Error())
	}

	defer func() {
		err = history.Save(cli.WriteHistory)
		if err != nil {
			println(err.Error())
		}

		_, _ = os.Stdout.Write([]byte{'\n'})
	}()

	err = repl(cli, cooked, uncooked, name)
	if !errors.Is(err, io.EOF) {
		println(err.Error())
	}

	return true
}

func join(s ...string) string {
	last := len(s) - 1
	head, tail := split(s[last])
	s[last] = head

	return filepath.Join(s...) + tail
}

func matches(cached func(string) []string, candidates []string, word string) []string {
	completions := []string{}

	for _, candidate := range candidates {
		dirname, basename := filepath.Split(candidate)

		if skip(dirname, basename) {
			continue
		}

		for _, p := range cached(dirname) {
			if candidate != pathSeparator && len(basename) == 0 {
				suffix := strings.TrimPrefix(p, dirname)
				if strings.HasPrefix(suffix, ".") {
					continue
				}
			} else if !strings.HasPrefix(p, candidate) {
				continue
			}

			if len(candidate) > len(p) {
				return nil
			}

			s := strings.Index(p, candidate) + len(candidate)
			completion := word + p[s:]
			completions = append(completions, completion)
		}
	}

	return completions
}

func skip(dirname, basename string) bool {
	stat, err := os.Stat(dirname)
	if err != nil {
		return true
	} else if len(basename) == 0 && !stat.IsDir() {
		return true
	}

	return false
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

func repl(cli *liner.State, cooked, uncooked liner.ModeApplier, name string) error {
	j := job.New(0)
	r := reader.New(name)

	cli.SetWordCompleter(completer(&r))
	cli.SetTabCompletionStyle(liner.TabPrints)

	continued := str.New("  ")
	initial := str.New(": ")
	suffix := initial

	for {
		v, _ := engine.System(j, list.New(sym.New("prompt"), suffix))
		p := common.String(v)

		err := uncooked.ApplyMode()
		if err != nil {
			return err
		}

		line, err := cli.Prompt(p)
		if err != nil {
			if errors.Is(err, liner.ErrPromptAborted) {
				r.Close()
				r = reader.New(name)
				suffix = initial

				continue
			} else {
				return err
			}
		}

		err = cooked.ApplyMode()
		if err != nil {
			return err
		}

		if line == "" {
			continue
		}

		cli.AppendHistory(line)
		j.Append(line)

		suffix = continued

		c, err := r.Scan(line + "\n")
		if err != nil {
			println(err.Error())

			r.Close()
			r = reader.New(name)
			suffix = initial

			continue
		}

		if c != nil {
			engine.Evaluate(j, c)

			j = job.New(0)

			process.RestoreForegroundGroup()

			suffix = initial
		}
	}
}

func main() {
	options.Parse()

	engine.Boot(options.Script(), options.Args())

	if !command() && !interactive() {
		println("unexpected error")
	}
}

//go:generate ./oh bin/test.oh
//go:generate ./oh bin/doc.oh manual ../doc/manual.md
//go:generate ./oh bin/doc.oh readme ../README.md
//go:generate go generate github.com/michaelmacinnis/oh/internal/engine/boot
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/chn
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/env
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/num
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/obj
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/pair
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/pipe
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/status
//go:generate go generate github.com/michaelmacinnis/oh/internal/common/type/str
