// Released under an MIT license. See LICENSE.

// Package boot provides what is necessary for bootstrapping oh.
package boot

import _ "embed"

//go:embed boot.oh
var script string

// Script returns the boot script for oh.
func Script() string { //nolint:funlen
	return script
}
