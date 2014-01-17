parser.go: parser.y
	go tool yacc -o parser.go parser.y
	rm -f y.output

clean:
	rm -rf main.[0-9] oh

.PHONY: clean

