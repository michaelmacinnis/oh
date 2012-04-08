/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"github.com/michaelmacinnis/tecla"
	"os"
	"runtime"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	Start(len(os.Args) <= 1)

	if len(os.Args) <= 1 {
		Parse(tecla.New("> "), Evaluate)
	} else {
		f, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0666)
		if err == nil {
			ParseFile(f, Evaluate)
		}
	}

	os.Exit(ExitStatus())
}
