// Released under an MIT-style license. See LICENSE. -*- mode: Go -*-

%token CTRLC DOLLAR_DOUBLE DOLLAR_SINGLE DOUBLE_QUOTED SINGLE_QUOTED SYMBOL
%left BACKGROUND /* & */
%left ORF        /* || */
%left ANDF       /* && */
%left PIPE	 /* |,|+,!|,!|+ */
%left REDIRECT   /* <,>,!>,>>,!>> */
%left SUBSTITUTE /* <(,>( */
%right "@"
%right "`"
%left CONS

%{
package parser

import (
	"github.com/michaelmacinnis/adapted"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/task"
	"strconv"
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

opt_evaluate_command: { $$.c = Null }; /* Empty */

opt_evaluate_command: command {
	$$.c = $1.c
	if ($1.c != Null) {
		s := yylex.(*scanner)
		s.process($1.c, s.filename, s.lineno, "")
		if task.ForegroundTask().Stack == Null {
			return -1
		}
	}
	goto start
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

command: sequence { $$.c = $1.c };

sequence: semicolon { $$.c = Null };

sequence: opt_semicolon substitution opt_clauses {
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

clauses: substitution { $$.c = Cons($1.c, Null) };

clauses: clauses semicolon substitution { $$.c = AppendTo($1.c, $3.c) };

opt_substitution: { $$.c = Null };

opt_substitution: SUBSTITUTE command ")" opt_statement opt_substitution {
	lst := List(Cons(NewSymbol($1.s), $2.c))
	if $4.c != Null {
		lst = JoinTo(lst, $4.c)
	}
	if $5.c != Null {
		lst = JoinTo(lst, $5.c)
	}
	$$.c = lst
}

substitution: statement opt_substitution {
	if $2.c != Null {
		sym := NewSymbol("_process_substitution_")
		$$.c = JoinTo(Cons(sym, $1.c), $2.c)
	} else {
		$$.c = $1.c
	}
}

opt_statement: { $$.c = Null };

opt_statement: statement { $$.c = $1.c };

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
	$$.c = List(NewSymbol("_splice_"), $2.c)
};

expression: "`" expression {
	$$.c = List(NewSymbol("_backtick_"), $2.c)
};

expression: expression CONS expression {
	$$.c = Cons($1.c, $3.c)
};

expression: "%" SYMBOL SYMBOL "%" {
	value, _ := strconv.ParseUint($3.s, 0, 64)
	$$.c = yylex.(*scanner).deref($2.s, uintptr(value))
};

expression: "(" command ")" { $$ = $2 };

expression: "(" ")" { $$.c = Null };

expression: word { $$ = $1 };

word: DOLLAR_DOUBLE {
	v, _ := adapted.Unquote($1.s[1:])
	s := task.NewString(yylex.(*scanner).task, v)
	$$.c = List(NewSymbol("interpolate"), s)
};

word: DOLLAR_SINGLE {
	s := task.NewString(yylex.(*scanner).task, $1.s[2:len($1.s)-1])
	$$.c = List(NewSymbol("interpolate"), s)
};

word: DOUBLE_QUOTED {
	v, _ := adapted.Unquote($1.s)
	$$.c = task.NewString(yylex.(*scanner).task, v)
};

word: SINGLE_QUOTED {
	$$.c = task.NewString(yylex.(*scanner).task, $1.s[1:len($1.s)-1])
};

word: SYMBOL { $$.c = NewSymbol($1.s) };

%%

