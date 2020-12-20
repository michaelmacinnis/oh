package cache

import (
	"os"
	"path/filepath"
	"strings"
)

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

func Executables(dirname string) []string {
	resultq := make(chan []string)

	requestq <- func() {
		resultq <- executables[dirname]
		close(resultq)
	}

	go Files(dirname)

	return <-resultq
}

func Files(dirname string) []string {
	max := strings.Count(dirname, pathSeparator)

	resultq := make(chan []string)

	requestq <- func() {
		e := []string{}
		f := []string{}

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

			if p != pathSeparator && i.IsDir() {
				p += pathSeparator

				e = append(e, p)
				f = append(f, p)
			} else if i.Mode()&0111 != 0 {
				e = append(e, p)
			} else {
				f = append(f, p)
			}

			return nil
		})

		executables[dirname] = e

		resultq <- f
		close(resultq)
	}

	return <-resultq
}

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

var (
	executables       = map[string][]string{}
	pathListSeparator = string(os.PathListSeparator)
	pathSeparator     = string(os.PathSeparator)
	requestq          chan func()
)

func init() {
	requestq = make(chan func(), 1)

	go service()
}

func service() {
	for {
		(<-requestq)()
	}
}
