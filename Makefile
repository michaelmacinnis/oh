include ${GOROOT}/src/Make.inc

TARGET=oh
SOURCE=cell.go engine.go parser.go main.go

all: ${TARGET}

${TARGET}: ${SOURCE}
	${GC} -o main.${O} ${SOURCE}
	${LD} -o ${TARGET} main.${O}

parser.go: parser.y
	${GOBIN}/goyacc -o parser.go parser.y

clean:
	rm -rf y.output main.${O} ${TARGET}

.PHONY: all clean
