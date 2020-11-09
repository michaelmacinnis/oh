// Use of code in this package is governed by Go's BSD-style license.

// Package adapted contains functions adapted from Go's standard library.
//nolint:nakedret,nlreturn,wsl
package adapted

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// ErrBadPattern indicates a pattern was malformed.
var ErrBadPattern = errors.New("syntax error in pattern")

// Match reports whether name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-Separator characters
//		'?'         matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func Match(pattern, name string) (matched bool, err error) { //nolint:gocognit
	last := os.PathSeparator

Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
		star, chunk, pattern = scanChunk(pattern)
		if star {
			if last == os.PathSeparator && strings.HasPrefix(name, ".") {
				return false, nil
			}
			if chunk == "" {
				return !strings.Contains(name, string(os.PathSeparator)), nil
			}
		}
		// Look for match at current position.
		t, ok, err := matchChunk(chunk, name, &last)
		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t
			continue
		}
		if err != nil {
			return false, err
		}
		if star {
			// Look for match skipping i+1 bytes.
			// Cannot skip /.
			for i := 0; i < len(name) && name[i] != os.PathSeparator; i++ {
				t, ok, err := matchChunk(chunk, name[i+1:], &last)
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
					name = t
					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return len(name) == 0, nil
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	inrange := false
	var i int
Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			// error check handled in matchChunk: bad pattern.
			if i+1 < len(pattern) {
				i++
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				break Scan
			}
		}
	}
	return star, pattern[0:i], pattern[i:]
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func matchChunk(chunk, s string, last *rune) (rest string, ok bool, err error) { //nolint:funlen,gocognit
	for len(chunk) > 0 {
		if len(s) == 0 {
			return
		}

		r, n := utf8.DecodeRuneInString(s)

		switch chunk[0] {
		case '[':
			if *last == os.PathSeparator && r == '.' {
				return
			}
			// character class
			*last = r
			s = s[n:]
			chunk = chunk[1:]
			// We can't end right after '[', we're expecting at least
			// a closing bracket and possibly a caret.
			if len(chunk) == 0 {
				err = ErrBadPattern
				return
			}
			// possibly negated
			negated := chunk[0] == '^'
			if negated {
				chunk = chunk[1:]
			}
			// parse all ranges
			match := false
			nrange := 0
			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]
					break
				}
				var lo, hi rune
				if lo, chunk, err = getEsc(chunk); err != nil {
					return
				}
				hi = lo
				if chunk[0] == '-' {
					if hi, chunk, err = getEsc(chunk[1:]); err != nil {
						return
					}
				}
				if lo <= r && r <= hi {
					match = true
				}
				nrange++
			}
			if match == negated {
				return
			}

		case '?':
			if s[0] == os.PathSeparator {
				return
			}
			if *last == os.PathSeparator && r == '.' {
				return
			}
			*last = r
			s = s[n:]
			chunk = chunk[1:]

		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				err = ErrBadPattern
				return
			}
			fallthrough

		default:
			if chunk[0] != s[0] {
				return
			}
			*last = rune(s[0])
			s = s[1:]
			chunk = chunk[1:]
		}
	}
	return s, true, nil
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func getEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = ErrBadPattern
		return
	}
	if chunk[0] == '\\' {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = ErrBadPattern
			return
		}
	}
	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = ErrBadPattern
	}
	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = ErrBadPattern
	}
	return
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func Glob(pattern string) (matches []string, err error) {
	if !hasMeta(pattern) {
		if _, err = os.Lstat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}

	dir, file := filepath.Split(pattern)
	dir = cleanGlobPath(dir)

	if !hasMeta(dir) {
		return glob(dir, file, nil)
	}

	// Prevent infinite recursion. See issue 15879.
	if dir == pattern {
		return nil, ErrBadPattern
	}

	var m []string
	m, err = Glob(dir)
	if err != nil {
		return
	}
	for _, d := range m {
		matches, err = glob(d, file, matches)
		if err != nil {
			return
		}
	}
	return
}

// cleanGlobPath prepares path for glob matching.
func cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(os.PathSeparator):
		// do nothing to the path
		return path
	default:
		return path[0 : len(path)-1] // chop off trailing separator
	}
}

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func glob(dir, pattern string, matches []string) (m []string, e error) {
	m = matches
	fi, err := os.Stat(dir)
	if err != nil {
		return // ignore I/O error
	}
	if !fi.IsDir() {
		return // ignore I/O error
	}
	d, err := os.Open(dir)
	if err != nil {
		return // ignore I/O error
	}
	defer d.Close() //nolint:errcheck

	names, _ := d.Readdirnames(-1)
	sort.Strings(names)

	for _, n := range names {
		matched, err := Match(pattern, n)
		if err != nil {
			return m, err
		}
		if matched {
			m = append(m, filepath.Join(dir, n))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func hasMeta(path string) bool {
	return strings.ContainsAny(path, `*?[`)
}
