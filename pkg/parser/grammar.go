
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
	c Cell
	s string
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
	"^",
	"@",
	"`",
	"CONS",
	"\n",
	")",
	";",
	":",
	"{",
	"}",
	"%",
	"(",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line grammar.y:198



//line yacctab:1
var yyExca = [...]int{
	-1, 0,
	18, 4,
	-2, 15,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 6,
	8, 13,
	9, 13,
	10, 13,
	11, 13,
	12, 13,
	13, 13,
	18, 13,
	19, 13,
	-2, 16,
	-1, 9,
	1, 1,
	18, 4,
	-2, 15,
	-1, 46,
	18, 33,
	-2, 15,
	-1, 66,
	18, 33,
	-2, 15,
}

const yyNprod = 48
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 124

var yyAct = [...]int{

	4, 7, 17, 60, 6, 8, 20, 67, 69, 51,
	8, 8, 32, 33, 34, 8, 36, 16, 58, 38,
	19, 35, 39, 66, 44, 41, 42, 50, 46, 43,
	47, 48, 9, 28, 29, 30, 43, 14, 15, 40,
	62, 3, 54, 23, 24, 49, 27, 61, 57, 55,
	56, 31, 25, 26, 59, 64, 45, 18, 65, 53,
	37, 28, 29, 30, 13, 14, 15, 61, 68, 5,
	70, 23, 24, 28, 29, 30, 16, 21, 22, 2,
	25, 26, 1, 23, 24, 12, 13, 14, 15, 21,
	22, 0, 25, 26, 10, 11, 12, 13, 14, 15,
	0, 0, 0, 0, 0, 63, 10, 11, 12, 13,
	14, 15, 0, 0, 0, 0, 0, 52, 10, 11,
	12, 13, 14, 15,
}
var yyPact = [...]int{

	-9, -1000, 14, -1000, 110, -1000, -3, 68, -1000, -9,
	-1000, -9, -9, -9, 28, -9, -1000, -9, 68, -1000,
	12, 68, 10, 28, 28, 38, -10, -1000, -1000, -1000,
	-1000, -1000, 75, 53, 25, 12, 98, -1000, -1000, 56,
	-1000, 12, 28, 28, -1000, 68, -5, 19, 19, 33,
	86, -1000, -1000, -9, -1000, 19, -1000, -1000, -1000, 5,
	-1000, 110, -17, -1000, -1000, 56, -15, -1000, -1000, -1000,
	-1000,
}
var yyPgo = [...]int{

	0, 82, 79, 41, 0, 6, 69, 4, 1, 2,
	60, 59, 57, 20, 56, 54, 3, 46,
}
var yyR1 = [...]int{

	0, 1, 2, 2, 3, 3, 4, 4, 4, 4,
	4, 4, 4, 6, 6, 8, 8, 7, 7, 10,
	10, 11, 11, 9, 9, 9, 13, 13, 13, 14,
	14, 15, 15, 16, 16, 12, 12, 5, 5, 5,
	5, 5, 5, 5, 5, 17, 17, 17,
}
var yyR2 = [...]int{

	0, 2, 1, 3, 0, 1, 2, 3, 3, 3,
	3, 4, 1, 1, 3, 0, 1, 1, 2, 1,
	3, 1, 3, 1, 2, 1, 2, 3, 2, 2,
	4, 1, 3, 0, 1, 1, 2, 3, 2, 2,
	3, 4, 3, 2, 1, 1, 1, 1,
}
var yyChk = [...]int{

	-1000, -1, -2, -3, -4, -6, -7, -8, 20, 18,
	8, 9, 10, 11, 12, 13, 20, -9, -12, -13,
	-5, 21, 22, 15, 16, 24, 25, -17, 5, 6,
	7, -3, -4, -4, -4, -5, -4, -10, -8, -7,
	-13, -5, 14, 17, -9, -14, 18, -5, -5, 7,
	-4, 19, 19, -11, -9, -5, -5, -9, 23, -15,
	-16, -4, 7, 19, -8, -7, 18, 24, -9, 23,
	-16,
}
var yyDef = [...]int{

	-2, -2, 0, 2, 5, 12, -2, 0, 17, -2,
	6, 15, 15, 15, 0, 15, 18, 15, 23, 25,
	35, 0, 0, 0, 0, 0, 15, 44, 45, 46,
	47, 3, 7, 8, 9, 10, 0, 14, 19, 16,
	24, 36, 0, 0, 26, 28, -2, 38, 39, 0,
	0, 43, 11, 15, 21, 37, 40, 27, 29, 0,
	31, 34, 0, 42, 20, 16, -2, 41, 22, 30,
	32,
}
var yyTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	18, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 24, 3, 3,
	25, 19, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 21, 20,
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
	state     func() int
}

func (p *yyParserImpl) Lookahead() int {
	return p.lookahead()
}

func yyNewParser() yyParser {
	p := &yyParserImpl{
		lookahead: func() int { return -1 },
		state:     func() int { return -1 },
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
	yyS := make([]yySymType, yyMaxDepth)

	startyyVAL := yyVAL
start:
	yyVAL = startyyVAL

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yytoken := -1 // yychar translated into internal numbering
	yyrcvr.state = func() int { return yystate }
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
		yyDollar = yyS[yypt-0:yypt+1]
		//line grammar.y:39
		{ yyVAL.c = Null }
	case 5:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:41
		{
		yyVAL.c = yyDollar[1].c
		if (yyDollar[1].c != Null) {
			yylex.(*scanner).process(yyDollar[1].c)
		}
		goto start
	}
	case 6:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:49
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c)
	}
	case 7:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:53
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
	}
	case 8:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:57
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
	}
	case 9:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:61
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
	}
	case 10:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:65
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[3].c, yyDollar[1].c)
	}
	case 11:
		yyDollar = yyS[yypt-4:yypt+1]
		//line grammar.y:69
		{
		yyVAL.c = List(NewSymbol(yyDollar[2].s), yyDollar[1].c, yyDollar[3].c)
	}
	case 12:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:73
		{ yyVAL.c = yyDollar[1].c }
	case 13:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:75
		{ yyVAL.c = Null }
	case 14:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:77
		{
		if yyDollar[3].c == Null {
			yyVAL.c = yyDollar[2].c
		} else {
			yyVAL.c = Cons(NewSymbol("block"), Cons(yyDollar[2].c, yyDollar[3].c))
		}
	}
	case 19:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:93
		{ yyVAL.c = Null }
	case 20:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:95
		{ yyVAL.c = yyDollar[2].c }
	case 21:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:97
		{ yyVAL.c = Cons(yyDollar[1].c, Null) }
	case 22:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:99
		{ yyVAL.c = AppendTo(yyDollar[1].c, yyDollar[3].c) }
	case 23:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:101
		{ yyVAL.c = yyDollar[1].c }
	case 24:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:103
		{
		yyVAL.c = JoinTo(yyDollar[1].c, yyDollar[2].c)
	}
	case 25:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:107
		{ yyVAL.c = yyDollar[1].c }
	case 26:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:109
		{ yyVAL.c = Cons(yyDollar[2].c, Null) }
	case 27:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:111
		{
		if yyDollar[2].c == Null {
			yyVAL.c = yyDollar[3].c
		} else {
			yyVAL.c = JoinTo(yyDollar[2].c, yyDollar[3].c)
		}
	}
	case 28:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:119
		{
		yyVAL.c = yyDollar[2].c
	}
	case 29:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:123
		{ yyVAL.c = Null }
	case 30:
		yyDollar = yyS[yypt-4:yypt+1]
		//line grammar.y:125
		{ yyVAL.c = yyDollar[2].c }
	case 31:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:127
		{
		if yyDollar[1].c == Null {
			yyVAL.c = yyDollar[1].c
		} else {
			yyVAL.c = Cons(yyDollar[1].c, Null)
		}
	}
	case 32:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:135
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
	case 33:
		yyDollar = yyS[yypt-0:yypt+1]
		//line grammar.y:151
		{ yyVAL.c = Null }
	case 34:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:153
		{ yyVAL.c = yyDollar[1].c }
	case 35:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:155
		{ yyVAL.c = Cons(yyDollar[1].c, Null) }
	case 36:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:157
		{ yyVAL.c = AppendTo(yyDollar[1].c, yyDollar[2].c) }
	case 37:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:159
		{
		t := yylex.(*scanner).task
		s := Cons(task.NewString(t, ""), NewSymbol("join"))
		yyVAL.c = List(s, yyDollar[1].c, yyDollar[3].c)
	}
	case 38:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:165
		{
		yyVAL.c = List(NewSymbol("splice"), yyDollar[2].c)
	}
	case 39:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:169
		{
		yyVAL.c = List(NewSymbol("backtick"), yyDollar[2].c)
	}
	case 40:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:173
		{
		yyVAL.c = Cons(yyDollar[1].c, yyDollar[3].c)
	}
	case 41:
		yyDollar = yyS[yypt-4:yypt+1]
		//line grammar.y:177
		{
		yyVAL.c = yylex.(*scanner).deref(yyDollar[2].s, yyDollar[3].s)
	}
	case 42:
		yyDollar = yyS[yypt-3:yypt+1]
		//line grammar.y:181
		{ yyVAL = yyDollar[2] }
	case 43:
		yyDollar = yyS[yypt-2:yypt+1]
		//line grammar.y:183
		{ yyVAL.c = Null }
	case 44:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:185
		{ yyVAL = yyDollar[1] }
	case 45:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:187
		{
		v, _ := strconv.Unquote(yyDollar[1].s)
		yyVAL.c = task.NewString(yylex.(*scanner).task, v)
	}
	case 46:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:192
		{
		yyVAL.c = task.NewString(yylex.(*scanner).task, yyDollar[1].s[1:len(yyDollar[1].s)-1])
	}
	case 47:
		yyDollar = yyS[yypt-1:yypt+1]
		//line grammar.y:196
		{ yyVAL.c = NewSymbol(yyDollar[1].s) }
	}
	goto yystack /* stack new state and value */
}
