// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/integer"
	"github.com/michaelmacinnis/oh/internal/common/type/sym"
	"github.com/michaelmacinnis/oh/internal/common/validate"
	"github.com/michaelmacinnis/oh/internal/system/process"
)

func umask(args cell.I) cell.I {
	v := validate.Fixed(args, 0, 1)

	nmask := int64(0)
	if len(v) == 1 {
		nmask = integer.Value(v[0])
	}

	omask := process.Umask(int(nmask))

	if nmask == 0 {
		process.Umask(omask)
	}

	return sym.New(fmt.Sprintf("0o%o", omask))
}
