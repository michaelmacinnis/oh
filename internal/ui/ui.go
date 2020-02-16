// Released under an MIT license. See LICENSE.

// Package ui provides a command-line interface for the oh language.
package ui

import (
	"os"

	"github.com/michaelmacinnis/oh/internal/interface/cell"
	"github.com/michaelmacinnis/oh/internal/reader/lexer"
	"github.com/michaelmacinnis/oh/internal/reader/parser"
	"github.com/michaelmacinnis/oh/internal/reader/token"
	"github.com/peterh/liner"
)

// Evaluator is the interface for things that want to process parsed commands.
type Evaluator interface {
	Evaluate(command cell.T)
	System(command cell.T)
}

// Run launches the UI which sends commands to the Evaluator.
func Run(e Evaluator) {
	cooked, err := liner.TerminalMode()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	cli := liner.NewLiner()

	uncooked, err := liner.TerminalMode()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	/*
	   if hpath, err := system.GetHistoryFilePath(); err == nil {
	           if f, err := os.Open(hpath); err == nil {
	                   cli.ReadHistory(f)
	                   f.Close()
	           }
	   }
	*/

	cli.SetCtrlCAborts(true)
	//cli.SetTabCompletionStyle(liner.TabPrints)
	//cli.SetShouldRestart(system.ResetForegroundGroup)

start:
	restart := false

	l := lexer.New("oh")

	p := parser.New(e.Evaluate, func() *token.T {
		for {
			t := l.Token()
			if t != nil {
				return t
			}

			merr := uncooked.ApplyMode()
			if merr != nil {
				println(merr.Error())
				os.Exit(1)
			}

			line, err := cli.Prompt("$ ")

			merr = cooked.ApplyMode()
			if merr != nil {
				println(merr.Error())
				os.Exit(1)
			}

			switch err {
			case nil:
				cli.AppendHistory(line)
			case liner.ErrPromptAborted:
				restart = true
				return nil
			default:
				os.Stdout.Write([]byte("exit\n"))
				return nil
			}

			l.Scan(line + "\n")
		}
	})

	complete := func(s string, n int) (h string, cs []string, t string) {
		h = s[:n]
		t = s[n:]

		lc := l.Copy()

		lc.Scan(h)

		lp := p.Copy(func(_ cell.T) {}, lc.Token)

		lp.Parse()

		cs = lc.Expected()

		if len(cs) == 0 {
			// TODO: Call executables/files functions.

			if h[n-1] != byte(' ') {
				cs = []string{" "}
			}
		}

		return
	}

	cli.SetWordCompleter(complete)

	p.Parse()

	if restart {
		goto start
	}
}
