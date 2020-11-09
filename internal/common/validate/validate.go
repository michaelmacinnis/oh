// Released under an MIT license. See LICENSE.

package validate

import (
	"fmt"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

func Variadic(actual cell.I, min, max int) ([]cell.I, cell.I) {
	expected := make([]cell.I, 0, max)

	for i := 0; i < max; i++ {
		if actual == pair.Null {
			if i < min {
				s := Count(min, "argument", "s")
				panic(fmt.Sprintf("expected %s, passed %d", s, i))
			}

			break
		}

		expected = append(expected, pair.Car(actual))

		actual = pair.Cdr(actual)
	}

	return expected, actual
}

func Fixed(actual cell.I, min, max int) []cell.I {
	expected, rest := Variadic(actual, min, max)
	if rest != pair.Null {
		s := Count(max, "argument", "s")
		n := int(list.Length(actual))

		panic(fmt.Sprintf("expected %s, passed %d", s, n))
	}

	return expected
}

func Count(n int, label string, p string) string {
	if n == 1 {
		p = ""
	}

	return fmt.Sprintf("%d %s%s", n, label, p)
}
