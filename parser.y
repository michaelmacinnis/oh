/* Released under an MIT-style license. See LICENSE. -*- mode: Go -*- */

%token DEDENT END INDENT STRING SYMBOL
%left BACKGROUND /* & */
%left ORF        /* || */
%left ANDF       /* && */
%left PIPE       /* |,!| */
%left REDIRECT   /* <,>,!>,>>,!>> */
%right "@"
%right "'"
%right "`"
%left CONS

%{
package main

import (
    "fmt"
    "strconv"
    "unsafe"
)

type yySymType struct {
    yys int
    c Cell
    s string
}
%}

%%

program: top_block "\n";

top_block: opt_evaluate_command;

top_block: top_block "\n" opt_evaluate_command;

opt_evaluate_command: error;

opt_evaluate_command: { $$.c = Null }; /* Empty */

opt_evaluate_command: command {
    $$.c = $1.c
    if ($1.c != Null) {
        yylex.(*scanner).process($1.c)
    }
};

command: command BACKGROUND {
    $$.c = List(NewSymbol($2.s), $1.c)
};

command: command ORF command {
    $$.c = List(NewSymbol($2.s), $1.c, $3.c)
};

command: command ANDF command  {
    $$.c = List(NewSymbol($2.s), $1.c, $3.c)
};

command: command PIPE command  {
    $$.c = List(NewSymbol($2.s), $1.c, $3.c)
};

command: command REDIRECT expression {
    $$.c = List(NewSymbol($2.s), $3.c, $1.c)
};

command: unit { $$.c = $1.c };

unit: semicolon { $$.c = Null };

unit: opt_semicolon statement opt_clauses {
    if $3.c == Null {
        $$.c = $2.c
    } else {
        $$.c = Cons(NewSymbol("block"), Cons($2.c, $3.c))
    }
};

opt_semicolon: ; /* Empty */

opt_semicolon: semicolon;

semicolon: ";";

semicolon: semicolon ";";

opt_clauses: opt_semicolon { $$.c = Null };

opt_clauses: semicolon clauses opt_semicolon { $$.c = $2.c };

clauses: statement { $$.c = Cons($1.c, Null) };

clauses: clauses semicolon statement { $$.c = AppendTo($1.c, $3.c) };

statement: list { $$.c = $1.c };

statement: list sub_statement {
    $$.c = JoinTo($1.c, $2.c)
};

statement: sub_statement { $$.c = $1.c };

sub_statement: ":" statement { $$.c = Cons($2.c, Null) };

sub_statement: "{" sub_block statement {
    if $2.c == Null {
        $$.c = $3.c
    } else {
        $$.c = JoinTo($2.c, $3.c)
    }
};

sub_statement: "{" sub_block {
    $$.c = $2.c
};

sub_block: "\n" "}" { $$.c = Null };

sub_block: "\n" block "\n" "}" { $$.c = $2.c };

block: opt_command {
    if $1.c == Null {
        $$.c = $1.c
    } else {
        $$.c = Cons($1.c, Null)
    }
};

block: block "\n" opt_command {
    if $1.c == Null {
        if $3.c == Null {
            $$.c = $3.c
        } else {
            $$.c = Cons($3.c, Null)
        }
    } else {
        if $3.c == Null {
            $$.c = $1.c
        } else {
            $$.c = AppendTo($1.c, $3.c)
        }
    }
};

opt_command: { $$.c = Null };

opt_command: command { $$.c = $1.c };

list: expression { $$.c = Cons($1.c, Null) };

list: list expression { $$.c = AppendTo($1.c, $2.c) };

expression: "@" expression {
    $$.c = List(NewSymbol("splice"), $2.c)
};

expression: "'" expression {
    $$.c = List(NewSymbol("quote"), $2.c)
};

expression: "`" expression {
    $$.c = List(NewSymbol("backtick"), $2.c)
};

expression: expression CONS expression {
    $$.c = Cons($1.c, $3.c)
};

expression: "%" SYMBOL SYMBOL "%" {
    kind := $2.s
    value, _ := strconv.ParseUint($3.s, 0, 64)

    addr := uintptr(value)

    switch {
    case kind == "bound":
        $$.c = (*Bound)(unsafe.Pointer(addr))
    case kind == "builtin":
        $$.c = (*Builtin)(unsafe.Pointer(addr))
    case kind == "channel":
        $$.c = (*Channel)(unsafe.Pointer(addr))
    case kind == "env":
        $$.c = (*Env)(unsafe.Pointer(addr))
    case kind == "function":
        $$.c = (*Function)(unsafe.Pointer(addr))
    case kind == "method":
        $$.c = (*Method)(unsafe.Pointer(addr))
    case kind == "object":
        $$.c = (*Object)(unsafe.Pointer(addr))
    case kind == "reference":
        $$.c = (*Reference)(unsafe.Pointer(addr))
    case kind == "scope":
        $$.c = (*Scope)(unsafe.Pointer(addr))
    case kind == "syntax":
        $$.c = (*Syntax)(unsafe.Pointer(addr))
    case kind == "task":
        $$.c = (*Task)(unsafe.Pointer(addr))
    case kind == "unbound":
        $$.c = (*Unbound)(unsafe.Pointer(addr))

    default:
        $$.c = Null
    }

};

expression: "(" command ")" { $$ = $2 };

expression: "(" ")" { $$.c = Null };

expression: word { $$ = $1 };

word: STRING { $$.c = NewString($1.s[1:len($1.s)-1]) };

word: SYMBOL { $$.c = NewSymbol($1.s) };

%%

type ReadStringer interface {
    ReadString(delim byte) (line string, err error)
}

type scanner struct {
    process func(Cell)
        
    input ReadStringer
    line []rune

    state int
    indent int

    cursor int
    start int

    previous rune
    token rune

    finished bool
}

const (
    ssStart = iota; ssAmpersand; ssBang; ssBangGreater;
    ssColon; ssComment; ssGreater; ssPipe; ssString; ssSymbol
)

func (s *scanner) Lex(lval *yySymType) (token int) {
    var operator = map[string] string {
        "!>": "redirect-stderr",
        "!>>": "append-stderr",
        "!|": "pipe-stderr",
        "!|+": "channel-stderr",
        "&": "spawn",
        "&&": "and",
        "<": "redirect-stdin",
        ">": "redirect-stdout",
        ">>": "append-stdout",
        "|": "pipe-stdout",
        "|+": "channel-stdout",
        "||": "or",
    }

    defer func() {
        exists := false

        switch s.token {
        case BACKGROUND, ORF, ANDF, PIPE, REDIRECT:
            lval.s, exists = operator[string(s.line[s.start:s.cursor])]
            if exists {
                break
            }
            fallthrough
        default:
            lval.s = string(s.line[s.start:s.cursor])
        }

        s.state = ssStart
        s.previous = s.token
        s.token = 0
    }()

main:
    for s.token == 0 {
        if s.cursor >= len(s.line) {
            if s.finished {
                return 0
            }
            
            line, error := s.input.ReadString('\n')
            if error != nil {
                line += "\n"
                s.finished = true
            }
            
            if s.start < s.cursor - 1 {
                s.line = append(s.line[s.start:s.cursor], []rune(line)...)
                s.cursor -= s.start
            } else {
                s.cursor = 0
            }
            s.line = []rune(line)
            s.start = 0
            s.token = 0
        }

        switch s.state {
        case ssStart:
            s.start = s.cursor

            switch s.line[s.cursor] {
            default:
                s.state = ssSymbol
                continue main
            case '\n', '%', '\'', '(', ')', ';', '@', '`', '{', '}':
                s.token = s.line[s.cursor]
            case '&':
                s.state = ssAmpersand
            case '<':
                s.token = REDIRECT
            case '|':
                s.state = ssPipe
            case '\t', ' ':
                s.state = ssStart
            case '!':
                s.state = ssBang
            case '"':
                s.state = ssString
            case '#':
                s.state = ssComment
            case ':':
                s.state = ssColon
            case '>':
                s.state = ssGreater
            }

        case ssAmpersand:
            switch s.line[s.cursor] {
            case '&':
                s.token = ANDF
            default:
                s.token = BACKGROUND
                continue main
            }

        case ssBang:
            switch s.line[s.cursor] {
            case '>':
                s.state = ssBangGreater
            case '|':
		s.state = ssPipe
            default:
                s.state = ssSymbol
                continue main
            }

        case ssBangGreater:
            s.token = REDIRECT
            if s.line[s.cursor] != '>' {
                continue main
            }

        case ssColon:
            switch s.line[s.cursor] {
            case ':':
                s.token = CONS
            default:
                s.token = ':'
                continue main
            }

        case ssComment:
            for s.line[s.cursor + 1] != '\n' ||
                s.line[s.cursor] == '\\' {
                s.cursor++

                if s.cursor + 1 >= len(s.line) {
                    continue main
                }
            }
            s.state = ssStart

        case ssGreater:
            s.token = REDIRECT
            if s.line[s.cursor] != '>' {
                continue main
            }

        case ssPipe:
            switch s.line[s.cursor] {
            case '+':
                s.token = PIPE
            case '|':
                s.token = ORF
            default:
                s.token = PIPE
                continue main
            }

        case ssString:
            for s.line[s.cursor] != '"' ||
                s.line[s.cursor - 1] == '\\' {
                s.cursor++

                if s.cursor >= len(s.line) {
                    continue main
                }
            }
            s.token = STRING

        case ssSymbol:
            switch s.line[s.cursor] {
            case '\n','%','&','\'','(',')',';',
                '<','@','`','{','|','}',
                '\t',' ','"','#',':','>':
                if s.line[s.cursor - 1] != '\\' {
                    s.token = SYMBOL
                    continue main
                }
            }

        }
        s.cursor++

        if (s.token == '\n') {
            switch s.previous {
            case ORF, ANDF, PIPE, REDIRECT:
                s.token = 0
            }
        }
    }

    return int(s.token)
}

func (s *scanner) Error (msg string) {
    println(msg)
}

func Parse(r ReadStringer, p func(Cell)) {
    s := new(scanner)

    s.process = p

    s.input = r
    s.line = []rune("")

    s.state = ssStart
    s.indent = 0

    s.cursor = 0
    s.start = 0

    s.previous = 0
    s.token = 0

    yyParse(s)
}
