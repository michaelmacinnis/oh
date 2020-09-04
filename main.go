package main

import (
	"errors"
	"io"
	"os"

	"github.com/michaelmacinnis/oh/internal/engine"
	"github.com/michaelmacinnis/oh/internal/reader"
	"github.com/michaelmacinnis/oh/internal/system/history"
	"github.com/michaelmacinnis/oh/internal/system/job"
	"github.com/michaelmacinnis/oh/internal/system/options"
	"github.com/michaelmacinnis/oh/internal/system/process"
	"github.com/peterh/liner"
)

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
	cli.SetTabCompletionStyle(liner.TabPrints)
	//cli.SetWordCompleter(complete)

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

	for {
		err := uncooked.ApplyMode()
		if err != nil {
			return err
		}

		line, err := cli.Prompt(": ")
		for errors.Is(err, liner.ErrPromptAborted) {
			r = reader.New(name)
			line, err = cli.Prompt(": ")
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
