// Released under an MIT license. See LICENSE.

// Package pipe provides oh's pipe type.
package pipe

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/interface/literal"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
	"github.com/michaelmacinnis/oh/internal/reader"
)

const name = "pipe"

// T (pipe) is oh's pipe conduit type.
type T struct {
	sync.RWMutex
	b *bufio.Reader
	r *os.File
	w *os.File
}

type pipe = T

// New creates a new pipe cell.
func New(r *os.File, w *os.File) *pipe {
	if r == nil && w == nil {
		var err error

		r, w, err = os.Pipe()
		if err != nil {
			panic(err)
		}
	}

	p := &pipe{
		b: bufio.NewReader(r),
		r: r,
		w: w,
	}

	runtime.SetFinalizer(p, (*pipe).Close)

	return p
}

// Close closes both the read and write ends of the pipe.
func (p *pipe) Close() {
	if p.closeableReadEnd() {
		p.ReaderClose()
	}

	if p.closeableWriteEnd() {
		p.WriterClose()
	}
}

// Equal returns true if the cell c is the same pipe and false otherwise.
func (p *pipe) Equal(c cell.I) bool {
	return Is(c) && p == To(c)
}

// Name returns the name of the pipe type.
func (p *pipe) Name() string {
	return name
}

// Read reads a cell from the pipe.
func (p *pipe) Read() cell.I {
	p.RLock()
	defer p.RUnlock()

	if p.r == nil {
		return pair.Null
	}

	r := reader.New(p.r.Name())

	var c cell.I

	s, ok := p.line()
	for ok {
		c = r.Scan(s)
		if c != nil {
			break
		}

		s, ok = p.line()
	}

	if c == nil {
		return pair.Null
	}

	return c
}

// ReadLine reads a line from the pipe.
func (p *pipe) ReadLine() cell.I {
	p.RLock()
	defer p.RUnlock()

	s, _ := p.line()

	return str.New(strings.TrimRight(s, "\n"))
}

// ReaderClose closes the read end of the pipe.
func (p *pipe) ReaderClose() {
	p.Lock()
	defer p.Unlock()

	if p.r == nil {
		return
	}

	err := p.r.Close()
	if err != nil {
		panic(err)
	}

	p.r = nil
}

// Write writes a cell to the pipe.
func (p *pipe) Write(c cell.I) {
	// Yes, RLock. This is a write but doesn't change the pipe itself.
	p.RLock()
	defer p.RUnlock()

	if p.w == nil {
		panic("write to closed pipe")
	}

	_, err := p.w.WriteString(literal.String(c))
	if err != nil {
		panic(err)
	}

	_, err = p.w.WriteString("\n")
	if err != nil {
		panic(err)
	}
}

// WriterClose closes the write end of the pipe.
func (p *pipe) WriterClose() {
	p.Lock()
	defer p.Unlock()

	if p.w == nil {
		return
	}

	err := p.w.Close()
	if err != nil {
		panic(err)
	}

	p.w = nil
}

func (p *pipe) closeableReadEnd() bool {
	p.RLock()
	defer p.RUnlock()

	return p.r != nil && len(p.r.Name()) > 0
}

func (p *pipe) closeableWriteEnd() bool {
	p.RLock()
	defer p.RUnlock()

	return p.w != nil && len(p.w.Name()) > 0
}

// Read a line from the pipe and return it, including the newline.
func (p *pipe) line() (string, bool) {
	s, err := p.b.ReadString('\n')

	if err == io.EOF {
		if len(s) > 0 {
			return s, true
		}

		return "", false
	}

	if err != nil {
		panic(err)
	}

	return s, true
}

// R converts c to a pipe and returns the read end of the pipe.
func R(c cell.I) *os.File {
	return To(c).r
}

// W converts c to a pipe and returns the write end of the pipe.
func W(c cell.I) *os.File {
	return To(c).w
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t pipe

	// The pipe type is a cell.
	_ = cell.I(&t)

	// The pipe type is a conduit.
	_ = conduit.I(&t)
}
