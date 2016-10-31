//line grammar.y:16
package parser

import __yyfmt__ "fmt"

//line grammar.y:16
import (
	"github.com/michaelmacinnis/adapted"
	. "github.com/michaelmacinnis/oh/pkg/cell"
	"strconv"
)

//line grammar.y:25
type ohSymType struct {
	yys int
	c   Cell
	s   string
}

const BANG_STRING = 57346
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

var ohToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"BANG_STRING",
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
var ohStatenames = [...]string{}

const ohEofCode = 1
const ohErrCode = 2
const ohInitialStackSize = 16

//line grammar.y:251

//line yacctab:1
var ohExca = [...]int{
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
	-1, 49,
	19, 37,
	-2, 14,
	-1, 70,
	19, 37,
	-2, 14,
}

const ohNprod = 55
const ohPrivate = 57344

var ohTokenNames []string
var ohStates []string

const ohLast = 132

var ohAct = [...]int{

	4, 41, 17, 62, 16, 7, 6, 8, 71, 8,
	54, 75, 34, 35, 36, 28, 32, 23, 29, 30,
	31, 8, 39, 40, 47, 60, 8, 53, 24, 21,
	22, 15, 37, 21, 22, 15, 25, 26, 46, 70,
	49, 9, 51, 57, 50, 56, 19, 12, 13, 14,
	63, 59, 42, 10, 11, 12, 13, 14, 13, 14,
	14, 67, 68, 58, 69, 43, 65, 52, 64, 2,
	1, 63, 74, 72, 76, 77, 28, 32, 44, 29,
	30, 31, 27, 3, 48, 5, 73, 28, 32, 24,
	29, 30, 31, 33, 21, 22, 38, 25, 26, 45,
	24, 28, 32, 20, 29, 30, 31, 18, 25, 26,
	55, 61, 0, 0, 24, 10, 11, 12, 13, 14,
	0, 0, 25, 26, 0, 0, 66, 10, 11, 12,
	13, 14,
}
var ohPact = [...]int{

	6, -1000, 22, -1000, 117, -1000, 15, 72, -1000, 6,
	-1000, 6, 6, 6, 97, -1000, 6, 37, 7, -1000,
	83, 72, 21, 26, 97, 58, -11, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 35, 45, 46, 26, -1000, -1000,
	11, -1000, 6, -1000, -1000, 97, 26, -1000, 72, 1,
	97, 26, 57, 105, -1000, 6, -1000, 43, 26, -1000,
	-1000, 20, -1000, 117, -1000, -17, -1000, -1000, 11, 72,
	-13, -1000, -1000, 37, -1000, -1000, -1000, -1000,
}
var ohPgo = [...]int{

	0, 111, 110, 107, 0, 17, 103, 96, 3, 83,
	86, 1, 85, 2, 84, 46, 4, 82, 78, 70,
	69, 6, 5,
}
var ohR1 = [...]int{

	0, 19, 20, 20, 9, 9, 4, 4, 4, 4,
	4, 4, 12, 12, 22, 22, 21, 21, 7, 7,
	2, 2, 11, 11, 16, 10, 10, 13, 13, 13,
	15, 15, 15, 14, 14, 1, 1, 8, 8, 3,
	6, 6, 18, 18, 5, 5, 5, 5, 5, 5,
	17, 17, 17, 17, 17,
}
var ohR2 = [...]int{

	0, 2, 1, 3, 0, 1, 2, 3, 3, 3,
	3, 1, 1, 3, 0, 1, 1, 2, 1, 3,
	1, 3, 0, 5, 2, 0, 1, 1, 2, 1,
	2, 3, 2, 2, 4, 1, 3, 0, 1, 1,
	1, 2, 2, 1, 2, 3, 4, 3, 2, 1,
	1, 1, 1, 1, 1,
}
var ohChk = [...]int{

	-1000, -19, -20, -9, -4, -12, -21, -22, 20, 19,
	10, 11, 12, 13, 14, 20, -16, -13, -3, -15,
	-6, 22, 23, -5, 17, 25, 26, -17, 4, 7,
	8, 9, 5, -9, -4, -4, -4, -5, -7, -22,
	-21, -11, 15, -15, -18, 16, -5, -13, -14, 19,
	18, -5, 9, -4, 21, -2, -16, -4, -5, -13,
	24, -1, -8, -4, -5, 9, 21, -22, -21, 21,
	19, 25, -16, -10, -13, 24, -8, -11,
}
var ohDef = [...]int{

	-2, -2, 0, 2, 5, 11, -2, 0, 16, -2,
	6, 14, 14, 14, 0, 17, 14, 22, 27, 29,
	39, 0, 0, 40, 0, 0, 14, 49, 50, 51,
	52, 53, 54, 3, 7, 8, 9, 10, 13, 18,
	15, 24, 14, 28, 41, 0, 43, 30, 32, -2,
	0, 44, 0, 0, 48, 14, 20, 0, 42, 31,
	33, 0, 35, 38, 45, 0, 47, 19, 15, 25,
	-2, 46, 21, 22, 26, 34, 36, 23,
}
var ohTok1 = [...]int{

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
var ohTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 18,
}
var ohTok3 = [...]int{
	0,
}

var ohErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	ohDebug        = 0
	ohErrorVerbose = false
)

type ohLexer interface {
	Error(s string) bool
	Fatal(*ohSymType) bool
	Lex() *ohSymType
}

type ohParser interface {
	Lookahead() int
	Parse(ohLexer) int
}

type ohParserImpl struct {
	char  int
	lval  *ohSymType
	n     int
	p     int
	stack [ohInitialStackSize]ohSymType
}

func (p *ohParserImpl) Lookahead() int {
	return p.char
}

func ohNewParser() ohParser {
	return &ohParserImpl{}
}

const ohFlag = -1000

func ohTokname(c int) string {
	if c >= 1 && c-1 < len(ohToknames) {
		if ohToknames[c-1] != "" {
			return ohToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func ohStatname(s int) string {
	if s >= 0 && s < len(ohStatenames) {
		if ohStatenames[s] != "" {
			return ohStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func ohErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !ohErrorVerbose {
		return "syntax error"
	}

	for _, e := range ohErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + ohTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := ohPact[state]
	for tok := TOKSTART; tok-1 < len(ohToknames); tok++ {
		if n := base + tok; n >= 0 && n < ohLast && ohChk[ohAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if ohDef[state] == -2 {
		i := 0
		for ohExca[i] != -1 || ohExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; ohExca[i] >= 0; i += 2 {
			tok := ohExca[i]
			if tok < TOKSTART || ohExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if ohExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += ohTokname(tok)
	}
	return res
}

func ohlex1(lex ohLexer) (lval *ohSymType, char, token int) {
	token = 0
	lval = lex.Lex()

	if lval == nil {
		return nil, 0, 0
	}

	char = lval.yys
	if char <= 0 {
		token = ohTok1[0]
		goto out
	}
	if char < len(ohTok1) {
		token = ohTok1[char]
		goto out
	}
	if char >= ohPrivate {
		if char < ohPrivate+len(ohTok2) {
			token = ohTok2[char-ohPrivate]
			goto out
		}
	}
	for i := 0; i < len(ohTok3); i += 2 {
		token = ohTok3[i+0]
		if token == char {
			token = ohTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = ohTok2[1] /* unknown char */
	}
	if ohDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", ohTokname(token), uint(char))
	}
	return lval, char, token
}

func ohParse(ohlex ohLexer) int {
	return ohNewParser().Parse(ohlex)
}

func (ohrcvr *ohParserImpl) Parse(ohlex ohLexer) int {
	var ohn int
	var ohVAL ohSymType
	var ohDollar []ohSymType
	_ = ohDollar // silence set and not used
	ohS := ohrcvr.stack[:]

	zeroohVAL := ohVAL

ohstart:
	ohn = 0
	ohVAL = zeroohVAL

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	ohstate := 0
	ohrcvr.char = -1
	ohtoken := -1 // ohrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		ohstate = -1
		ohrcvr.char = -1
		ohtoken = -1
	}()
	ohp := -1
	goto ohstack

ret0:
	return 0

ret1:
	return 1

ohstack:
	/* put a state and value onto the stack */
	if ohDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", ohTokname(ohtoken), ohStatname(ohstate))
	}

	ohp++
	if ohp >= len(ohS) {
		nyys := make([]ohSymType, len(ohS)*2)
		copy(nyys, ohS)
		ohS = nyys
	}
	ohS[ohp] = ohVAL
	ohS[ohp].yys = ohstate

ohnewstate:
	ohn = ohPact[ohstate]
	if ohn <= ohFlag {
		goto ohdefault /* simple state */
	}
	if ohrcvr.char < 0 {
		ohrcvr.lval, ohrcvr.char, ohtoken = ohlex1(ohlex)
		if ohrcvr.lval == nil {
			goto ret0
		}
		if ohlex.Fatal(ohrcvr.lval) {
			goto ret1
		}
	}
	ohn += ohtoken
	if ohn < 0 || ohn >= ohLast {
		goto ohdefault
	}
	ohn = ohAct[ohn]
	if ohChk[ohn] == ohtoken { /* valid shift */
		ohrcvr.char = -1
		ohtoken = -1
		ohVAL = *ohrcvr.lval
		ohstate = ohn
		if Errflag > 0 {
			Errflag--
		}
		goto ohstack
	}

ohdefault:
	/* default state action */
	ohn = ohDef[ohstate]
	if ohn == -2 {
		if ohrcvr.char < 0 {
			ohrcvr.lval, ohrcvr.char, ohtoken = ohlex1(ohlex)
			if ohrcvr.lval == nil {
				goto ret0
			}
			if ohlex.Fatal(ohrcvr.lval) {
				goto ret1
			}
		}

		/* look through exception table */
		xi := 0
		for {
			if ohExca[xi+0] == -1 && ohExca[xi+1] == ohstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			ohn = ohExca[xi+0]
			if ohn < 0 || ohn == ohtoken {
				break
			}
		}
		ohn = ohExca[xi+1]
		if ohn < 0 {
			goto ret0
		}
	}
	if ohn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			if ohlex.Error(ohErrorMessage(ohstate, ohtoken)) {
				goto ret1
			}
			Nerrs++
			if ohDebug >= 1 {
				__yyfmt__.Printf("%s", ohStatname(ohstate))
				__yyfmt__.Printf(" saw %s\n", ohTokname(ohtoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for ohp >= 0 {
				ohn = ohPact[ohS[ohp].yys] + ohErrCode
				if ohn >= 0 && ohn < ohLast {
					ohstate = ohAct[ohn] /* simulate a shift of "error" */
					if ohChk[ohstate] == ohErrCode {
						goto ohstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if ohDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", ohS[ohp].yys)
				}
				ohp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if ohDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", ohTokname(ohtoken))
			}
			if ohtoken == ohEofCode {
				goto ret1
			}
			ohrcvr.char = -1
			ohtoken = -1
			goto ohnewstate /* try again in the same state */
		}
	}

	/* reduction by production ohn */
	if ohDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", ohn, ohStatname(ohstate))
	}

	ohnt := ohn
	ohpt := ohp
	_ = ohpt // guard against "declared and not used"

	ohp -= ohR2[ohn]
	// ohp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if ohp+1 >= len(ohS) {
		nyys := make([]ohSymType, len(ohS)*2)
		copy(nyys, ohS)
		ohS = nyys
	}
	ohVAL = ohS[ohp+1]

	/* consult goto table to find next state */
	ohn = ohR1[ohn]
	ohg := ohPgo[ohn]
	ohj := ohg + ohS[ohp].yys + 1

	if ohj >= ohLast {
		ohstate = ohAct[ohg]
	} else {
		ohstate = ohAct[ohj]
		if ohChk[ohstate] != -ohn {
			ohstate = ohAct[ohg]
		}
	}
	// dummy call; replaced with literal code
	switch ohnt {

	case 4:
		ohDollar = ohS[ohpt-0 : ohpt+1]
		//line grammar.y:47
		{
			ohVAL.c = Null
		}
	case 5:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:49
		{
			ohVAL.c = ohDollar[1].c
			if ohDollar[1].c != Null {
				s := GetLexer(ohlex)
				_, ok := s.yield(ohDollar[1].c, s.label, s.lines, "")
				if !ok {
					return -1
				}
			}
			goto ohstart
		}
	case 6:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:61
		{
			ohVAL.c = List(NewSymbol(ohDollar[2].s), ohDollar[1].c)
		}
	case 7:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:65
		{
			ohVAL.c = List(NewSymbol(ohDollar[2].s), ohDollar[1].c, ohDollar[3].c)
		}
	case 8:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:69
		{
			ohVAL.c = List(NewSymbol(ohDollar[2].s), ohDollar[1].c, ohDollar[3].c)
		}
	case 9:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:73
		{
			ohVAL.c = List(NewSymbol(ohDollar[2].s), ohDollar[1].c, ohDollar[3].c)
		}
	case 10:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:77
		{
			ohVAL.c = List(NewSymbol(ohDollar[2].s), ohDollar[3].c, ohDollar[1].c)
		}
	case 11:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:81
		{
			ohVAL.c = ohDollar[1].c
		}
	case 12:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:83
		{
			ohVAL.c = Null
		}
	case 13:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:85
		{
			if ohDollar[3].c == Null {
				ohVAL.c = ohDollar[2].c
			} else {
				ohVAL.c = Cons(NewSymbol("block"), Cons(ohDollar[2].c, ohDollar[3].c))
			}
		}
	case 18:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:101
		{
			ohVAL.c = Null
		}
	case 19:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:103
		{
			ohVAL.c = ohDollar[2].c
		}
	case 20:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:105
		{
			ohVAL.c = Cons(ohDollar[1].c, Null)
		}
	case 21:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:107
		{
			ohVAL.c = AppendTo(ohDollar[1].c, ohDollar[3].c)
		}
	case 22:
		ohDollar = ohS[ohpt-0 : ohpt+1]
		//line grammar.y:109
		{
			ohVAL.c = Null
		}
	case 23:
		ohDollar = ohS[ohpt-5 : ohpt+1]
		//line grammar.y:111
		{
			lst := List(Cons(NewSymbol(ohDollar[1].s), ohDollar[2].c))
			if ohDollar[4].c != Null {
				lst = JoinTo(lst, ohDollar[4].c)
			}
			if ohDollar[5].c != Null {
				lst = JoinTo(lst, ohDollar[5].c)
			}
			ohVAL.c = lst
		}
	case 24:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:122
		{
			if ohDollar[2].c != Null {
				sym := NewSymbol("_process_substitution_")
				ohVAL.c = JoinTo(Cons(sym, ohDollar[1].c), ohDollar[2].c)
			} else {
				ohVAL.c = ohDollar[1].c
			}
		}
	case 25:
		ohDollar = ohS[ohpt-0 : ohpt+1]
		//line grammar.y:131
		{
			ohVAL.c = Null
		}
	case 26:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:133
		{
			ohVAL.c = ohDollar[1].c
		}
	case 27:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:135
		{
			ohVAL.c = ohDollar[1].c
		}
	case 28:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:137
		{
			ohVAL.c = JoinTo(ohDollar[1].c, ohDollar[2].c)
		}
	case 29:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:141
		{
			ohVAL.c = ohDollar[1].c
		}
	case 30:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:143
		{
			ohVAL.c = Cons(ohDollar[2].c, Null)
		}
	case 31:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:145
		{
			if ohDollar[2].c == Null {
				ohVAL.c = ohDollar[3].c
			} else {
				ohVAL.c = JoinTo(ohDollar[2].c, ohDollar[3].c)
			}
		}
	case 32:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:153
		{
			ohVAL.c = ohDollar[2].c
		}
	case 33:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:157
		{
			ohVAL.c = Null
		}
	case 34:
		ohDollar = ohS[ohpt-4 : ohpt+1]
		//line grammar.y:159
		{
			ohVAL.c = ohDollar[2].c
		}
	case 35:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:161
		{
			if ohDollar[1].c == Null {
				ohVAL.c = ohDollar[1].c
			} else {
				ohVAL.c = Cons(ohDollar[1].c, Null)
			}
		}
	case 36:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:169
		{
			if ohDollar[1].c == Null {
				if ohDollar[3].c == Null {
					ohVAL.c = ohDollar[3].c
				} else {
					ohVAL.c = Cons(ohDollar[3].c, Null)
				}
			} else {
				if ohDollar[3].c == Null {
					ohVAL.c = ohDollar[1].c
				} else {
					ohVAL.c = AppendTo(ohDollar[1].c, ohDollar[3].c)
				}
			}
		}
	case 37:
		ohDollar = ohS[ohpt-0 : ohpt+1]
		//line grammar.y:185
		{
			ohVAL.c = Null
		}
	case 38:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:187
		{
			ohVAL.c = ohDollar[1].c
		}
	case 39:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:189
		{
			s := GetLexer(ohlex)
			s.first = ""

			ohVAL.c = ohDollar[1].c
		}
	case 40:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:196
		{
			ohVAL.c = Cons(ohDollar[1].c, Null)
		}
	case 41:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:198
		{
			ohVAL.c = AppendTo(ohDollar[1].c, ohDollar[2].c)
		}
	case 42:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:200
		{
			ohVAL.c = List(NewSymbol("_splice_"), ohDollar[2].c)
		}
	case 43:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:204
		{
			ohVAL.c = ohDollar[1].c
		}
	case 44:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:206
		{
			ohVAL.c = List(NewSymbol("_backtick_"), ohDollar[2].c)
		}
	case 45:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:210
		{
			ohVAL.c = Cons(ohDollar[1].c, ohDollar[3].c)
		}
	case 46:
		ohDollar = ohS[ohpt-4 : ohpt+1]
		//line grammar.y:214
		{
			value, _ := strconv.ParseUint(ohDollar[3].s, 0, 64)
			ohVAL.c = GetLexer(ohlex).deref(ohDollar[2].s, uintptr(value))
		}
	case 47:
		ohDollar = ohS[ohpt-3 : ohpt+1]
		//line grammar.y:219
		{
			ohVAL.c = ohDollar[2].c
		}
	case 48:
		ohDollar = ohS[ohpt-2 : ohpt+1]
		//line grammar.y:221
		{
			ohVAL.c = Null
		}
	case 49:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:223
		{
			ohVAL.c = ohDollar[1].c
		}
	case 50:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:225
		{
			v, _ := adapted.Unquote(ohDollar[1].s[1:])
			ohVAL.c = NewString(v)
		}
	case 51:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:230
		{
			v, _ := adapted.Unquote(ohDollar[1].s)
			s := NewString(v)
			ohVAL.c = List(NewSymbol("interpolate"), s)
		}
	case 52:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:236
		{
			ohVAL.c = NewString(ohDollar[1].s[1 : len(ohDollar[1].s)-1])
		}
	case 53:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:240
		{
			s := GetLexer(ohlex)
			if s.first == "" {
				s.first = ohDollar[1].s
			}

			ohVAL.c = NewSymbol(ohDollar[1].s)
		}
	case 54:
		ohDollar = ohS[ohpt-1 : ohpt+1]
		//line grammar.y:249
		{
			ohVAL.c = NewSymbol(ohDollar[1].s)
		}
	}

	ohrcvr.n = ohn
	ohrcvr.p = ohp
	goto ohstack /* stack new state and value */
}
