package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

// ConduitMethods returns a mapping of names to methods shared by channels and pipes.
func ConduitMethods() map[string]func(cell.I, cell.I) cell.I {
	return map[string]func(cell.I, cell.I) cell.I{
		"close":        close,
		"read":         read,
		"read-line":    readLine,
		"read-list":    readList,
		"reader-close": readerClose,
		"write":        write,
		"write-line":   writeLine,
		"writer-close": writerClose,
	}
}

func close(s cell.I, _ cell.I) cell.I {
	conduit.To(s).Close()

	return pair.Null
}

func read(s cell.I, _ cell.I) cell.I {
	return pair.Car(conduit.To(s).Read())
}

func readerClose(s cell.I, _ cell.I) cell.I {
	conduit.To(s).ReaderClose()

	return pair.Null
}

func readLine(s cell.I, _ cell.I) cell.I {
	return conduit.To(s).ReadLine()
}

func readList(s cell.I, _ cell.I) cell.I {
	return conduit.To(s).Read()
}

func write(s cell.I, args cell.I) cell.I {
	conduit.To(s).Write(args)

	return args
}

func writeLine(s cell.I, args cell.I) cell.I {
	conduit.To(s).WriteLine(args)

	return args
}

func writerClose(s cell.I, _ cell.I) cell.I {
	conduit.To(s).WriterClose()

	return pair.Null
}
