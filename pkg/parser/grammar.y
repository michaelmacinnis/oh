// Released under an MIT license. See LICENSE. -*- mode: Go -*-

%token BANG_STRING BRACE_EXPANSION CTRLC
%token DOUBLE_QUOTED SINGLE_QUOTED SYMBOL
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
	"strconv"
)
%}

%union{
	c Cell
	s string
}

%type <c> block clauses command expression list
%type <c> opt_clauses opt_command opt_evaluate_command
%type <c> opt_statement opt_substitution
%type <c> sequence statement sub_block sub_statement
%type <c> substitution word tail
%type <s> ANDF BACKGROUND BANG_STRING BRACE_EXPANSION
%type <s> DOUBLE_QUOTED ORF PIPE REDIRECT SINGLE_QUOTED
%type <s> SUBSTITUTE SYMBOL

%%

program: top_block "\n";

top_block: opt_evaluate_command;

top_block: top_block "\n" opt_evaluate_command;

opt_evaluate_command: { $$ = Null }; /* Empty */

opt_evaluate_command: command {
	$$ = $1
	if ($1 != Null) {
		s := GetLexer(ohlex)
		_, ok := s.yield($1, s.label, s.lines, "")
		if !ok {
			return -1
		}
	}
	goto ohstart
};

command: command BACKGROUND {
	$$ = List(NewSymbol($2), $1)
};

command: command ORF command {
	$$ = List(NewSymbol($2), $1, $3)
};

command: command ANDF command  {
	$$ = List(NewSymbol($2), $1, $3)
};

command: command PIPE command  {
	$$ = List(NewSymbol($2), $1, $3)
};

command: command REDIRECT expression {
	$$ = List(NewSymbol($2), $3, $1)
};

command: sequence { $$ = $1 };

sequence: semicolon { $$ = Null };

sequence: opt_semicolon substitution opt_clauses {
	if $3 == Null {
		$$ = $2
	} else {
		$$ = Cons(NewSymbol("block"), Cons($2, $3))
	}
};

opt_semicolon: ; /* Empty */

opt_semicolon: semicolon;

semicolon: ";";

semicolon: semicolon ";";

opt_clauses: opt_semicolon { $$ = Null };

opt_clauses: semicolon clauses opt_semicolon { $$ = $2 };

clauses: substitution { $$ = Cons($1, Null) };

clauses: clauses semicolon substitution { $$ = AppendTo($1, $3) };

opt_substitution: { $$ = Null };

opt_substitution: SUBSTITUTE command ")" opt_statement opt_substitution {
	lst := List(Cons(NewSymbol($1), $2))
	if $4 != Null {
		lst = JoinTo(lst, $4)
	}
	if $5 != Null {
		lst = JoinTo(lst, $5)
	}
	$$ = lst
}

substitution: statement opt_substitution {
	if $2 != Null {
		sym := NewSymbol("_process_substitution_")
		$$ = JoinTo(Cons(sym, $1), $2)
	} else {
		$$ = $1
	}
}

opt_statement: { $$ = Null };

opt_statement: statement { $$ = $1 };

statement: list { $$ = $1 };
	
statement: list sub_statement {
	$$ = JoinTo($1, $2)
};

statement: sub_statement { $$ = $1 };

sub_statement: ":" statement { $$ = Cons($2, Null) };

sub_statement: "{" sub_block statement {
	if $2 == Null {
		$$ = $3
	} else {
		$$ = JoinTo($2, $3)
	}
};

sub_statement: "{" sub_block {
	$$ = $2
};

sub_block: "\n" "}" { $$ = Null };

sub_block: "\n" block "\n" "}" { $$ = $2 };

block: opt_command {
	if $1 == Null {
		$$ = $1
	} else {
		$$ = Cons($1, Null)
	}
};

block: block "\n" opt_command {
	if $1 == Null {
		if $3 == Null {
			$$ = $3
		} else {
			$$ = Cons($3, Null)
		}
	} else {
		if $3 == Null {
			$$ = $1
		} else {
			$$ = AppendTo($1, $3)
		}
	}
};

opt_command: { $$ = Null };

opt_command: command { $$ = $1 };

list: expression { $$ = Cons($1, Null) };

list: list tail { $$ = AppendTo($1, $2) };

tail: "@" expression {
	$$ = List(NewSymbol("_splice_"), $2)
};

tail: expression { $$ = $1 };

expression: "`" expression {
	$$ = List(NewSymbol("_backtick_"), $2)
};

expression: expression CONS expression {
	$$ = Cons($1, $3)
};

expression: "%" SYMBOL SYMBOL "%" {
	value, _ := strconv.ParseUint($3, 0, 64)
	$$ = GetLexer(ohlex).deref($2, uintptr(value))
};

expression: "(" command ")" { $$ = $2 };

expression: "(" ")" { $$ = Null };

expression: word { $$ = $1 };

word: BANG_STRING {
	v, _ := adapted.Unquote($1[1:])
	$$ = NewString(v)
};

word: DOUBLE_QUOTED {
	v, _ := adapted.Unquote($1)
	s := NewString(v)
	$$ = List(NewSymbol("interpolate"), s)
};

word: SINGLE_QUOTED {
	$$ = NewString($1[1:len($1)-1])
};

word: SYMBOL {
	$$ = NewSymbol($1)
};

word: BRACE_EXPANSION { $$ = NewSymbol($1) };

%%

