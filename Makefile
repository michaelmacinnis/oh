grammar.go: grammar.y
	go tool yacc -o grammar.go grammar.y
	sed -i.save -f grammar.sed grammar.go
	go fmt grammar.go
	rm -f y.output grammar.go.save

MANUAL.md:
	doctest/test.oh
	doctest/doc.oh manual > $@

README.md:
	doctest/test.oh
	doctest/doc.oh readme > $@

clean:
	rm -rf main.[0-9] oh

.PHONY: clean MANUAL.md README.md

