parser.go: parser.y
	go tool yacc -o parser.go parser.y

clean:
	rm -rf y.output main.[0-9] oh

.PHONY: clean

