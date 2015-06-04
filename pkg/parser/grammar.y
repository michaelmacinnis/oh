// Released under an MIT-style license. See LICENSE. -*- mode: Go -*-

%token CTRLC DOUBLE_QUOTED SINGLE_QUOTED SYMBOL
%left BACKGROUND /* & */
%left ORF		/* || */
%left ANDF	   /* && */
%left PIPE	   /* |,|+,!|,!|+ */
%left REDIRECT   /* <,>,!>,>>,!>> */
%left SUBSTITUTE /* <(,>( */
%left "^"
%right "@"
%right "`"
%left CONS

%{
package parser

import (
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
		yylex.(*scanner).process($1.c)
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

command: command SUBSTITUTE command ")" {
	$$.c = List(NewSymbol($2.s), $1.c, $3.c)
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

expression: expression "^" expression {
	t := yylex.(*scanner).task
	s := Cons(task.NewString(t, ""), NewSymbol("join"))
	$$.c = List(s, $1.c, $3.c)
};

expression: "@" expression {
	$$.c = List(NewSymbol("splice"), $2.c)
};

expression: "`" expression {
	$$.c = List(NewSymbol("backtick"), $2.c)
};

expression: expression CONS expression {
	$$.c = Cons($1.c, $3.c)
};

expression: "%" SYMBOL SYMBOL "%" {
	$$.c = yylex.(*scanner).deref($2.s, $3.s)
};

expression: "(" command ")" { $$ = $2 };

expression: "(" ")" { $$.c = Null };

expression: word { $$ = $1 };

word: DOUBLE_QUOTED {
	v, _ := strconv.Unquote($1.s)
	$$.c = task.NewString(yylex.(*scanner).task, v)
};

word: SINGLE_QUOTED {
	$$.c = task.NewString(yylex.(*scanner).task, $1.s[1:len($1.s)-1])
};

word: SYMBOL { $$.c = NewSymbol($1.s) };

%%

