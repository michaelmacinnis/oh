#!/usr/bin/env oh

define label: method () as: return return
define continue: method (label) as: label label

define while2: syntax e (condition: body) as {
	set body: cons 'block body
	define loop: label
	if (not (e::eval condition)): return '()
	e::eval body
	continue loop
}

define count: integer 0
while2 (lt count (integer 100)) {
	set count: add count 1
	echo: "Hello, World! (%03d)"::sprintf count
}

