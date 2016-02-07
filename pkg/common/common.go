// Released under an MIT-style license. See LICENSE.

package common

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}

var (
	ErrNotExecutable = "oh: 126: error/runtime: "
	ErrNotFound      = "oh: 127: error/runtime: "
	ErrSyntax        = "oh: 1: error/syntax: "
)
