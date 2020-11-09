// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

//nolint:gochecknoglobals,gomnd,nlreturn,wsl
package adapted

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

var (
	rand   uint32
	randmu sync.Mutex
)

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

func TempFifo(prefix string) (name string, err error) {
	dir := os.TempDir()

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name = filepath.Join(dir, prefix+nextSuffix())
		err = unix.Mkfifo(name, 0o600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				rand = reseed()
			}
			continue
		}
		break
	}
	return
}
