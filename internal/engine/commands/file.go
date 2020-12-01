// Released under an MIT license. See LICENSE.

package commands

import (
	"os"
	"strings"

	"github.com/michaelmacinnis/oh/internal/adapted"
	"github.com/michaelmacinnis/oh/internal/common"
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/type/boolean"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/pipe"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
)

func device(m os.FileMode) bool {
	return m&(os.ModeDevice|os.ModeCharDevice) > 0
}

func exists(args cell.I) cell.I {
	count := 0
	ignore := false

	for ; args != pair.Null; args = pair.Cdr(args) {
		path := literal.String(pair.Car(args))
		if path == "-i" {
			ignore = true

			continue
		}

		count++

		s, err := os.Stat(path)
		if err != nil {
			return boolean.False
		}

		if ignore && device(s.Mode()) {
			// Report device files as not existing.
			// So that redirections to /dev/null etc. work.
			return boolean.False
		}
	}

	return boolean.Bool(count > 0)
}

func open(args cell.I) cell.I {
	mode := common.String(pair.Car(args))
	path := common.String(pair.Cadr(args))
	flags := 0

	if !strings.ContainsAny(mode, "-") {
		flags = os.O_CREATE
	}

	read := false
	if strings.ContainsAny(mode, "r") {
		read = true
	}

	write := false
	if strings.ContainsAny(mode, "w") {
		write = true

		if !strings.ContainsAny(mode, "a") {
			flags |= os.O_TRUNC
		}
	}

	if strings.ContainsAny(mode, "a") {
		write = true
		flags |= os.O_APPEND
	}

	if read == write {
		read = true
		write = true
		flags |= os.O_RDWR
	} else if write {
		flags |= os.O_WRONLY
	}

	f, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		panic(err.Error())
	}

	r := f
	if !read {
		r = nil
	}

	w := f
	if !write {
		w = nil
	}

	return pipe.New(r, w)
}

func tempfifo(args cell.I) cell.I {
	validate.Fixed(args, 0, 0)

	name, err := adapted.TempFifo("fifo-")
	if err != nil {
		panic(err.Error())
	}

	return sym.New(name)
}
