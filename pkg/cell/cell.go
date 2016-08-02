// Released under an MIT license. See LICENSE.

package cell

func AppendTo(list Cell, elements ...Cell) Cell {
	var pair, prev, start Cell

	index := 0

	start = Null

	if list == nil {
		panic("cannot append to non-existent list")
	}

	if list != Null {
		start = list

		for prev = list; Cdr(prev) != Null; prev = Cdr(prev) {
		}

	} else if len(elements) > 0 {
		start = Cons(elements[index], Null)
		prev = start
		index++
	}

	for ; index < len(elements); index++ {
		pair = Cons(elements[index], Null)
		SetCdr(prev, pair)
		prev = pair
	}

	return start
}

func Car(c Cell) Cell {
	return c.(*Pair).car
}

func Cdr(c Cell) Cell {
	return c.(*Pair).cdr
}

func Caar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).car
}

func Cadr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).car
}

func Cdar(c Cell) Cell {
	return c.(*Pair).car.(*Pair).cdr
}

func Cddr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).cdr
}

func Caddr(c Cell) Cell {
	return c.(*Pair).cdr.(*Pair).cdr.(*Pair).car
}

func IsAtom(c Cell) bool {
	switch c.(type) {
	case Atom:
		return true
	}
	return false
}

func IsNull(c Cell) bool {
	return c == Null
}

func IsSimple(c Cell) bool {
	return IsAtom(c) || IsCons(c)
}

func IsNumber(c Cell) bool {
	switch t := c.(type) {
	case *Symbol:
		return t.isNumeric()
	case Number:
		return true
	}
	return false
}

func IsPair(c Cell) bool {
	switch c.(type) {
	case *Pair:
		return true
	}
	return false
}

func JoinTo(list Cell, elements ...Cell) Cell {
	var pair, prev, start Cell

	start = list

	if list == nil {
		panic("cannot append to non-existent list")
	} else if list == Null {
		panic("cannot destructively modify nil value")
	}

	for ; list != Null; list = Cdr(list) {
		prev = list
	}

	for index := 0; index < len(elements); index++ {
		for list = elements[index]; list != Null; list = Cdr(list) {
			pair = Cons(Car(list), Null)
			SetCdr(prev, pair)
			prev = pair
		}
	}

	return start
}

func Length(list Cell) int64 {
	var length int64

	for ; list != nil && list != Null && IsPair(list); list = Cdr(list) {
		length++
	}

	return length
}

func List(elements ...Cell) Cell {
	if len(elements) <= 0 {
		return Null
	}

	start := Cons(elements[0], Null)
	prev := start

	for index := 1; index < len(elements); index++ {
		pair := Cons(elements[index], Null)
		SetCdr(prev, pair)
		prev = pair
	}

	return start
}

func Raw(c Cell) string {
	if s, ok := c.(*String); ok {
		return s.Raw()
	}

	return c.String()
}

func Reverse(list Cell) Cell {
	reversed := Null

	for ; list != nil && list != Null; list = Cdr(list) {
		reversed = Cons(Car(list), reversed)
	}

	return reversed
}

func SetCar(c, value Cell) {
	c.(*Pair).car = value
}

func SetCdr(c, value Cell) {
	c.(*Pair).cdr = value
}

func Slice(list Cell, start, end int64) Cell {
	length := Length(list)

	if start < 0 {
		start = length + start
	}

	if start < 0 {
		panic("slice starts before first element")
	} else if start >= length {
		panic("slice starts after last element")
	}

	if end <= 0 {
		end = length + end
	}

	if end < 0 {
		panic("slice ends before first element")
	} else if end > length {
		end = length
	}

	end -= start

	if end < 0 {
		panic("end of slice before start")
	} else if end == 0 {
		return Null
	}

	for ; start > 0; start-- {
		list = Cdr(list)
	}

	slice := Cons(Car(list), Null)

	for c := slice; end > 1; end-- {
		list = Cdr(list)
		n := Cons(Car(list), Null)
		SetCdr(c, n)
		c = n
	}

	return slice
}

func Tail(list Cell, index int64, dflt Cell) Cell {
	length := Length(list)

	if index < 0 {
		index = length + index
	}

	msg := ""
	if index < 0 {
		msg = "index before first element"
	} else if index >= length {
		msg = "index after last element"
	}

	if msg != "" {
		if dflt == nil {
			panic(msg)
		} else {
			return dflt
		}
	}

	for ; index > 0; index-- {
		list = Cdr(list)
	}

	return list
}
