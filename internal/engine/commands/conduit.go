package commands

import (
	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
	"github.com/michaelmacinnis/oh/internal/common/interface/conduit"
	"github.com/michaelmacinnis/oh/internal/common/type/pair"
)

func ConduitMethods() map[string]func(cell.I, cell.I) cell.I {
	return map[string]func(cell.I, cell.I) cell.I{
		"_reader_close_": readerClose,
		"_writer_close_": writerClose,
		"close":          close,
		"read":           read,
		"read-line":      readLine,
		"read-list":      readList,
		"write":          write,
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

func writerClose(s cell.I, _ cell.I) cell.I {
	conduit.To(s).WriterClose()
	return pair.Null
}
