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

Oh is released under an MIT-style license.
*/
package main

import (
	"github.com/michaelmacinnis/oh/src/parser"
	"github.com/michaelmacinnis/oh/src/task"
	"github.com/michaelmacinnis/oh/src/ui"
	"os"
)

func main() {
	task.Start(parser.Parse, ui.New(os.Args))
}

//go:generate bin/test.oh
//go:generate bin/doc.oh manual ../MANUAL.md
//go:generate bin/doc.oh readme ../README.md
