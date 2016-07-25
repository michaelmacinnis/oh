//line grammar.y:16
package parser

import __yyfmt__ "fmt"

//line grammar.y:16
import (
	"github.com/michaelmacinnis/adapted"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"strconv"
)

type yySymType struct {
	yys int
	c   Cell
	s   string
}

const BANG_DOUBLE = 57346
const BRACE_EXPANSION = 57347
const CTRLC = 57348
const DOUBLE_QUOTED = 57349
const SINGLE_QUOTED = 57350
const SYMBOL = 57351
const BACKGROUND = 57352
const ORF = 57353
const ANDF = 57354
const PIPE = 57355
const REDIRECT = 57356
const SUBSTITUTE = 57357
const CONS = 57358

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"BANG_DOUBLE",
	"BRACE_EXPANSION",
	"CTRLC",
	"DOUBLE_QUOTED",
	"SINGLE_QUOTED",
	"SYMBOL",
	"BACKGROUND",
	"ORF",
	"ANDF",
	"PIPE",
	"REDIRECT",
	"SUBSTITUTE",
	"\"@\"",
	"\"`\"",
	"CONS",
	"\"\\n\"",
	"\";\"",
	"\")\"",
	"\":\"",
	"\"{\"",
	"\"}\"",
	"\"%\"",
	"\"(\"",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyInitialStackSize = 16

//line grammar.y:227

//line yacctab:1
var yyExca = [...]int{
	-1, 0,
	19, 4,
	-2, 14,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 6,
	10, 12,
	11, 12,
	12, 12,
	13, 12,
	14, 12,
	19, 12,
	21, 12,
	-2, 15,
	-1, 9,
	1, 1,
	19, 4,
	-2, 14,
	-1, 48,
	19, 37,
	-2, 14,
	-1, 68,
	19, 37,
	-2, 14,
}

const yyNprod = 53
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 131

var yyAct = [...]int{

	4, 41, 17, 61, 16, 7, 6, 8, 69, 8,
	20, 73, 34, 35, 36, 10, 11, 12, 13, 14,
	8, 53, 39, 40, 46, 37, 67, 52, 8, 44,
	15, 68, 59, 48, 49, 50, 10, 11, 12, 13,
	14, 9, 45, 56, 42, 55, 14, 64, 63, 62,
	58, 10, 11, 12, 13, 14, 57, 19, 13, 14,
	65, 66, 28, 32, 51, 29, 30, 31, 27, 62,
	72, 70, 74, 75, 23, 24, 43, 3, 15, 60,
	21, 22, 47, 25, 26, 28, 32, 33, 29, 30,
	31, 12, 13, 14, 18, 71, 54, 23, 24, 38,
	5, 2, 1, 21, 22, 0, 25, 26, 28, 32,
	0, 29, 30, 31, 0, 0, 0, 0, 0, 0,
	23, 24, 0, 0, 0, 0, 0, 0, 0, 25,
	26,
}
var yyPact = [...]int{

	-11, -1000, 22, -1000, 41, -1000, 10, 81, -1000, -11,
	-1000, -11, -11, -11, 104, -1000, -11, 29, 81, -1000,
	24, 81, 14, 104, 104, 55, 0, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 79, 45, 32, 24, -1000, -1000,
	58, -1000, -11, -1000, 24, 104, -1000, 81, 8, 24,
	24, 39, 26, -1000, -11, -1000, 5, -1000, -1000, -1000,
	12, -1000, 41, -17, -1000, -1000, 58, 81, -13, -1000,
	-1000, 29, -1000, -1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 102, 101, 77, 0, 10, 100, 6, 5, 4,
	99, 96, 1, 95, 2, 94, 57, 82, 79, 3,
	68,
}
var yyR1 = [...]int{

	0, 1, 2, 2, 3, 3, 4, 4, 4, 4,
	4, 4, 6, 6, 8, 8, 7, 7, 10, 10,
	11, 11, 12, 12, 9, 13, 13, 14, 14, 14,
	16, 16, 16, 17, 17, 18, 18, 19, 19, 15,
	15, 5, 5, 5, 5, 5, 5, 5, 20, 20,
	20, 20, 20,
}
var yyR2 = [...]int{

	0, 2, 1, 3, 0, 1, 2, 3, 3, 3,
	3, 1, 1, 3, 0, 1, 1, 2, 1, 3,
	1, 3, 0, 5, 2, 0, 1, 1, 2, 1,
	2, 3, 2, 2, 4, 1, 3, 0, 1, 1,
	2, 2, 2, 3, 4, 3, 2, 1, 1, 1,
	1, 1, 1,
}
var yyChk = [...]int{

	-1000, -1, -2, -3, -4, -6, -7, -8, 20, 19,
	10, 11, 12, 13, 14, 20, -9, -14, -15, -16,
	-5, 22, 23, 16, 17, 25, 26, -20, 4, 7,
	8, 9, 5, -3, -4, -4, -4, -5, -10, -8,
	-7, -12, 15, -16, -5, 18, -14, -17, 19, -5,
	-5, 9, -4, 21, -11, -9, -4, -5, -14, 24,
	-18, -19, -4, 9, 21, -8, -7, 21, 19, 25,
	-9, -13, -14, 24, -19, -12,
}
var yyDef = [...]int{

	-2, -2, 0, 2, 5, 11, -2, 0, 16, -2,
	6, 14, 14, 14, 0, 17, 14, 22, 27, 29,
	39, 0, 0, 0, 0, 0, 14, 47, 48, 49,
	50, 51, 52, 3, 7, 8, 9, 10, 13, 18,
	15, 24, 14, 28, 40, 0, 30, 32, -2, 41,
	42, 0, 0, 46, 14, 20, 0, 43, 31, 33,
	0, 35, 38, 0, 45, 19, 15, 25, -2, 44,
	21, 22, 26, 34, 36, 23,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	19, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 25, 3, 3,
	26, 21, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 22, 20,
	3, 3, 3, 3, 16, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 17, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 23, 3, 24,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 18,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lval  yySymType
	stack [yyInitialStackSize]yySymType
	char  int
}

func (p *yyParserImpl) Lookahead() int {
	return p.char
}

func yyNewParser() yyParser {
	return &yyParserImpl{}
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := yyrcvr.stack[:]

	startyyVAL := yyVAL
start:
	yyVAL = startyyVAL

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yyrcvr.char = -1
	yytoken := -1 // yyrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yyrcvr.char = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yyrcvr.char < 0 {
		yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yyrcvr.char = -1
		yytoken = -1
		yyVAL = yyrcvr.lval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yyrcvr.char < 0 {
			yyrcvr.char, yytoken = yylex1(yylex, &yyrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yyrcvr.char = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 4:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammar.y:39
		{
			yyVAL.c = Null
		}
	case 5:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:41
		{
			yyVAL.c = yyDollar[1].c
			if yyDollar[1].c != Null {
				s := yylex.(*scanner)
				_, ok := s.process(yyDollar[1].c, s.filename, s.lineno, "")
				if !ok {
					return -1
				}
			}
			goto start
		}
	case 6:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:53
		{
			yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c)
		}
	case 7:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:57
		{
			yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
		}
	case 8:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:61
		{
			yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
		}
	case 9:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:65
		{
			yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
		}
	case 10:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:69
		{
			yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[3].c, yyDollar[1].c)
		}
	case 11:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:73
		{
			yyVAL.c = yyDollar[1].c
		}
	case 12:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:75
		{
			yyVAL.c = Null
		}
	case 13:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:77
		{
			if yyDollar[3].c == Null {
				yyVAL.c = yyDollar[2].c
			} else {
				yyVAL.c = Cons(NewSymbol("block"), Cons(yyDollar[2].c, yyDollar[3].c))
			}
		}
	case 18:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:93
		{
			yyVAL.c = Null
		}
	case 19:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:95
		{
			yyVAL.c = yyDollar[2].c
		}
	case 20:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:97
		{
			yyVAL.c = Cons(yyDollar[1].c, Null)
		}
	case 21:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:99
		{
			yyVAL.c = AppendTo(yyDollar[1].c, yyDollar[3].c)
		}
	case 22:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammar.y:101
		{
			yyVAL.c = Null
		}
	case 23:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line grammar.y:103
		{
			lst := List(Cons(NewSymbol(yyDollar[1].s), yyDollar[2].c))
			if yyDollar[4].c != Null {
				lst = JoinTo(lst, yyDollar[4].c)
			}
			if yyDollar[5].c != Null {
				lst = JoinTo(lst, yyDollar[5].c)
			}
			yyVAL.c = lst
		}
	case 24:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:114
		{
			if yyDollar[2].c != Null {
				sym := NewSymbol("_process_substitution_")
				yyVAL.c = JoinTo(Cons(sym, yyDollar[1].c), yyDollar[2].c)
			} else {
				yyVAL.c = yyDollar[1].c
			}
		}
	case 25:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammar.y:123
		{
			yyVAL.c = Null
		}
	case 26:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:125
		{
			yyVAL.c = yyDollar[1].c
		}
	case 27:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:127
		{
			yyVAL.c = yyDollar[1].c
		}
	case 28:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:129
		{
			yyVAL.c = JoinTo(yyDollar[1].c, yyDollar[2].c)
		}
	case 29:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:133
		{
			yyVAL.c = yyDollar[1].c
		}
	case 30:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:135
		{
			yyVAL.c = Cons(yyDollar[2].c, Null)
		}
	case 31:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:137
		{
			if yyDollar[2].c == Null {
				yyVAL.c = yyDollar[3].c
			} else {
				yyVAL.c = JoinTo(yyDollar[2].c, yyDollar[3].c)
			}
		}
	case 32:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:145
		{
			yyVAL.c = yyDollar[2].c
		}
	case 33:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:149
		{
			yyVAL.c = Null
		}
	case 34:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammar.y:151
		{
			yyVAL.c = yyDollar[2].c
		}
	case 35:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:153
		{
			if yyDollar[1].c == Null {
				yyVAL.c = yyDollar[1].c
			} else {
				yyVAL.c = Cons(yyDollar[1].c, Null)
			}
		}
	case 36:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:161
		{
			if yyDollar[1].c == Null {
				if yyDollar[3].c == Null {
					yyVAL.c = yyDollar[3].c
				} else {
					yyVAL.c = Cons(yyDollar[3].c, Null)
				}
			} else {
				if yyDollar[3].c == Null {
					yyVAL.c = yyDollar[1].c
				} else {
					yyVAL.c = AppendTo(yyDollar[1].c, yyDollar[3].c)
				}
			}
		}
	case 37:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammar.y:177
		{
			yyVAL.c = Null
		}
	case 38:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:179
		{
			yyVAL.c = yyDollar[1].c
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:181
		{
			yyVAL.c = Cons(yyDollar[1].c, Null)
		}
	case 40:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:183
		{
			yyVAL.c = AppendTo(yyDollar[1].c, yyDollar[2].c)
		}
	case 41:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:185
		{
			yyVAL.c = List(NewSymbol("_splice_"), yyDollar[2].c)
		}
	case 42:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:189
		{
			yyVAL.c = List(NewSymbol("_backtick_"), yyDollar[2].c)
		}
	case 43:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:193
		{
			yyVAL.c = Cons(yyDollar[1].c, yyDollar[3].c)
		}
	case 44:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammar.y:197
		{
			value, _ := strconv.ParseUint(yyDollar[3].s, 0, 64)
			yyVAL.c = yylex.(*scanner).deref(yyDollar[2].s, uintptr(value))
		}
	case 45:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:202
		{
			yyVAL = yyDollar[2]
		}
	case 46:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:204
		{
			yyVAL.c = Null
		}
	case 47:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:206
		{
			yyVAL = yyDollar[1]
		}
	case 48:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:208
		{
			v, _ := adapted.Unquote(yyDollar[1].s[1:])
			yyVAL.c = NewString(v)
		}
	case 49:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:213
		{
			v, _ := adapted.Unquote(yyDollar[1].s)
			s := NewString(v)
			yyVAL.c = List(NewSymbol("interpolate"), s)
		}
	case 50:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:219
		{
			yyVAL.c = NewString(yyDollar[1].s[1 : len(yyDollar[1].s)-1])
		}
	case 51:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:223
		{
			yyVAL.c = NewSymbol(yyDollar[1].s)
		}
	case 52:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:225
		{
			yyVAL.c = NewSymbol(yyDollar[1].s)
		}
	}
	goto yystack /* stack new state and value */
}
