// Released under an MIT license. See LICENSE.

package commands

import (
	"math/big"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/rational"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func add(args cell.I) cell.I {
	sum := &big.Rat{}

	for args != pair.Null {
		sum.Add(sum, rational.Number(pair.Car(args)))

		args = pair.Cdr(args)
	}

	return num.Rat(sum)
}

func div(args cell.I) cell.I {
	v, args := validate.Variadic(args, 1, 1)

	quotient := &big.Rat{}
	quotient.Set(rational.Number(v[0]))

	for args != pair.Null {
		quotient.Quo(quotient, rational.Number(pair.Car(args)))

		args = pair.Cdr(args)
	}

	return num.Rat(quotient)
}

func mod(args cell.I) cell.I {
	v := validate.Fixed(args, 2, 2)

	remainder := rational.Number(v[0])
	divisor := rational.Number(v[1])

	if !remainder.IsInt() {
		panic("dividend must be an integer")
	}

	if !divisor.IsInt() {
		panic("divisor must be an integer")
	}

	dividend := &big.Int{}
	dividend.Set(remainder.Num())

	dividend.Mod(dividend, divisor.Num())

	remainder = &big.Rat{}
	remainder.SetInt(dividend)

	return num.Rat(remainder)
}

func mul(args cell.I) cell.I {
	v, args := validate.Variadic(args, 1, 1)

	product := &big.Rat{}
	product.Set(rational.Number(v[0]))

	for args != pair.Null {
		product.Mul(product, rational.Number(pair.Car(args)))

		args = pair.Cdr(args)
	}

	return num.Rat(product)
}

func sub(args cell.I) cell.I {
	v, args := validate.Variadic(args, 1, 1)

	difference := &big.Rat{}
	difference.Set(rational.Number(v[0]))

	for args != pair.Null {
		difference.Sub(difference, rational.Number(pair.Car(args)))

		args = pair.Cdr(args)
	}

	return num.Rat(difference)
}
