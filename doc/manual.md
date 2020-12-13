# Using oh

## Using oh Interactively

Oh provides a command-line interface to Unix and Unix-like systems.

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

Standard output and standard error may be redirected,

    ls non-existent-filename >&errors

or appended to a file.

    ls errors >>&errors

Standard input may also be redirected.

    wc -l <file

To redirect the output of a command to a file that already exists (replacing
the contents of the file), use the "clobber" redirection.

    ls >| file

### Pipelines and Filters

The standard output of one command may be connected to the standard input
of another command using the pipe operator.

    ls | wc -l

The commands connected in this way constitute a pipeline. The overall
effect is the same as,

    ls >file; wc -l file

except that no file is used. Instead the two commands are connected by a
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
If no variable exists an exception is thrown. While the opening and closing
braces are not required their use is encouraged to avoid ambiguity.

## Using oh Programmatically

In addition to providing a command-line interface to Unix and Unix-like
systems, oh is also a programming language.

### Control Structures

#### While

Oh has a fairly standard pre-test loop. The commands,

    define x 0
    while (lt? $x 10) {
        echo $x
        set x: add $x 1
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


#### Context

In oh, environments are first-class values with public and private halves.
For a variable to be public it must be created with the `export` command
instead of the `define` command.

The commands,

    define o: block {
        export get $resolve
        export x 1
        define y 2
        (method self () = (return $self))
    }
    
    echo "public variable" (o get x)
    echo "private variable" (o get y)

produce the output,

    public variable 1
    error: 'y' not defined

#### Object

In oh, environments are first-class values with public and private halves.
For a variable to be public it must be created with the `export` command
instead of the `define` command. A reference to an environment can be
created with the `object` command.

    define o: object {
        export get $resolve
    
        export x 1
        define y 2
    }
    
    echo "public member" (o get x)
    echo "private member" (o get y)

#### Method

A sequence of actions can be saved with the `method` command.

    define hello: method () {
        echo "Hello, World!"
    }

Once defined, a method can be called in the same way as other commands.

    hello

Methods can have named parameters.

    define sum3: method (a b c) {
        add $a $b $c
    }
    echo (sum3 1 2 3)

Methods may have a self parameter. The name for the self parameter must
appear before the list of arguments.

    define point: method (r s) = (object {
        define x: add 0 $r
        define y: add 0 $s
    
        export get-x: method () {
            return x
        }
    
        export get-y: method () {
            return y
        }
    
        export move: method self (a b) {
            set x: add $x $a
            set y: add $y $b
        }
    
        export show: method () {
            echo $x $y
        }
    })
    
    define p: point 0 0
    p move 1 2
    p show

Shared behavior can be implemented by defining a method in an outer scope
and explicitly pulling that method "up".

The following code,

    export me: method self () {
        echo 'my name is:' (self name)
    }
    
    define x: object {
        export me $me
        export name: method () {
            return 'x'
        }
    }
    
    x me

produces the output,

    my name is: x

An object may redirect a call to another object. The code below,

    define z: object {
        export me $me
        export name: method () {
            return 'z'
        }
        export you: method () {
            x me    # Redirection.
        }
    }
    
    z me
    z you

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

    define until: syntax (condition (body)) e {
        e eval (cons while (cons (list not $condition) $body))
    }
    
    define x 0
    until (eq? 10 $x) {
        echo $x
        set x: add $x 1
    }

### Maps

Using oh's map type, it is relatively simple to record the exit status
for each stage in a pipeline. The code below,

    define exit-status: map
    
    define pipe-fitting: method (label (cmd)) e {
        exit-status set $label (e eval $cmd)
    }
    
    pipe-fitting 1st echo 1 2 3 |
    pipe-fitting 2nd tr ' ' '\n' |
    pipe-fitting 3rd grep 2 |
    pipe-fitting 4th grep 3
    
    echo '1st stage exit status =>' (exit-status get 1st)
    echo '2nd stage exit status =>' (exit-status get 2nd)
    echo '3rd stage exit status =>' (exit-status get 3rd)
    echo '4th stage exit status =>' (exit-status get 4th)

produces the output,

    1st stage exit status => 0
    2nd stage exit status => 0
    3rd stage exit status => 0
    4th stage exit status => 1

### Channels

Oh exposes channels as first-class values. Channels allow particularly
elegant solutions to some problems, as shown in the prime sieve example
below (adapted from "Newsqueak: A Language for Communicating with Mice").

    
    define filter: method (base) {
        mill (n) {
            mod $n $base && write $n
        }
    }
    
    define connector: chan
    
    spawn {
        define n: number 1
        while true {
            write (set n: add $n 1)
        }
    } >$connector
    
    define prime-numbers: chan
    
    while true {
        define prime: connector read
        write $prime
    
        define filtered: chan
        spawn {
            filter $prime
        } <$connector >$filtered
    
        set connector $filtered
    } >$prime-numbers &
    
    
    define count: number 100
    printf "The first %d prime numbers\n" $count
    
    define line ''
    while $count {
        define p: prime-numbers read
    
        set line: mend '' $line (str format "%7.7s" $p)
    
        set count: sub $count 1
        mod $count 10 || block {
            echo $line
            set line ''
        }
    }

