package main

import (
	"github.com/michaelmacinnis/oh/internal/engine"
	"github.com/michaelmacinnis/oh/internal/ui"
)

func main() {
	ui.Run(engine.New())
}
