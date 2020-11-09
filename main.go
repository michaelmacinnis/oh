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
	"github.com/michaelmacinnis/oh/internal/system/history"
	"github.com/michaelmacinnis/oh/internal/system/job"
	"github.com/michaelmacinnis/oh/internal/system/options"
	"github.com/michaelmacinnis/oh/internal/system/process"
	"github.com/peterh/liner"
)

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
    } else if candidate != "" {
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


func command() bool {
	if options.Command() == "" {
		return false
	}

	r := reader.New(options.Args()[0])

	c := r.Scan(options.Command() + "\n")
	if c == nil {
		println("incomplete command:", options.Command)
		os.Exit(1)
	}

	engine.Evaluate(job.New(process.Group()), c)

	return true
}

func interactive() bool {
	if !options.Interactive() {
		return false
	}

	name := options.Args()[0]

	process.BecomeForegroundGroup()

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

func repl(cli *liner.State, cooked, uncooked liner.ModeApplier, name string) error {
	j := job.New(0)
	r := reader.New(name)

	complete := func(s string, n int) (h string, cs []string, t string) {
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

		lc := r.Lexer().Copy()

		lc.Scan(h)

		lp := r.Parser().Copy(func(_ cell.I) {}, lc.Token)

		lp.Parse()

		cs = lc.Expected()
		if len(cs) != 0 {
            return
        }

        // Ensure line == prefix + completing + tail
        prefix := h[0 : len(h)-len(completing)]

        cwd := engine.Resolve("PWD")
        home := engine.Resolve("HOME")

        if lp.Current() == pair.Null {
            if completing == "" {
                cs = []string{"    "}
                return
            }

            cs = files(cwd, home, engine.Resolve("PATH"), completing)
        } else {
            cs = files(cwd, home, engine.Resolve("PWD"), completing)
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

        return prefix, cs, t
	}

	cli.SetWordCompleter(complete)
	cli.SetTabCompletionStyle(liner.TabPrints)

	for {
		p := common.String(engine.System(j, list.New(sym.New("prompt"), str.New("> "))))

		err := uncooked.ApplyMode()
		if err != nil {
			return err
		}

		line, err := cli.Prompt(p)
		for errors.Is(err, liner.ErrPromptAborted) {
			r = reader.New(name)
			line, err = cli.Prompt(p)
		}

		if err != nil {
			return err
		}

		err = cooked.ApplyMode()
		if err != nil {
			return err
		}

		cli.AppendHistory(line)
		j.Append(line)

		c := r.Scan(line + "\n")
		if c != nil {
			engine.Evaluate(j, c)

			j = job.New(0)

			process.RestoreForegroundGroup()
		}
	}
}

func main() {
	options.Parse()

	engine.Boot(options.Args())

	if !command() && !interactive() {
		println("unexpected error")
	}
}
