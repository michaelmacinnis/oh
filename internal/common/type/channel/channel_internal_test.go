// Released under an MIT license. See LICENSE.

package channel

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/common/type/str"
)

func TestWriteRead(t *testing.T) {
	p := New(1).(*channel)

	sent := str.New("hello")

	p.Write(sent)

	received := p.Read()

	if !received.Equal(sent) {
		t.Fail()
	}
}

func TestWriteReadLine(t *testing.T) {
	p := New(1).(*channel)

	sent := str.New("hello")

	p.Write(sent)

	received := p.ReadLine()

	if !received.Equal(sent) {
		t.Fail()
	}
}
