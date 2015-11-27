# Using oh

## Using oh Interactively

The oh shell provides a command-line interface to Unix and Unix-like
systems.

(Much of this section shamelessly copied from "An Introduction to the
UNIX Shell")

### Simple Commands

Simple commands consist of one or more words separated by blanks. The first
word is the name of the command to be executed; any remaining words are
passed as arguments to the command. For example,

    ls -l

is a command that prints a list of files in the current directory. The
argument `-l` tells `ls` to print status information, the size and the
creation date for each file.

Multiple commands may be written on the same line separated by a semicolon.

### Input/Output Redirection

Standard input, standard output and standard error are initially connected
to the terminal. Standard output may be sent to a file.

    ls > file

The notation `>file` is interpreted by the shell and is not passed as an
argument to `ls`. If the file does not exist then the shell creates it,
otherwise the original contents of the file are replaced with the output
from ls. Output may also be appended to a file.

    ls >> file

Standard error may be redirected,

    ls -l !>errors

or appended to a file.

    ls -l !>>errors

Standard input may also be redirected.

    wc -l <file

### Pipelines and Filters

The standard output of one command may be connected to the standard input
of another command using the pipe operator.

    ls | wc -l

The commands connected in this way constitute a pipeline. The overall
effect is the same as,

    ls >file; wc -l file

except that no file is used. Instead the two processes are connected by a
pipe and are run in parallel.

A pipeline may consist of more than two commands.

    ls | grep old | wc -l

### File Name Generation

The oh shell provides a mechanism for generating a list of file names that
match a pattern. The patterns are called globs. The glob, `*.go` in the
command,

    ls *.go

generates, as arguments to `ls`, all file names in the current directory
that end in `.go`. The character * is a pattern that will match any string
including the empty string. In general patterns are specified as follows.

| Pattern | Action                                                         |
|:-------:|:---------------------------------------------------------------|
|   `*`   | Matches any sequence of zero or more characters.               |
|   `?`   | Matches any single character.                                  |
| `[...]` | Matches any one of the characters enclosed. A pair separated by a minus will match a lexical range of characters.|

For example,

    ls [a-z]*

matches all names in the current directory beginning with on of the letters
`a` through `z`, while,

    ls ?

matches all names in the current directory that consist of a single
character.

There is one exception to the general rules given for patterns. The
character `.` at the start of a file name must be explicitly matched.

    echo *

will therefore echo all file names not beginning with a `.` in the current
directory, while,

    echo .*

will echo all those file names that begin with `.` as the `.` was explicitly
specified. This avoids inadvertent matching of the names `.` and `..` which
mean the current directory and the parent directory, respectively.

### Quoting

Characters that have a special meaning to the shell, such as `<` and `>`,
are called metacharacters. These characters must be quoted to strip them of
their special meaning.

    echo '?'

will echo a single `?',

while, 

    echo "xx**\"**xx"

will echo,

    xx**"**xx

A double quoted string may not contain an unescaped double quote but may
contain newlines, which are preserved, and escape sequences which are
interpreted. Escape sequences are not interpreted in a single quoted
string. A single quoted string may not contain a single quote as there is
no way to escape it.

    echo "Hello,
    World!"

## Using oh Programmatically

In addition to providing a command-line interface to Unix and Unix-like
systems, oh is also a programming language.

### Types

#### Symbols

Oh's default data type is the symbol. A symbol is one or more alphanumeric
or `!`, `$`, `*`, `+`, `,`, `-`, `/`, `=`, `?`, `[`, `]`, or  `_` 
characters. Unless a symbol has been used as a variable name, it evaluates
to itself.

The command,

    write this-is-a-symbol

produces the output,

    this-is-a-symbol

#### Integers

In oh, things that look like integers are still symbols by default. To
explicitly create an integer, the `integer` command can be used with an
argument that will parse correctly as an integer.

The command,

    write (integer -1)

produces the output,

    -1

In the above example parentheses are used around the `integer` command to
indicate that it should be evaluated and its value used as the argument
to write. Oh has a convenient shorthand when a command is to be evaluated
and the result of this evaluation used as the final argument of another
command:

    write: integer -1

This shorthand is used often to avoid lots of irritating and
superfluous parentheses when constructing more compilcated commands:

    write: is-integer -1
    write: is-symbol -1
    write: is-integer: integer -1
    write: is-symbol: integer -1

Oh will also try to help by converting symbols that will parse correctly
as an integer when used in a context where that would be appropriate. For
example,

    write: add 1 2 3

produces the output,

    6

Note that oh does not have infix arithmetic operators instead the commands
`add`, `sub`, `mul`, `div` and `mod` must be used.

    write: sub 3 2 1
    write: mul 1024 1024 16
    write: div 65536 256
    write: mod 511 256

#### Floats

Just like integers in oh, things that look like floats are still symbols
by default. To explicitly create a float, the `float` command can be used
with an argument that will parse correctly as an float.

The command,

    write: float "3.14"

produces the output,

    3.14

Again, like integers, oh will try to help by converting symbols that will
parse correctly as a float when used in a context where that would be
appropriate. For example,

    write: float: add 3.14 2.72 1.41 2.50 4.67

produces the output,

    14.44

#### Rationals

Without the `float` command in the previous example the command,

    write: add 3.14 2.72 1.41 2.50 4.67

produces the output,

    361/25

All arithmetic operations in oh are performed by first converting all
operands to rational numbers. The result is a rational number which can
be explicitly converted with the `float`, `integer` or `status` commands.

A rational number can be explicitly declared with the `rational` command,

    define r: rational 100/3
    write: is-rational r
    write: float r
    write: integer r
    write: status r

#### Booleans

Oh has a boolean type and the boolean values `true` and `false`.
The `boolean` command can be used to create a boolean value. Passing
`boolean` a non-zero number, any string (including the empty string),
and any symbol with the exception of the symbol `false` will result
in a value of `true`.

The command,

    write: boolean 0

produces the output,

    true

Oh provides short-circuit `and` and `or` commands as well as the `not`
command.

The commands below,

    echo "short-circuit and (with false and something) =>" {
        and false (echo never be evaluated)
    }
    echo "short-circuit or (with true and something) =>" {
        or true (echo never be evaluated)
    }
    echo "not false =>": not false

produce the output,

    short-circuit and (with false and something) => false
    short-circuit or (with true and something) => true
    not false => true

Oh also provides a set of relational operators:

    echo "3 is equal to 2 =>": eq 3 2
    echo "3 is greater than or equal to 2 =>": ge 3 2
    echo "3 is greater than 2 =>": ge 3 2
    echo "3 is less than or equal to 2 =>": le 3 2
    echo "3 is less than 2 =>": lt 3 2
    echo "3 is not equal to 2 =>": ne 3 2

#### Status

Typical Unix commands return 0 on success and non-zero on failure. This
makes sense as there are many ways to fail but typicaly only one way to
succeed. As a result, most Unix shells treat the zero value as true and
non-zero values as false - unlike most other mainstream programming
languages. To cause less confusion, oh introduces a status type. The status
type is an integer that evaluates to true only when it has the value 0.

The command,

    write: boolean: status 11

produces the output,

    false

#### Conses

Because of its Lisp heritage, one of oh's fundamental types is the cons
cell. A cons cell is a pair of values referred to as the head and tail or
(for historical reasons) the car and cdr. The head of a cons cell is
accessed using the `car` command. The tail of a cons cell is accessed using
the `cdr` command. The `cons` command is used to construct a new cons cell.

The commands,

    write: car: cons 11 12
    write: cdr: cons 11 12

produce the output,

    11
    12

Lists, in oh, are formed by chaining cons cells. The cdr of each cons cell
is set to the next cons cell in the list. The cdr of the last element in
the list is set to empty list, which is written as `()`.

The following commands are equivalent:

    write: cons 1 (cons 2 (cons 3 ()))
    write: cons 1: cons 2: cons 3 ()
    write: list 1 2 3
    write: quote: 1 2 3

(The `quote` command tells oh not to evaluate the following expression).

### Control Structures

#### Block

The most basic control structure in oh is the block. A block creates a new
scope and evaluates to the value returned by the final command in the block.

The command,

    block {
        define x = 1
    }
    set x = 3

produces the output,

    oh: error/runtime: 'x' undefined

as the variable x is not accessible outside the scope in which it was
defined.

Variable are created with the `define` command. A variable's value can be
changed with the `set` (or, in the same scope, `define`) command.

#### If

The command,

    if (cd /tmp) {
        echo $cwd
    }

produces the output,

    /tmp

(The current working directory is stored in the variable `$cwd`).

If statements may have an else clause:

    if (cd /non-existent-directory) {
        echo $cwd
    } else {
        echo "Couldn't change the current working directory."
    }

If statements can be chained:

    if (cd /non-existent-directory) {
        echo $cwd
    } else: if (cd /another-non-existent-directory) {
        echo $cwd
    } else {
        echo "Couldn't change the current working directory, again."
    }

#### While

The command,

    define x = 0
    while (lt x 10) {
        write x
        set x: add x 1
    }

produces the output,

    0
    1
    2
    3
    4
    5
    6
    7
    8
    9


### Objects and Methods

#### Context

In oh, environments are first-class values. The command `context` returns
the current environment. The `context` command along with the `::` operator
can be used to evaluate a public variable in an explicit context. For a
variable to be public it must be created with the `public` command instead
of the `define` command.

The commands,

    define o: block {
        public x = 1
        define y = 2
        context
    }
    
    echo "public variable" o::x
    echo "private variable" o::y

produce the output,

    public variable 1
    oh: error/runtime: 'y' undefined

#### Object

Oh's first-class environments form the basis for its object system. In fact,
oh's `object` command is really just a convenience wrapper around a `block`
command with a `context` command as the final action.

The previous example can be rewritten as,

    define o: object {
        public x = 1
        define y = 2
    }
    
    echo "public member" o::x
    echo "private member" o::y

#### $root

All variables in oh belong to an environment. These environments are
chained. The top-level environment is called `$root`.

#### Method

A sequence of actions can be saved with the `method` command.

    define hello: method () as {
        echo "Hello, World!"
    }

Once defined, a method can be called in the same way as other commands.

    hello

Arguments allow a method to be parameterized.

    define sum3: method (a b c) as {
        add a b c
    }
    write: sum3 1 2 3

Methods may have a self parameter. The name for the self parameter must
appear before the list of arguments.

    define point: method (r s) as: object {
        define x: integer r
        define y: integer s
    
        public get-x: method self () as {
            return self::x
        }
    
        public get-y: method self () as {
            return self::y
        }
    
        public move: method self (a b) as {
            set self::x: add self::x a
            set self::y: add self::y b
        }
    
        public show: method self () as {
            echo self::x self::y
        }
    }
    
    define p: point 0 0
    p::show

Shared behavior can be implemented by defining a method in an outer scope.

The following code,

    public me: method self () as: echo "my name is:" self::name
    
    define x: object {
        define name = "x"
    }
    
    x::me

produces the output,

    my name is: x

An object may explicitly delegate behavior, as shown in the following code,

    define y: object {
        define name = "y"
        public me-too = x::me    # Explicit delegation.
    }
    
    y::me
    y::me-too

which produces the output,

    my name is: y
    my name is: y

An object may redirect a call to another object, as shown in the code below,

    define z: object {
        define name = "z"
        public you: method () as: x::me    # Redirection.
    }
    
    z::me
    z::you

which produces the output,

    my name is: z
    my name is: x

### Pipes

Using oh, it is relatively simple to record the exit status for each stage
in a pipeline. The example below,

    define exit-status: object
    
    define pipe-fitting: method (label cmd: args) as {
        exit-status::set-slot label: cmd @args
    }
    
    pipe-fitting "1st" echo 1 2 3 |
    pipe-fitting "2nd" tr " " "\n" |
    pipe-fitting "3rd" grep 2
    
    echo "1st stage exit status =>": exit-status::get-slot "1st"
    echo "2nd stage exit status =>": exit-status::get-slot "2nd"
    echo "3rd stage exit status =>": exit-status::get-slot "3rd"

produces the output,

    2
    1st stage exit status => true
    2nd stage exit status => 0
    3rd stage exit status => 0

### Channels

In addition to pipes, oh exposes channels as first-class values. Channels
allow particularly elegant solutions to some problems, as shown in the prime
sieve example below (adapted from "Newsqueak: A Language for Communicating
with Mice").

    define counter: method (n) as {
        while true {
            write: set n: add n 1
        }
    }
    
    define filter: method (base) as {
        while true {
    	define n: car: read
            if (mod n base): write n
        }
    }
    
    define prime-numbers: channel
    
    counter 2 |+ block {
        define in = $stdin
    
        while true {
            define prime: car: in::read
            write prime
    
            define out: channel
            spawn: filter prime <in >out
    
            set in = out
        }
    } >prime-numbers &
    
    define count: integer 100
    printf "The first %d prime numbers" count
    
    define line = ""
    while count {
        define p: car: read
        set line = line ^ ("%7.7s"::sprintf p)
        set count: sub count 1
        if (not: mod count 10) {
            echo line
    	set line = ""
        }
    } <prime-numbers

