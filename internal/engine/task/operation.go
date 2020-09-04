// Released under an MIT license. See LICENSE.

package task

import (
	"reflect"
	"runtime"
	"strings"
)

// Op represents a single step of a task.
type Op interface {
	Perform(*T) Op
}

func opString(o Op) string {
	if o == nil {
		return "<nil>"
	}

	if a, ok := o.(Action); ok {
		return funcName(a)
	}

	if r, ok := o.(*registers); ok {
		s := "Restore("
		comma := ""

		if r.code != nil {
			s += "code"
			comma = ", "
		}

		if r.dump != nil {
			s += comma + "dump"
			comma = ", "
		}

		if r.frame != nil {
			s += comma + "frame"
			comma = ", "
		}

		if r.stack != nil {
			s += comma + "dump"
		}

		s += ")"

		return s
	}

	return "<unknown>"
}

// Get the function i's name. Useful for debugging.
func funcName(i interface{}) string {
	n := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	a := strings.Split(n, ".")

	l := len(a)
	if l == 0 {
		return n
	}

	return a[l-1]
}
