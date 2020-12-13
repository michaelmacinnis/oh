package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/type/list"
	"github.com/michaelmacinnis/oh/internal/common/type/num"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func ListMethods() map[string]func(cell.I, cell.I) cell.I {
	return map[string]func(cell.I, cell.I) cell.I{
		"append":   appendMethod,
		"extend":   extend,
		"get":      get,
		"head":     head,
		"length":   length,
		"reverse":  reverse,
		"set-head": setHead,
		"set-tail": setTail,
		"slice":    slice,
		"tail":     tail,
	}
}

func appendMethod(s cell.I, args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	self := pair.To(s)

	return list.Append(self, v...)
}

func extend(s cell.I, args cell.I) cell.I {
	v := validate.Fixed(args, 1, 1)

	self := pair.To(s)

	return list.Join(self, v[0])
}

func get(s cell.I, args cell.I) cell.I {
	v, args := validate.Variadic(args, 0, 1)

	self := pair.To(s)

	i := int64(0)
	if len(v) != 0 {
		i = integer.Value(v[0])
	}

	var dflt cell.I
	if args != pair.Null {
		dflt = args
	}

	return pair.Car(list.Tail(self, i, dflt))
}

func head(s cell.I, _ cell.I) cell.I {
	return pair.Car(pair.To(s))
}

func length(s cell.I, args cell.I) cell.I {
	validate.Fixed(args, 0, 0)

	return num.Int(int(list.Length(pair.To(s))))
}

func reverse(s cell.I, args cell.I) cell.I {
	validate.Fixed(args, 0, 0)

	return list.Reverse(pair.To(s))
}

func setHead(s cell.I, args cell.I) cell.I {
	v := pair.Car(args)
	pair.SetCar(s, v)

	return v
}

func setTail(s cell.I, args cell.I) cell.I {
	v := pair.Car(args)
	pair.SetCdr(s, v)

	return v
}

func slice(s cell.I, args cell.I) cell.I {
	v := validate.Fixed(args, 1, 2)

	start := integer.Value(v[0])
	end := int64(0)

	if len(v) == 2 { //nolint:gomnd
		end = integer.Value(v[1])
	}

	return list.Slice(s, start, end)
}

func tail(s cell.I, _ cell.I) cell.I {
	return pair.Cdr(pair.To(s))
}
