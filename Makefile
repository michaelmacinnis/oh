parser.go: parser.y
	go tool yacc -o parser.go parser.y
	sed -i.save -f parser.sed parser.go
	go fmt parser.go
	rm -f y.output parser.go.save

README.md:
	doctest/test.oh
	doctest/doc.oh readme > $@

clean:
	rm -rf main.[0-9] oh

.PHONY: clean README.md

