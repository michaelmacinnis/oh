TARGET=oh
SOURCE=cell.go engine.go parser.go main.go

O=8
GC=go tool 8g
LD=go tool 8l

all: ${TARGET}

${TARGET}: ${SOURCE}
	${GC} -o main.${O} ${SOURCE}
	${LD} -o ${TARGET} main.${O}

parser.go: parser.y
	${GOBIN}/goyacc -o parser.go parser.y

clean:
	rm -rf y.output main.${O} ${TARGET}

.PHONY: all clean
