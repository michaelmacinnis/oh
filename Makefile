include ${GOROOT}/src/Make.inc

TARGET=oh

all: ${TARGET}

${TARGET}: parser.${O} main.${O} engine.${O} cell.${O}
	${LD} -o ${TARGET} main.${O}

engine.${O}: cell.${O}
main.${O}: parser.${O} engine.${O}

cell.${O} parser.${O}: cell.go parser.go
	${GC} cell.go parser.go

%.${O}: %.go
	${GC} $<

parser.go: parser.y
	${GOBIN}/goyacc -o parser.go parser.y

clean:
	rm -rf parser.go y.output *.${O} ${TARGET}

.PHONY: all clean
