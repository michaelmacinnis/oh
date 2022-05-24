// Released under an MIT license. See LICENSE.

// Package pipe provides oh's pipe type.
package pipe

import (
	"bufio"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/michaelmacinnis/oh/internal/common"
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
	p *reader.T
	r *os.File
	w *os.File
}

type pipe = T

// New creates a new pipe cell.
func New(r, w *os.File) cell.I {
	if r == nil && w == nil {
		var err error

		r, w, err = os.Pipe()
		if err != nil {
			panic(err.Error())
		}
	}

	p := &pipe{
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
	b := p.buffer()
	if b == nil {
		return pair.Null
	}

	r := p.reader()
	if r == nil {
		return pair.Null
	}

	p.RLock()
	defer p.RUnlock()

	var (
		c   cell.I
		err error
	)

	s, ok := line(b)
	for ok {
		c, err = r.Scan(s)
		if err != nil {
			panic(err.Error())
		}

		if c != nil {
			break
		}

		s, ok = line(b)
	}

	if c == nil {
		return pair.Null
	}

	return c
}

// ReadLine reads a line from the pipe.
func (p *pipe) ReadLine() cell.I {
	b := p.buffer()
	if b == nil {
		return pair.Null
	}

	p.RLock()
	defer p.RUnlock()

	s, ok := line(b)
	if !ok {
		return pair.Null
	}

	return str.New(strings.TrimRight(s, "\n"))
}

// ReaderClose closes the read end of the pipe and sets it to nil.
func (p *pipe) ReaderClose() {
	p.readerClosePipe()
	p.readerPipeNil()
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
		panic(err.Error())
	}

	_, err = p.w.WriteString("\n")
	if err != nil {
		panic(err.Error())
	}
}

// WriteLine writes the string value of a cell to the pipe.
func (p *pipe) WriteLine(c cell.I) {
	// Yes, RLock. This is a write but doesn't change the pipe itself.
	p.RLock()
	defer p.RUnlock()

	if p.w == nil {
		panic("write to closed pipe")
	}

	_, err := p.w.WriteString(common.String(c))
	if err != nil {
		panic(err.Error())
	}

	_, err = p.w.WriteString("\n")
	if err != nil {
		panic(err.Error())
	}
}

// WriterClose closes the write end of the pipe.
func (p *pipe) WriterClose() {
	p.writerClosePipe()
	p.writerPipeNil()
}

func (p *pipe) buffer() *bufio.Reader {
	p.Lock()
	defer p.Unlock()

	if p.r == nil {
		return nil
	}

	if p.b == nil {
		p.b = bufio.NewReader(p.r)
	}

	return p.b
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

func (p *pipe) reader() *reader.T {
	p.Lock()
	defer p.Unlock()

	if p.r == nil {
		return nil
	}

	if p.p == nil {
		p.p = reader.New(p.r.Name())
	}

	return p.p
}

// readerClosePipe closes the read end of the pipe.
func (p *pipe) readerClosePipe() {
	p.RLock()
	defer p.RUnlock()

	if p.p != nil {
		p.p.Close()
	}

	if p.r != nil {
		err := p.r.Close()
		if err != nil {
			panic(err.Error())
		}
	}
}

// readerPipeNil sets the read end of the pipe to nil.
func (p *pipe) readerPipeNil() {
	p.Lock()
	defer p.Unlock()

	p.b = nil
	p.p = nil
	p.r = nil
}

// writerClosePipe closes the write end of the pipe.
func (p *pipe) writerClosePipe() {
	p.RLock()
	defer p.RUnlock()

	if p.w == nil {
		return
	}

	err := p.w.Close()
	if err != nil {
		panic(err.Error())
	}
}

// writerPipeNil sets the write end of the pipe to nil.
func (p *pipe) writerPipeNil() {
	p.Lock()
	defer p.Unlock()

	p.w = nil
}

// R converts c to a pipe and returns the read end of the pipe.
func R(c cell.I) *os.File {
	return To(c).r
}

// W converts c to a pipe and returns the write end of the pipe.
func W(c cell.I) *os.File {
	return To(c).w
}

// Read a line and return it, including the newline.
func line(b *bufio.Reader) (string, bool) {
	s, err := b.ReadString('\n')

	if errors.Is(err, io.EOF) {
		if len(s) > 0 {
			return s, true
		}

		return "", false
	}

	if err != nil {
		panic(err.Error())
	}

	return s, true
}

// A compiler-checked list of interfaces this type satisfies. Never called.
func implements() { //nolint:deadcode,unused
	var t pipe

	// The pipe type is a cell.
	_ = cell.I(&t)

	// The pipe type is a conduit.
	_ = conduit.I(&t)
}
