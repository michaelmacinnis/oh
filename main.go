/* Released under an MIT-style license. See LICENSE. */

package main

import (
    "os"
    "github.com/michaelmacinnis/go-tecla"
    "./cell"
    "./engine"
)

func main() {
    engine.Start()

    if len(os.Args) > 1 {
        f, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0666)
        if err == nil {
            cell.ParseFile(f, engine.Evaluate)
        }
    } else {
        cell.Parse(tecla.New("> "), engine.Evaluate)
    }

    os.Exit(engine.Status())
}
