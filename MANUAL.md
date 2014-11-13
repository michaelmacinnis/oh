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

    write: add 3.14 2.72 1.41 2.50 4.67

produces the output,

    14.44

#### Booleans

Oh has a boolean type and the boolean values `true` and `false`.
The `boolean` command can be used to create a boolean value. Passing
`boolean` a non-zero number, any string (including the empty string),
and any symbol with the exception of the symbol `false` will result
in a value of `true`.

The command,

    write: boolean 0

produces the output,

    false

Oh provides short-circuit `and` and `or` commands as well as the `not`
command.

The commands below,

    echo "short-circuit and =>": and false (echo this will never be evaluated)
    echo "short-circuit or =>": or true (echo this will never be evaluated)
    echo "boolean not =>": not false

produce the output,

    short-circuit and => false
    short-circuit or => true
    boolean not => true

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
languages. To solve this problem, oh introduces a status type. The status
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
the list is set to empty list, which is written as `'()`.

(The single quote before the parentheses is a short hand for the `quote`
command, which tells oh not to evaluate the following expression).

The following commands are equivalent:

    write: cons 1 (cons 2 (cons 3 '()))
    write: cons 1: cons 2: cons 3 '()
    write: list 1 2 3
    write '(1 2 3)

### Control Structures

#### Block

The most basic control structure in oh is the block. A block creates a new
scope.

The command,

    block {
        define x = 1
    }
    set x = 3

produces the output,

    oh: 'x' is not defined

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

    define count = 0
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


