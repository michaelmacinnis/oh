# Oh, a new Unix shell

## Why oh?

Oh is a reimagining of the Unix shell as a programming language.

Oh provides:

- Lexical scope;
- Exceptions;
- First-class channels, pipes, environments and functions;
- A list type (no word splitting);
- Rich return values that work with standard shell constructs;
- Kernel-style fexprs (allowing the definition of new language constructs);
- Support for modularity;
- A simplified set of evaluation and quoting rules; and
- A syntax that deviates as little as possible from established conventions;

Oh's goal is a language that is not only more powerful and more regular
but one that works with and within the interactive constraints and
conventions established by the Unix shell over the last half-century.

## Getting started

### Installing

The easiest way to try oh is to download a precompiled binary.


#### DragonFly BSD

- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_dragonfly_amd64)

#### FreeBSD

- [386](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_freebsd_386)
- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_freebsd_amd64)
- [arm](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_freebsd_arm)
- [arm64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_freebsd_arm64)

#### illumos

- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_illumos_amd64)

#### Linux

- [386](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_386)
- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_amd64)
- [arm](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_arm)
- [arm64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_arm64)
- [mips](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_mips)
- [mips64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_mips64)
- [mips64le](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_mips64le)
- [mipsle](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_mipsle)
- [ppc64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_ppc64)
- [ppc64le](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_ppc64le)
- [riscv64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_riscv64)
- [s390x](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_linux_s390x)

#### macOS

- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_darwin_amd64)

#### OpenBSD

- [386](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_openbsd_386)
- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_openbsd_amd64)
- [arm](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_openbsd_arm)
- [arm64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_openbsd_arm64)

#### Solaris

- [amd64](https://github.com/michaelmacinnis/oh/releases/download/v0.7.0/oh_solaris_amd64)

Alternatively, you can build oh from source. With a recent version of
Go installed, type,

    go get github.com/michaelmacinnis/oh

to install oh.

### Configuring

When oh starts it attempts to read a file called `.oh-rc` in the home
directory of the current user. You can override this path by setting
the OH_RC environment variable to the full path of an alternative file
before invoking oh.

The oh rc file is useful for setting environment variables and defining
custom commands. It's also a good place to override oh's default prompt.
The command below replaces oh's default prompt method with one that
displays the current date.

    replace-make-prompt (method (suffix) {
        return `(date)$suffix
    })

Oh (thanks to peterh/liner) also provides a searchable command history.
By default, this history is stored in a file called `.oh-history` in
your home directory. You can override this by setting the OH_HISTORY
environment variable to the full path of an alternative file before
invoking oh.

## Comparing oh to other Unix shells

Oh is a Unix shell. If you've used other Unix shells, oh should feel
familiar. Below are some specific differences you may encounter.

### Clobbering

When redirecting output oh will not overwrite an existing file. To force
oh to overwrite (clobber) an existing file add a pipe, `|`, character
immediately after the redirection operator. For example,

    command >| out.txt

Oh's pipe and redirection syntax is as follows.

| Syntax    | Redirection                        |
|----------:|:----------------------------------:|
|    `<`    | input-from                         |
|    `>`    | output-to                          |
|    `>&`   | output-errors-to                   |
|    `>&\|` | output-errors-clobbers             |
|    `>>`   | append-output-to                   |
|    `>>&`  | append-output-errors-to            |
|    `>\|`  | output-clobbers                    |
|    `\|`   | pipe-output-to                     |
|    `\|&`  | pipe-output-errors-to              |
|    `\|<`  | -named-pipe-input-from<sup>*</sup> |
|    `\|>`  | -named-pipe-output-to<sup>*</sup>  |

\* - Used in process substitution.

### Command substitution

Many Unix shells support command substitution using the historical
backtick syntax,

    `command`

or the POSIX syntax,

    $(command)

Oh has one syntax for command substitution,

    `(command)

This syntax is both nestable and unambiguous.

### Here documents

Oh does not have here documents. It does however allow strings to span
lines and provides a `here` command that takes a string argument and can
be used to the same effect. For example,

    # Build oh for supported BSD platforms
    here "
    dragonfly amd64
    freebsd 386
    freebsd amd64
    freebsd arm
    freebsd arm64
    openbsd 386
    openbsd amd64
    openbsd arm
    openbsd arm64
    " | mill (o a) {
        echo ${o}/${a}
        GOOS=${o} GOARCH=${a} go build -o oh_${o}_${a}
    }

### Variables and implicit concatenation

Like other shells, oh implicitly concatenates adjacent string/symbol
values. Unlike other shells, oh allows a larger set of characters to
appear in variable names. In addition to letters, numbers, and the
underscore character, the following characters,

    '!', '%', '*', '+', '-', '?', '[', ']',  and '^' 

can be used in oh variable names. The command,

    echo $set!

will cause oh to attempt to resolve a variable called `set!`. 
The following characters,

    ',', '.', '/', ':', '=', '@', and '~'

always result in a symbol of one character. This ensures that commands
like,

    cd $PWD/$dir

work as expected. When using implicit concatentation, unexpected behavior
can be avoided by enclosing variable names in braces.

### More detailed comparison

For a detailed comparison to other Unix shells see: [Comparing oh to other Unix Shells](https://htmlpreview.github.io/?https://raw.githubusercontent.com/michaelmacinnis/oh/master/doc/comparison.html)

## Using oh

For more information on using oh, see: [Using oh](doc/manual.md)

## Contributing to oh

Oh is an ongoing experiment and it needs your help. Try oh. Let me know
what works for you and what doesn't.

Pull requests are welcome. For information on contributing, see: [CONTRIBUTING](CONTRIBUTING.md)

You can also sponsor me through GitHub Sponsors or Patreon.

## License

[MIT](LICENSE)

