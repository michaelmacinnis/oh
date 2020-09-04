// Released under an MIT license. See LICENSE.

// Package list provides common list operations. A list is not a true type.
// Lists are more of a type by convention. They are composed of cons cells.
package list

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

// Append appends each element in elements to list.
// If list is Null, a new list is created.
// A non-pair value where a pair is expected will cause a panic.
// The list must be non-circular.
func Append(start cell.I, elements ...cell.I) cell.I {
	if start == nil {
		panic("cannot append to non-existent list")
	}

	if len(elements) == 0 {
		return start
	}

	if start == pair.Null {
		start = pair.Cons(elements[0], pair.Null)
		elements = elements[1:]
	}

	var end cell.I
	for list := start; list != pair.Null; list = pair.Cdr(list) {
		end = list
	}

	for _, e := range elements {
		p := pair.Cons(e, pair.Null)
		pair.SetCdr(end, p)
		end = p
	}

	return start
}

// Join extends the first non-nil, non-NULL list in list
// with every element from every list remaining in lists.
// A non-pair where a pair is expected will cause a panic.
// All lists must be non-circular.
func Join(lists ...cell.I) cell.I {
	var end, start cell.I

	// Find the first non-nil, non-Null list, start.
	for len(lists) != 0 {
		start = lists[0]
		lists = lists[1:]

		if start != nil && start != pair.Null {
			break
		}
	}

	if start == nil {
		panic("join must be passed at least one list")
	}

	if start == pair.Null {
		return start
	}

	// Find the end of the list start.
	for list := start; list != pair.Null; list = pair.Cdr(list) {
		end = list
	}

	for _, list := range lists {
		if list == nil {
			continue
		}

		for list != pair.Null {
			p := pair.Cons(pair.Car(list), pair.Null)
			pair.SetCdr(end, p)
			end = p

			list = pair.Cdr(list)
		}
	}

	return start
}

// Length returns the number of elements in list.
// A non-pair value where a pair is expected will cause a panic.
// The list must be non-circular.
func Length(list cell.I) int64 {
	var length int64

	for list != nil && list != pair.Null {
		length++

		list = pair.Cdr(list)
	}

	return length
}

// New creates a new list composed of all of the elements in elements.
func New(elements ...cell.I) cell.I {
	if len(elements) == 0 {
		return pair.Null
	}

	start := pair.Cons(elements[0], pair.Null)
	end := start

	for _, e := range elements[1:] {
		p := pair.Cons(e, pair.Null)
		pair.SetCdr(end, p)
		end = p
	}

	return start
}

// Reverse reverses list.
// A non-pair value where a pair is expected will cause a panic.
// The list must be non-circular.
func Reverse(list cell.I) cell.I {
	reversed := pair.Null

	for list != nil && list != pair.Null {
		reversed = pair.Cons(pair.Car(list), reversed)

		list = pair.Cdr(list)
	}

	return reversed
}

// Slice creates a new list that is a slice of list.
// Start must be non-zero. If start is > length it will be set to length.
// End must be >= -length. If end is > length it will be set to length.
// Negative values of end count backwards from the end of list.
// Invalid start or end values will cause this function to panic.
// A non-pair value where a pair is expected will cause a panic.
// The list must be non-circular.
func Slice(list cell.I, start, end int64) cell.I {
	length := Length(list)

	if start < 0 {
		start = length + start
	}

	if start < 0 {
		panic("slice starts before first element")
	} else if start > length {
		start = length
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
	} else if end == start {
		return pair.Null
	}

	for start > 0 {
		list = pair.Cdr(list)

		start--
	}

	slice := pair.Cons(pair.Car(list), pair.Null)

	for c := slice; end > 1; end-- {
		list = pair.Cdr(list)
		n := pair.Cons(pair.Car(list), pair.Null)
		pair.SetCdr(c, n)
		c = n
	}

	return slice
}

// Tail returns the sublist of list starting at element index.
// Negative values of index count backwards from the end of list.
// If index is out of range and dflt is provided it is returned.
// Otherwise, this function panics.
// A non-pair value where a pair is expected will cause a panic.
// The list must be non-circular.
func Tail(list cell.I, index int64, dflt cell.I) cell.I {
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

	for index > 0 {
		list = pair.Cdr(list)

		index--
	}

	return list
}
