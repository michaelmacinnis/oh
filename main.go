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
	"github.com/michaelmacinnis/oh/pkg/task"
	"github.com/michaelmacinnis/oh/pkg/ui"
	"os"
)

func main() {
	task.Start(ui.New(os.Args))
}

//go:generate bin/test.oh
//go:generate bin/generate.oh
//go:generate bin/doc.oh manual ../doc/manual.md
//go:generate bin/doc.oh readme ../README.md
//go:generate go generate oh/pkg/boot oh/pkg/parser oh/pkg/task
