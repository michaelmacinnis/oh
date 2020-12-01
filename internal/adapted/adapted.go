// Use of code in this package is governed by Go's BSD-style license.

// Package adapted contains functions adapted from Go's standard library.
//nolint:funlen,gomnd,nakedret,nlreturn,wrapcheck,wsl
package adapted

import (
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ActualBytes converts any escape-sequence in s to the bytes they
// represent and returns the resulting sequence of actual bytes.
func ActualBytes(s string) (string, error) {
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.

	for len(s) > 0 {
		c, multibyte, ss, err := unquote(s)
		if err != nil {
			return "", err
		}

		s = ss

		if c < utf8.RuneSelf || !multibyte {
			buf = append(buf, byte(c))
		} else {
			var runeTmp [utf8.UTFMax]byte
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}

	return string(buf), nil
}

// CanonicalString returns a string in oh's dollar single-quoted format.
func CanonicalString(s string) string {
	buf := make([]byte, 0, 3*len(s)/2) // Try to avoid more allocations.

	buf = append(buf, `$'`...)

	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1

		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}

		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, hex(rune(s[0]>>4)))
			buf = append(buf, hex(rune(s[0])))
			continue
		}

		// Append escaped rune.
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		case '\'':
			buf = append(buf, `\'`...)
		case '\\':
			buf = append(buf, `\\`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, hex(r>>4))
				buf = append(buf, hex(r))
			case r <= '~':
				buf = append(buf, byte(r))
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, hex(r>>uint(s)))
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, hex(r>>uint(s)))
				}
			}
		}
	}

	buf = append(buf, '\'')

	return string(buf)
}

// LookPath finds name in path.
func LookPath(name, path string) (string, bool, error) {
	cnf := "command not found"

	// Only bypass the path if file begins with / or ./ or ../
	prefix := name + "   "
	if prefix[0:1] == "/" || prefix[0:2] == "./" || prefix[0:3] == "../" {
		exe, err := findPath(name)
		if err == nil {
			return name, exe, nil
		}
		return "", false, &pathError{name, err.Error()}
	}
	if path == "" {
		return "", false, &pathError{name, cnf}
	}
	for _, dir := range strings.Split(path, ":") {
		pathname := dir + "/" + name
		if exe, err := findPath(pathname); err == nil {
			return pathname, exe, nil
		}
	}
	return "", false, &pathError{name, cnf}
}

type pathError struct {
	Path string
	Err  string
}

func (e *pathError) Error() string {
	return e.Path + ": " + e.Err
}

func findPath(file string) (bool, error) {
	d, err := os.Stat(file)
	if err != nil {
		return false, err
	}

	m := d.Mode()
	if m.IsDir() {
		return false, nil
	} else if m&0o111 != 0 {
		return true, nil
	}
	return false, os.ErrPermission
}

func hex(n rune) byte {
	return "0123456789abcdef"[n&0xF]
}

func unhex(b byte) (v rune, ok bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return
}

func unquote(s string) (value rune, multibyte bool, tail string, err error) {
	// easy cases
	if len(s) == 0 {
		err = strconv.ErrSyntax
		return
	}
	switch c := s[0]; {
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}

	// hard case: c is backslash
	if len(s) <= 1 {
		err = strconv.ErrSyntax
		return
	}
	c := s[1]
	s = s[2:]

	switch c {
	case 'a':
		value = '\a'
	case 'b':
		value = '\b'
	case 'f':
		value = '\f'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case 'v':
		value = '\v'
	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		if len(s) < n {
			err = strconv.ErrSyntax
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = strconv.ErrSyntax
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		if c == 'x' {
			// single-byte string, possibly not UTF-8
			value = v
			break
		}
		if v > utf8.MaxRune {
			err = strconv.ErrSyntax
			return
		}
		value = v
		multibyte = true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := rune(c) - '0'
		if len(s) < 2 {
			err = strconv.ErrSyntax
			return
		}
		for j := 0; j < 2; j++ { // one digit already; two more
			x := rune(s[j]) - '0'
			if x < 0 || x > 7 {
				err = strconv.ErrSyntax
				return
			}
			v = (v << 3) | x
		}
		s = s[2:]
		if v > 255 {
			err = strconv.ErrSyntax
			return
		}
		value = v
	case '\\':
		value = '\\'
	case '\'', '"':
		value = rune(c)
	default:
		err = strconv.ErrSyntax
		return
	}
	tail = s
	return
}
