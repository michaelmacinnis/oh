// Released under an MIT license. See LICENSE.

package pipe

import (
	"testing"

	"github.com/michaelmacinnis/oh/internal/common/type/pair"
	"github.com/michaelmacinnis/oh/internal/common/type/str"
)

func TestWriteRead(t *testing.T) {
	p := New(nil, nil).(*pipe)

	sent := str.New("hello")

	p.Write(sent)

	received := pair.Car(p.Read())

	if !received.Equal(sent) {
		t.Fail()
	}
}
