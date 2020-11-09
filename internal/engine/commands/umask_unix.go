// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package commands

import (
	"fmt"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
	"golang.org/x/sys/unix"
)

func umask(args cell.I) cell.I {
	v := validate.Fixed(args, 0, 1)

	nmask := int64(0)
	if len(v) == 1 {
		nmask = integer.Value(v[0])
	}

	omask := unix.Umask(int(nmask))

	if nmask == 0 {
		unix.Umask(omask)
	}

	return sym.New(fmt.Sprintf("0o%o", omask))
}
