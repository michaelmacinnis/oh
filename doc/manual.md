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
argument to `ls`. If the file does not exist then the shell creates it.
If the file already exists, oh will refuse to clobber the file.
Output may also be appended to a file.

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

To redirect the output of a command to a file that already exists (replacing
the contents of the file), pipe the output of the command to the `clobber`
command.

    ls | clobber file

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
| `[...]` | Matches any one of the characters enclosed. A pair separated by a hyphen, `-`, will match a lexical range of characters. If the first enclosed character is a `^` the match is negated. |

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

Double quoted strings also automatically perform string interpolation.
In a double quoted string, a dollar sign, `$`, followed by a variable name,
optionally enclosed in braces, will be replaced by the variable's value.
If no variable exists with the name specified the dollar sign, optional
opening brace, `{`, variable name, and optional closing brace, `}`, are
not replaced. While the opening and closing braces are not required their
use is encouraged to avoid ambiguity.

## Using oh Programmatically

In addition to providing a command-line interface to Unix and Unix-like
systems, oh is also a programming language.

### Types

#### Symbols

Oh's default data type is the symbol. Like other programming languages,
a symbol in oh, can be one or more alphanumeric characters. Unlike other
programming languages, there is no restriction that a symbol start with,
or even contain, an alphabetic character. Oh also permits the following
characters in symbols: `!`, `$`, `*`, `+`, `,`, `-`, `/`, `=`,`?`, `[`,
`]`, and `_`. Unless a symbol has been used as a variable name, it
evaluates to itself.

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

#### Rationals

All arithmetic operations in oh are performed by first converting all
operands to rational numbers. The result is a rational number which can
be explicitly converted with the `float`, `integer` or `status` commands.
A rational number can be explicitly declared with the `rational` command,

    define r: rational 100/3
    write: is-rational r
    write: float r
    write: integer r
    write: status r

Oh will also try to help by converting symbols that will parse correctly
as a number when used in a context where that would be appropriate. For
example,

    write: add 3.14 2.72 1.41 2.50 4.67

produces the output,

    361/25

#### Booleans

Oh has a boolean type and the boolean values `true` and `false`.
The `boolean` command can be used to create a boolean value. Passing
`boolean` a non-zero number, the zero status, any string (including
the empty string), and any symbol with the exception of the symbol
`false` will result in a value of `true`.

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
    echo "3 is greater than 2 =>": gt 3 2
    echo "3 is less than or equal to 2 =>": le 3 2
    echo "3 is less than 2 =>": lt 3 2
    echo "3 is not equal to 2 =>": ne 3 2

#### Status

Typical Unix commands return 0 on success and non-zero on failure. This
makes sense as there are many ways to fail but typicaly only one way to
succeed. As a result, most Unix shells treat the zero value as true and
non-zero values as false. This is unlike most other mainstream programming
languages. To cause less confusion, oh introduces a status type. The status
type is an integer that evaluates to true only when it has the value 0.

The command,

    if (status 11) {
        echo "A zero status is true"
    } else {
        echo "A non-zero status is false"
    }

produces the output,

    A non-zero status is false

#### Lists

Lists, in oh, are formed by chaining cons cells. A cons cell is a pair
of values, referred to as the head and tail. The tail of each cons cell
is set to the next cons cell in the list. The tail of the final element
is set to empty list, which is written as `()`.

The commands,

    write: cons 1 (cons 2 (cons 3 ()))
    write: cons 1: cons 2: cons 3 ()
    write: list 1 2 3
    write: quote: 1 2 3

are equivalent. (The `quote` command tells oh not to evaluate the
following expression).

The head of a cons cell can be accessed using the `head` method. The
tail of a cons cell can be accessed using the `tail` method. The `cons`
command can be used to construct a new cons cell.

The commands,

    write: (cons 11 12)::head
    write: (cons 11 12)::tail

produce the output,

    11
    12

In addition to `head`, `tail`, and `cons`, oh provides a number of
convenience methods for dealing with lists. The `length` method, as
expected, returns the length of a list. The `slice` method can be used
to select a sublist. Individual elements of a list can be accessed
and modified with the 'get' and 'set' methods, respectively. A list
of indices can be obtained with the `keys` method.

The commands,

    define a: list do re me
    write: a::get 0
    write: a::get -1
    write: a::slice 1 2
    a::set 0 foo
    a::set 1 bar
    a::set 2 baz
    write a
    write: a::length
    write: a::keys

produce the output,

    do
    me
    (re)
    (foo bar baz)
    3
    (0 1 2)

### Control Structures

#### Block

The most basic control structure in oh is the block. A block creates a
new context and evaluates to the value returned by the final command in
the block.

##### Variables

Variables can be created with the `define` command. A variable's value
can be changed with the `set` (or, in the same context, `define`)
command. With the exception of public variables, which are discussed
later, a variable cannot be accessed outsid the context in which it was
defined.

The command,

    block {
        define x = 1
    }
    set x = 3

produces the output,

    oh: error/runtime: 'x' undefined

as the variable x is not accessible outside the context in which it was
defined. (Note: the equal sign, `=`, in both the `define` and `set`
commands, is optional).

#### If

The command,

    if (cd /tmp) {
        echo $PWD
    }

produces the output,

    /tmp

(The current working directory is stored in the variable `$PWD`).

If statements may have an else clause:

    if (cd /non-existent-directory) {
        echo $PWD
    } else {
        echo "Couldn't change the current working directory."
    }

If statements can be chained:

    if (cd /non-existent-directory) {
        echo $PWD
    } else: if (cd /another-non-existent-directory) {
        echo $PWD
    } else {
        echo "Couldn't change the current working directory, again."
    }

#### While

Oh has a fairly standard pre-test loop. The commands,

    define x = 0
    while (lt x 10) {
        write x
        set x: add x 1
    }

produce the output,

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
the current environment or context. The `context` command along with the
`::` operator can be used to evaluate a public variable in an explicit
context. For a variable to be public it must be created with the `public`
command instead of the `define` command.

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

Oh's first-class contexts form the basis for its object system. In fact,
oh's `object` command is really just a convenience wrapper around a `block`
command with a `context` command as the final action.

The previous example can be rewritten as,

    define o: object {
        public x = 1
        define y = 2
    }
    
    echo "public member" o::x
    echo "private member" o::y

#### _root_

All variables in oh belong to a context. These contexts are chained.
The top-level context is called `_root_`.

#### Method

A sequence of actions can be saved with the `method` command.

    define hello: method () = {
        echo "Hello, World!"
    }

Once defined, a method can be called in the same way as other commands.

    hello

Methods can have named parameters.

    define sum3: method (a b c) = {
        add a b c
    }
    write: sum3 1 2 3

Methods may have a self parameter. The name for the self parameter must
appear before the list of arguments.

    define point: method (r s) =: object {
        define x: integer r
        define y: integer s
    
        public get-x: method self () = {
            return self::x
        }
    
        public get-y: method self () = {
            return self::y
        }
    
        public move: method self (a b) = {
            set self::x: add self::x a
            set self::y: add self::y b
        }
    
        public show: method self () = {
            echo self::x self::y
        }
    }
    
    define p: point 0 0
    p::show

Shared behavior can be implemented by defining a method in an outer scope.

The following code,

    public me: method self () =: echo "my name is:" self::name
    
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

An object may redirect a call to another object. The code below,

    define z: object {
        define name = "z"
        public you: method () =: x::me    # Redirection.
    }
    
    z::me
    z::you

produces the output,

    my name is: z
    my name is: x

#### Syntax

Oh can be extended with the `syntax` command. The `syntax` command is
very similar to the `method` command except that the methods it creates
are passed their arguments unevaluated. The `eval` command can be used
to explicitly evaluate arguments. A name may be specified for the calling
environment after the list of arguments. This can then be used to
evaluate arguments in the calling environment.

The example below uses the `syntax` command to define a new `until` command.

    define until: syntax (condition: body) e = {
        set condition: list (symbol "not") condition
        e::eval: list (symbol "while") condition @body
    }
    
    define x = 10
    until (eq x 0) {
        write x
        set x: sub x 1
    }

### Maps

Using oh's map type, it is relatively simple to record the exit status
for each stage in a pipeline. The code below,

    define exit-status: map
    
    define pipe-fitting: method (label cmd: args) = {
        exit-status::set label: cmd @args
    }
    
    pipe-fitting "1st" echo 1 2 3 |
    pipe-fitting "2nd" tr " " "\n" |
    pipe-fitting "3rd" grep 2
    
    echo "1st stage exit status =>": exit-status::get "1st"
    echo "2nd stage exit status =>": exit-status::get "2nd"
    echo "3rd stage exit status =>": exit-status::get "3rd"

produces the output,

    2
    1st stage exit status => true
    2nd stage exit status => 0
    3rd stage exit status => 0

### Channels

Oh exposes channels as first-class values. Channels allow particularly
elegant solutions to some problems, as shown in the prime sieve example
below (adapted from "Newsqueak: A Language for Communicating with Mice").

    define counter: method (n) = {
        while true {
            write: set n: add n 1
        }
    }
    
    define filter: method (base) = {
        while true {
            define n: read
            if (mod n base): write n
        }
    }
    
    define prime-numbers: channel
    
    counter 2 |+ block {
        define in = _stdin_
    
        while true {
            define prime: in::read
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
        define p: read
        set line: ""::join line ("%7.7s"::sprintf p)
        set count: sub count 1
        if (not: mod count 10) {
            echo line
            set line = ""
        }
    } <prime-numbers

### Pipes

Oh exposes pipes as first-class values. Pipes are created implicitly when
running commands as part of a pipeline but they may also be created
explicitly using the `pipe` command. Pipes are useful for communicating,
particularly with external commands, but, due to buffering, they are not
as convenient as channels for synchronization. Compare the example below
to the same example (shown previously) using channels.

    define counter: method (n) = {
        define welcome: pipe
    
        while true {
            write welcome: set n: add n 1
    
            welcome::read
        }
    }
    
    define filter: method (base) = {
        define welcome: pipe
        while true {
            define msg: readlist
    
            define thanks: msg::get 0
            define n: msg::get 1
    
            if (mod n base) {
                    write welcome n
    
                    welcome::read
            }
    
            thanks::write
        }
    }
    
    define prime-numbers: pipe
    
    counter 2 | block {
        define in = _stdin_
    
        while true {
            define msg: in::readlist
    
            define thanks: msg::get 0
            define prime: msg::get 1
    
            write prime
    
            define out: pipe
            block {
                filter prime &
            } <in >out
    
            thanks::write
    
            set in = out
        }
    } >prime-numbers &
    
    define count: integer 100
    printf "The first %d prime numbers" count
    
    define line = ""
    while count {
        define p: read
        set line: ""::join line ("%7.7s"::sprintf p)
        set count: sub count 1
        if (not: mod count 10) {
            echo line
            set line = ""
        }
    } <prime-numbers

