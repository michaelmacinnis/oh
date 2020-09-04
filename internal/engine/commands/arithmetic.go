// Released under an MIT license. See LICENSE.

package commands

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

func add(args cell.I) cell.I {
	sum := &big.Rat{}

	for args != pair.Null {
		sum.Add(sum, rational.Number(pair.Car(args)))

		args = pair.Cdr(args)
	}

	return num.Rat(sum)
}
