// Released under an MIT license. See LICENSE.

// Package loc provides the type used to track the source of tokens and commands.
// It is also used to keep track of the evaluator's current lexical location.
package loc

import (
	"strconv"
)

// T (loc) is a lexical location.
type T struct {
	Char int    // Character position (column).
	Line int    // Line number (row).
	Name string // Label for the source of this token.
	Text string // The text at this location.
}

type loc = T

func (l *loc) String() string {
	return l.Name + ":" + strconv.Itoa(l.Line) + ":" + strconv.Itoa(l.Char)
}
