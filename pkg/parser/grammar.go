//line grammar.y:16
package parser

import __yyfmt__ "fmt"

//line grammar.y:16
import (
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"github.com/michaelmacinnis/oh/pkg/task"
	"strconv"
)

type yySymType struct {
	yys int
	c   Cell
	s   string
}

const CTRLC = 57346
const DOUBLE_QUOTED = 57347
const SINGLE_QUOTED = 57348
const SYMBOL = 57349
const BACKGROUND = 57350
const ORF = 57351
const ANDF = 57352
const PIPE = 57353
const REDIRECT = 57354
const SUBSTITUTE = 57355
const CONS = 57356

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
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
	"\"^\"",
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
const yyMaxDepth = 200

//line grammar.y:225

//line yacctab:1
var yyExca = [...]int{
	-1, 0,
	18, 4,
	-2, 14,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 6,
	8, 12,
	9, 12,
	10, 12,
	11, 12,
	12, 12,
	18, 12,
	20, 12,
	-2, 15,
	-1, 9,
	1, 1,
	18, 4,
	-2, 14,
	-1, 47,
	18, 37,
	-2, 14,
	-1, 68,
	18, 37,
	-2, 14,
}

const yyNprod = 52
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 122

var yyAct = [...]int{

	4, 39, 17, 61, 16, 7, 6, 8, 69, 8,
	20, 73, 32, 33, 34, 15, 10, 11, 12, 13,
	14, 19, 37, 38, 45, 35, 8, 51, 67, 42,
	59, 8, 52, 68, 48, 49, 43, 47, 9, 44,
	41, 55, 44, 54, 12, 13, 14, 14, 62, 58,
	40, 13, 14, 3, 56, 57, 63, 50, 27, 65,
	66, 60, 46, 31, 28, 29, 30, 18, 71, 62,
	72, 70, 74, 75, 23, 24, 53, 36, 15, 5,
	21, 22, 2, 25, 26, 28, 29, 30, 28, 29,
	30, 1, 0, 0, 0, 23, 24, 0, 23, 24,
	0, 21, 22, 0, 25, 26, 0, 25, 26, 10,
	11, 12, 13, 14, 10, 11, 12, 13, 14, 0,
	0, 64,
}
var yyPact = [...]int{

	-10, -1000, 20, -1000, 106, -1000, -4, 80, -1000, -10,
	-1000, -10, -10, -10, 83, -1000, -10, 37, 80, -1000,
	22, 80, 19, 83, 83, 50, 12, -1000, -1000, -1000,
	-1000, -1000, 34, 40, 35, 22, -1000, -1000, 59, -1000,
	-10, -1000, 22, 83, 83, -1000, 80, 7, 25, 25,
	49, 101, -1000, -10, -1000, 8, 25, -1000, -1000, -1000,
	15, -1000, 106, -16, -1000, -1000, 59, 80, -12, -1000,
	-1000, 37, -1000, -1000, -1000, -1000,
}
var yyPgo = [...]int{

	0, 91, 82, 53, 0, 10, 79, 6, 5, 4,
	77, 76, 1, 68, 2, 67, 21, 62, 61, 3,
	58,
}
var yyR1 = [...]int{

	0, 1, 2, 2, 3, 3, 4, 4, 4, 4,
	4, 4, 6, 6, 8, 8, 7, 7, 10, 10,
	11, 11, 12, 12, 9, 13, 13, 14, 14, 14,
	16, 16, 16, 17, 17, 18, 18, 19, 19, 15,
	15, 5, 5, 5, 5, 5, 5, 5, 5, 20,
	20, 20,
}
var yyR2 = [...]int{

	0, 2, 1, 3, 0, 1, 2, 3, 3, 3,
	3, 1, 1, 3, 0, 1, 1, 2, 1, 3,
	1, 3, 0, 5, 2, 0, 1, 1, 2, 1,
	2, 3, 2, 2, 4, 1, 3, 0, 1, 1,
	2, 3, 2, 2, 3, 4, 3, 2, 1, 1,
	1, 1,
}
var yyChk = [...]int{

	-1000, -1, -2, -3, -4, -6, -7, -8, 19, 18,
	8, 9, 10, 11, 12, 19, -9, -14, -15, -16,
	-5, 21, 22, 15, 16, 24, 25, -20, 5, 6,
	7, -3, -4, -4, -4, -5, -10, -8, -7, -12,
	13, -16, -5, 14, 17, -14, -17, 18, -5, -5,
	7, -4, 20, -11, -9, -4, -5, -5, -14, 23,
	-18, -19, -4, 7, 20, -8, -7, 20, 18, 24,
	-9, -13, -14, 23, -19, -12,
}
var yyDef = [...]int{

	-2, -2, 0, 2, 5, 11, -2, 0, 16, -2,
	6, 14, 14, 14, 0, 17, 14, 22, 27, 29,
	39, 0, 0, 0, 0, 0, 14, 48, 49, 50,
	51, 3, 7, 8, 9, 10, 13, 18, 15, 24,
	14, 28, 40, 0, 0, 30, 32, -2, 42, 43,
	0, 0, 47, 14, 20, 0, 41, 44, 31, 33,
	0, 35, 38, 0, 46, 19, 15, 25, -2, 45,
	21, 22, 26, 34, 36, 23,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	18, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 24, 3, 3,
	25, 20, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 21, 19,
	3, 3, 3, 3, 15, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 14, 3, 16, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 22, 3, 23,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 17,
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
	lookahead func() int
}

func (p *yyParserImpl) Lookahead() int {
	return p.lookahead()
}

func yyNewParser() yyParser {
	p := &yyParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
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
	var yylval yySymType
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := make([]yySymType, yyMaxDepth)

	startyyVAL := yyVAL
start:
	yyVAL = startyyVAL

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yytoken := -1 // yychar translated into internal numbering
	yyrcvr.lookahead = func() int { return yychar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yychar = -1
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
	if yychar < 0 {
		yychar, yytoken = yylex1(yylex, &yylval)
		if yychar == CTRLC {
			goto start
		}

	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yychar = -1
		yytoken = -1
		yyVAL = yylval
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
		if yychar < 0 {
			yychar, yytoken = yylex1(yylex, &yylval)
			if yychar == CTRLC {
				goto start
			}

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
			yychar = -1
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
				s.process(yyDollar[1].c)
				if s.task.Stack == Null {
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
				sym := NewSymbol("process-substitution")
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
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:185
		{
			t := yylex.(*scanner).task
			s := Cons(task.NewString(t, ""), NewSymbol("join"))
			yyVAL.c = List(s, yyDollar[1].c, yyDollar[3].c)
		}
	case 42:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:191
		{
			yyVAL.c = List(NewSymbol("splice"), yyDollar[2].c)
		}
	case 43:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:195
		{
			yyVAL.c = List(NewSymbol("backtick"), yyDollar[2].c)
		}
	case 44:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:199
		{
			yyVAL.c = Cons(yyDollar[1].c, yyDollar[3].c)
		}
	case 45:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammar.y:203
		{
			value, _ := strconv.ParseUint(yyDollar[3].s, 0, 64)
			yyVAL.c = yylex.(*scanner).deref(yyDollar[2].s, uintptr(value))
		}
	case 46:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammar.y:208
		{
			yyVAL = yyDollar[2]
		}
	case 47:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammar.y:210
		{
			yyVAL.c = Null
		}
	case 48:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:212
		{
			yyVAL = yyDollar[1]
		}
	case 49:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:214
		{
			v, _ := strconv.Unquote(yyDollar[1].s)
			yyVAL.c = task.NewString(yylex.(*scanner).task, v)
		}
	case 50:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:219
		{
			yyVAL.c = task.NewString(yylex.(*scanner).task, yyDollar[1].s[1:len(yyDollar[1].s)-1])
		}
	case 51:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammar.y:223
		{
			yyVAL.c = NewSymbol(yyDollar[1].s)
		}
	}
	goto yystack /* stack new state and value */
}
