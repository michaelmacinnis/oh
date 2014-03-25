oh
==

Oh is a Unix shell written in Go and released under an MIT-style license.

Oh is properly tail recursive and exposes continuations as first-class
values:

    # Print 1 to 100 using the continuation (return) returned by label.
    define label: method () as: return return
    define continue: method (label) as: label label
    
    define count: integer 0
    define loop: label
    if (lt count (integer 100)) {
        set count: add count 1
        echo: Text::sprintf "Hello, World! (%03d)" count
        continue loop
    }

Oh also exposes environments as first-class values and uses them as the basis
for its prototype-based object system:

    # Create a point prototype.
    define point: object {
	# Private members are created with the define command.
        define x: integer 0
        define y: integer 0

	# Public members are created with the public command.
        public move: method self (a b) as {
            set self::x: add self::x a
            set self::y: add self::y b
        }

        public show: method self () as {
            echo self::x self::y
        }
    }

    # Create a new point by cloning the point prototype:
    define o: point::clone

Oh uses the same syntax for code and data.  This enables oh to be easily
extended:

    # The short-circuit and operator defined with the syntax command.
    define and: syntax e (: lst) as {
        define r False
        while (not: is-null: car lst) {
            set r: e::eval: car lst
            if (not r): return r
            set lst: cdr lst
        }
        return r
    }

Oh is a concurrent language. It exposes pipes, which are implicit in other
shells, as first-class values:

    define p: pipe

    spawn {
	# Save code to create a continuation-based while command. 
        define code '(syntax e (condition: body) as {
            define label: method () as: return return
            define continue: method (label) as: label label

            set body: cons 'block body
            define loop: label
            if (not (e::eval condition)): return '()
            e::eval body
            continue loop
        })

	# Now send this code through the pipe.
        p::write @code
    }

    # Create the new command by evaluating what was sent through the pipe.
    define while2: eval: p::read

    # Now use the new command.
    define count: integer 0
    while2 (lt count (integer 100)) {
        set count: add count 1
        write count
    }

Oh extends the shell's programming language features without sacrificing the
shell's interactive features. The following commands behave as expected:

    date
    cat /usr/share/dict/words
    who >user.names
    who >>user.names
    wc <file
    echo [a-f]*.c
    who | wc
    who; date
    cc *.c &
    mkdir junk && cd junk
    cd ..
    rm -r junk || echo "rm failed!"

To compile and run oh you will need to install the C libraries ncurses and
libtecla. On Ubuntu 12.10, for example, do:

    sudo apt-get install libncurses5-dev
    sudo apt-get install libtecla1-dev

Then go get oh:

    go get github.com/michaelmacinnis/oh


Background
==========

Despite multiple attempts to improve the Unix shell, its essential
character has remained largely unchanged since the release of the
Bourne shell.

If you squint hard enough, the Unix shell and Lisp look very similar.
So similar that others have combined the two by embedding the Unix
shell in Lisp. The problem with embedding the Unix shell in Lisp, or
another existing language is that the result is either "an uglier, and
confusing, language" or a language that is more combersome when used
as an interactive shell.([13](#13))

Attempts to improve the Unix shell without embedding it in an existing
language have been more successful. These shells share a strong family
resemblance, retaining the look and feel established by the Bourne
shell. The most successful shells in this family (bash, ksh, zsh) are
actually backward compatible with the Bourne shell. Unfortunately, this
backward compatibility results in shells are "inconsistent and confusing
command languages."([10](#10))

Like csh, es, fish and rc, oh asks the question, "What would a shell
look like if it retained the look and feel of the Unix shell but
without being concerned about strict backward compatibility?" Unlike
csh, es, fish and rc, oh makes substantial improvements to the
programming language features of the Unix shell. Oh combines the Unix
shell with the Scheme dialect of Lisp but, rather than attempting to
embed a Unix shell in scheme, oh was designed from scratch and
incorporates features from both languages.


References
==========

Shell History
-------------

<a name="1">1. [Shell History](http://www.in-ulm.de/~mascheck/bourne/n.u-w.mashey.html)</a>

<a name="2">2. [The Thompson Shell](http://v6shell.org/)</a>

<a name="3">3. [The Bourne Shell](http://partmaps.org/era/unix/shell.html)</a>

Attempts to Improve the Unix Shell
----------------------------------

<a name="4">1. [A High-Level Programming and Command Language](http://www.researchgate.net/publication/234805805_A_high-level_programming_and_command_language/file/60b7d51645d5d1022a.pdf)</a> 

<p name="5">2. Chris S. McDonald. fsh - A Functional UNIX Command Interpreter. Software - Practice & Experience 17(10): 685-700, 1987</p>

Embedding the Unix Shell in an Existing Language
------------------------------------------------

<p name="6">6. J. R. Ellis. A Lisp Shell. SIGPLAN Notices, 15(5):24-34, 1980</p>

<a name="7">7. [The Scheme Shell](http://scsh.net/)</a>

<p name="8">8. L. M. Campbell and M. D. Campbell. An Overview of the Ada Shell. In USENIX Winter: 302-313, 1986</p>

<a name="9">9. [Hell: A Haskell Shell](https://github.com/chrisdone/hell)</a>

<a name="10">10. [Using ML as a Command Language](http://www.hpdc.syr.edu/~chapin/papers/pdf/MLShell.pdf)</a>

<a name="11">11. [Zoidberg - A Modular Perl Shell](https://github.com/jberger/Zoidberg)</a>

<a name="12">12. [The Perl Shell](https://github.com/gnp/psh)</a>

<a name="13">13. [Pysh: A Python Shell](http://pysh.sourceforge.net/)</a>

<a name="14">14. [Rush](https://github.com/adamwiggins/rush)</a>

The Unix Shell Family
---------------------

<a name="15">15. [Bash](http://www.gnu.org/software/bash/bash.html)</a>

<a name="16">16. [An Introduction to the C shell](http://www.kitebird.com/csh-tcsh-book/csh-intro.pdf)</a>

<a name="17">17. [Es: A shell with higher-order functions](http://stuff.mit.edu/afs/sipb/user/yandros/doc/es-usenix-winter93.html)</a>

<a name="18">18. [The Fish Shell](http://fishshell.com/)</a>

<a name="19">19. [The Korn Shell](http://www.kornshell.com/)</a>

<a name="20">20. [Rc - The Plan 9 Shell](http://plan9.bell-labs.com/sys/doc/rc.html)</a>

<a name="21">21. [Tcsh](http://www.tcsh.org/Welcome)</a>

<a name="22">22. [Zsh](http://www.zsh.org/)</a>

Fexprs
------

<a name="23">23. [Fexprs as the Basis of Lisp Function Application or $vau : The Ultimate Abstraction](https://www.wpi.edu/Pubs/ETD/Available/etd-090110-124904/unrestricted/jshutt.pdf)</a>

First-class Environments
------------------------

<a name="24">24. [First-class environments. Discuss.  ;)](http://lambda-the-ultimate.org/node/3861)</a>
