package cache

import (
	"os"
	"path/filepath"
	"strings"
)

// Check ensures that all executables in path's directory are cached.
func Check(path string) {
	resultq := make(chan bool)

	dirname, basename := filepath.Split(path)
	requestq <- func() {
		for _, p := range executables[dirname] {
			if p == basename {
				resultq <- true

				break
			}
		}
		close(resultq)
	}

	if <-resultq {
		Files(dirname)
	}
}

// Executables returns executables (and directories) in dirname and schedules a rescan or dirname.
func Executables(dirname string) []string {
	resultq := make(chan []string)

	dirname = filepath.Clean(dirname)
	requestq <- func() {
		resultq <- executables[dirname]
		close(resultq)
	}

	res := <-resultq
	if res == nil {
		go Files(dirname)
	}

	return res
}

// Files caches executables (and directories) and returns files (and directories) found in dirname.
func Files(dirname string) []string {
	dirname = filepath.Clean(dirname)

	max := strings.Count(dirname, pathSeparator) + 1

	e := []string{}
	f := []string{}

	done := make(chan struct{})

	requestq <- func() {
		if _, ok := executables[dirname]; !ok {
			delete(executables, dirname)
		}
		close(done)
	}

	<-done

	_ = filepath.Walk(dirname, func(p string, i os.FileInfo, err error) error {
		if p == dirname {
			return nil
		}

		depth := strings.Count(p, pathSeparator)
		if depth > max {
			if i.IsDir() {
				return filepath.SkipDir
			}

			return nil
		} else if depth < max {
			return nil
		}

		switch {
		case p != pathSeparator && i.IsDir():
			p += pathSeparator

			e = append(e, p)
			f = append(f, p)

		case i.Mode()&0111 != 0:
			e = append(e, p)

		default:
			f = append(f, p)
		}

		return nil
	})

	requestq <- func() {
		if len(e) > 0 {
			executables[dirname] = e
		}
	}

	return f
}

// Populate scans each directory in the colon-separated list of dirnames.
func Populate(dirnames string) {
	for _, dirname := range strings.Split(dirnames, pathListSeparator) {
		if dirname == "" {
			dirname = "."
		} else {
			dirname = filepath.Clean(dirname)
		}

		stat, err := os.Stat(dirname)
		if err != nil || !stat.IsDir() {
			continue
		}

		Files(dirname)
	}
}

//nolint:gochecknoglobals
var (
	executables       = map[string][]string{}
	pathListSeparator = string(os.PathListSeparator)
	pathSeparator     = string(os.PathSeparator)
	requestq          chan func()
)

func init() { //nolint:gochecknoinits
	requestq = make(chan func(), 1)

	go service()
}

func service() {
	for {
		(<-requestq)()
	}
}
