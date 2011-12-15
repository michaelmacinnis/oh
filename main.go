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
	Start()

	if len(os.Args) > 1 {
		f, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0666)
		if err == nil {
			ParseFile(f, Evaluate)
		}
	} else {
		Parse(tecla.New("> "), Evaluate)
	}

	os.Exit(ExitStatus())
}
