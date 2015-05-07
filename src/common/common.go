// Released under an MIT-style license. See LICENSE.

package common

type CloseReadStringer interface {
	ReadStringer
	Close() error
}

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}
