// Released under an MIT-style license. See LICENSE.

package common

type UI interface {
	ReadStringer
	Close() error
	Exists() bool
}

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}
