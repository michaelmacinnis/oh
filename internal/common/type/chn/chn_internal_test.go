// Released under an MIT license. See LICENSE.

package chn

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/common/type/str"
)

func TestWriteRead(t *testing.T) {
	p := New(1).(*chn)

	sent := str.New("hello")

	p.Write(sent)

	received := p.Read()

	if !received.Equal(sent) {
		t.Fail()
	}
}

func TestWriteReadLine(t *testing.T) {
	p := New(1).(*chn)

	sent := str.New("hello")

	p.Write(sent)

	received := p.ReadLine()

	if !received.Equal(sent) {
		t.Fail()
	}
}
