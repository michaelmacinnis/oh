package options

import (
	"os"

	"github.com/docopt/docopt-go"
	"github.com/mattn/go-isatty"
)

//nolint:gochecknoglobals
var (
	args        []string
	command     string
	interactive bool
	monitor     bool
	script      string
	terminal    int
	usage       = `oh

Usage:
  oh [-m] SCRIPT [ARGUMENTS...]
  oh [-m] -c COMMAND [NAME [ARGUMENTS...]]
  oh [-im] [-s [ARGUMENTS...]]
  oh -h
  oh -v

Arguments:
  ARGUMENTS  Positional parameters.          
  SCRIPT     Path to oh script. Also used as the value for $0.
  NAME       Override $0. Otherwise, $0 is set to name used to invoke oh.

Options:
  -c, --command=COMMAND  Run the specified command.
  -m, --monitor          Invert job control mode.
  -i, --interactive      Disable interactive mode.
  -s, --stdin            Read commands from stdin.
  -h, --help             Display this help.
  -v, --version          Print oh version.

If oh's stdin is a TTY, and oh was invoked with no non-option operands or
oh was explicitly directed to evaluate commands from stdin, interactive and
job control features are enabled. Otherwise, these features are disabled.
`
)

func Args() []string {
	return args
}

func Command() string {
	return command
}

func Interactive() bool {
	return interactive
}

func Parse() {
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		// Error in the usage doc. This should never happen.
		panic(err.Error())
	}

	script = ""

	command, _ = opts.String("--command")

	name, _ := opts.String("NAME")
	if name == "" {
		name = os.Args[0]
	}

	path, _ := opts.String("SCRIPT")
	if path != "" {
		command = "source " + path
		name = path
		script = path
	} else if command == "" && isatty.IsTerminal(os.Stdin.Fd()) {
		interactive = true
		monitor = true
		terminal = int(os.Stdin.Fd())
	}

	args, _ = opts["ARGUMENTS"].([]string)
	args = append([]string{name}, args...)

	invertInteractive, _ := opts.Bool("--interactive")
	interactive = interactive != invertInteractive

	invertMonitor, _ := opts.Bool("--monitor")
	monitor = monitor != invertMonitor
}

func Script() string {
	return script
}

func Monitor() bool {
	return monitor
}

func Terminal() int {
	return terminal
}
