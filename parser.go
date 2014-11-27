//line parser.y:17
package main

import __yyfmt__ "fmt"

//line parser.y:17
import (
	"github.com/michaelmacinnis/liner"
	"strconv"
	"unsafe"
)

type yySymType struct {
	yys int
	c   Cell
	s   string
}

const DEDENT = 57346
const END = 57347
const ERROR = 57348
const INDENT = 57349
const STRING = 57350
const SYMBOL = 57351
const BACKGROUND = 57352
const ORF = 57353
const ANDF = 57354
const PIPE = 57355
const REDIRECT = 57356
const SUBSTITUTE = 57357
const CONS = 57358

var yyToknames = []string{
	"DEDENT",
	"END",
	"ERROR",
	"INDENT",
	"STRING",
	"SYMBOL",
	"BACKGROUND",
	"ORF",
	"ANDF",
	"PIPE",
	"REDIRECT",
	"SUBSTITUTE",
	"^",
	"@",
	"'",
	"`",
	"CONS",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line parser.y:235

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}

type scanner struct {
	process func(Cell)
	task    *Task

	input ReadStringer
	line  []rune

	state  int
	indent int

	cursor int
	start  int

	previous rune
	token    rune

	finished bool
}

const (
	ssStart = iota
	ssAmpersand
	ssBang
	ssBangGreater
	ssColon
	ssComment
	ssGreater
	ssLess
	ssPipe
	ssString
	ssSymbol
)

func (s *scanner) Lex(lval *yySymType) (token int) {
	var operator = map[string]string{
		"!>":  "redirect-stderr",
		"!>>": "append-stderr",
		"!|":  "pipe-stderr",
		"!|+": "channel-stderr",
		"&":   "spawn",
		"&&":  "and",
		"<":   "redirect-stdin",
		"<(":  "substitute-stdout",
		">":   "redirect-stdout",
		">(":  "substitute-stdin",
		">>":  "append-stdout",
		"|":   "pipe-stdout",
		"|+":  "channel-stdout",
		"||":  "or",
	}

	defer func() {
		exists := false

		switch s.token {
		case BACKGROUND, ORF, ANDF, PIPE, REDIRECT, SUBSTITUTE:
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
			if error == liner.ErrPromptAborted {
				s.start = 0
				s.token = ERROR
				break
			}

			runes := []rune(line)
			last := len(runes) - 2
			if last >= 0 && runes[last] == '\r' {
				runes = append(runes[0:last], '\n')
				last--
			}

			if last >= 0 && runes[last] == '\\' {
				runes = runes[0:last]
			}

			if error != nil {
				runes = append(runes, '\n')
				s.finished = true
			}

			if s.start < s.cursor-1 {
				s.line = append(s.line[s.start:s.cursor], runes...)
				s.cursor -= s.start
			} else {
				s.cursor = 0
				s.line = runes
			}
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
			case '\n', '%', '\'', '(', ')', ';', '@', '^', '`', '{', '}':
				s.token = s.line[s.cursor]
			case '&':
				s.state = ssAmpersand
			case '<':
				s.state = ssLess
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
			for s.line[s.cursor] != '\n' ||
				s.line[s.cursor-1] == '\\' {
				s.cursor++

				if s.cursor >= len(s.line) {
					continue main
				}
			}
			s.cursor--
			s.state = ssStart

		case ssGreater:
			s.token = REDIRECT
			if s.line[s.cursor] == '(' {
				s.token = SUBSTITUTE
			} else if s.line[s.cursor] != '>' {
				continue main
			}

		case ssLess:
			s.token = REDIRECT
			if s.line[s.cursor] == '(' {
				s.token = SUBSTITUTE
			} else {
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
			for s.cursor < len(s.line) && s.line[s.cursor] != '"' ||
				s.cursor > 0 && s.line[s.cursor-1] == '\\' {
				s.cursor++
			}
			if s.cursor >= len(s.line) {
				if s.line[s.cursor-1] == '\n' {
					s.line = append(s.line[0:s.cursor-1], []rune("\\n")...)
				}
				continue main
			}
			s.token = STRING

		case ssSymbol:
			switch s.line[s.cursor] {
			case '\n', '%', '&', '\'', '(', ')', ';',
				'<', '@', '^', '`', '{', '|', '}',
				'\t', ' ', '"', '#', ':', '>':
				s.token = SYMBOL
				continue main
			}

		}
		s.cursor++

		if s.token == '\n' {
			switch s.previous {
			case ORF, ANDF, PIPE, REDIRECT:
				s.token = 0
			}
		}
	}

	return int(s.token)
}

func (s *scanner) Error(msg string) {
	println(msg)
}

func Parse(t *Task, r ReadStringer, p func(Cell)) {
	s := new(scanner)

	s.process = p
	s.task = t

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

//line yacctab:1
var yyExca = []int{
	-1, 0,
	8, 16,
	9, 16,
	17, 16,
	18, 16,
	19, 16,
	21, 5,
	24, 16,
	25, 16,
	27, 16,
	28, 16,
	-2, 0,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 7,
	10, 14,
	11, 14,
	12, 14,
	13, 14,
	14, 14,
	15, 14,
	21, 14,
	22, 14,
	-2, 17,
	-1, 10,
	1, 1,
	8, 16,
	9, 16,
	17, 16,
	18, 16,
	19, 16,
	21, 5,
	24, 16,
	25, 16,
	27, 16,
	28, 16,
	-2, 0,
	-1, 47,
	21, 34,
	-2, 16,
	-1, 68,
	21, 34,
	-2, 16,
}

const yyNprod = 49
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 122

var yyAct = []int{

	5, 8, 18, 62, 7, 69, 21, 11, 12, 13,
	14, 15, 16, 33, 34, 35, 9, 37, 9, 65,
	39, 71, 36, 40, 9, 45, 42, 60, 17, 52,
	68, 48, 49, 50, 53, 9, 30, 31, 4, 47,
	43, 10, 44, 56, 44, 24, 25, 26, 63, 59,
	57, 58, 22, 23, 64, 27, 28, 66, 51, 9,
	67, 29, 20, 61, 30, 31, 46, 30, 31, 63,
	70, 19, 72, 24, 25, 26, 24, 25, 26, 17,
	22, 23, 41, 27, 28, 55, 27, 28, 11, 12,
	13, 14, 15, 16, 11, 12, 13, 14, 15, 16,
	54, 13, 14, 15, 16, 14, 15, 16, 15, 16,
	3, 38, 6, 2, 1, 0, 0, 0, 0, 0,
	0, 32,
}
var yyPact = []int{

	36, -1000, 20, -1000, -1000, 84, -1000, 5, 28, -1000,
	36, -1000, -7, -7, -7, 59, -7, -1000, -7, 28,
	-1000, 24, 28, 18, 59, 59, 59, 49, 12, -1000,
	-1000, -1000, -1000, 89, 92, 94, 24, 78, -1000, -1000,
	56, -1000, 24, 59, 59, -1000, 28, 1, 22, 22,
	22, 45, -3, -1000, -1000, -7, -1000, 22, -1000, -1000,
	-1000, 9, -1000, 84, -22, -1000, -1000, 56, -5, -1000,
	-1000, -1000, -1000,
}
var yyPgo = []int{

	0, 114, 113, 110, 0, 6, 112, 4, 1, 2,
	111, 85, 71, 62, 66, 63, 3, 61,
}
var yyR1 = []int{

	0, 1, 2, 2, 3, 3, 3, 4, 4, 4,
	4, 4, 4, 4, 6, 6, 8, 8, 7, 7,
	10, 10, 11, 11, 9, 9, 9, 13, 13, 13,
	14, 14, 15, 15, 16, 16, 12, 12, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 17, 17,
}
var yyR2 = []int{

	0, 2, 1, 3, 1, 0, 1, 2, 3, 3,
	3, 3, 4, 1, 1, 3, 0, 1, 1, 2,
	1, 3, 1, 3, 1, 2, 1, 2, 3, 2,
	2, 4, 1, 3, 0, 1, 1, 2, 3, 2,
	2, 2, 3, 4, 3, 2, 1, 1, 1,
}
var yyChk = []int{

	-1000, -1, -2, -3, 2, -4, -6, -7, -8, 23,
	21, 10, 11, 12, 13, 14, 15, 23, -9, -12,
	-13, -5, 24, 25, 17, 18, 19, 27, 28, -17,
	8, 9, -3, -4, -4, -4, -5, -4, -10, -8,
	-7, -13, -5, 16, 20, -9, -14, 21, -5, -5,
	-5, 9, -4, 22, 22, -11, -9, -5, -5, -9,
	26, -15, -16, -4, 9, 22, -8, -7, 21, 27,
	-9, 26, -16,
}
var yyDef = []int{

	-2, -2, 0, 2, 4, 6, 13, -2, 0, 18,
	-2, 7, 16, 16, 16, 0, 16, 19, 16, 24,
	26, 36, 0, 0, 0, 0, 0, 0, 16, 46,
	47, 48, 3, 8, 9, 10, 11, 0, 15, 20,
	17, 25, 37, 0, 0, 27, 29, -2, 39, 40,
	41, 0, 0, 45, 12, 16, 22, 38, 42, 28,
	30, 0, 32, 35, 0, 44, 21, 17, -2, 43,
	23, 31, 33,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	21, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 27, 3, 18,
	28, 22, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 24, 23,
	3, 3, 3, 3, 17, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 16, 3, 19, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 25, 3, 26,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 20,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
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

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	startyyVAL := yyVAL
start:
	yyVAL = startyyVAL

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
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
		yychar = yylex1(yylex, &yylval)
		if yychar == yyTok2[ERROR-yyPrivate] {
			goto start
		}

	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
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
			yychar = yylex1(yylex, &yylval)
			if yychar == yyTok2[ERROR-yyPrivate] {
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
			if yyn < 0 || yyn == yychar {
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
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
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
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
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

	case 5:
		//line parser.y:42
		{
			yyVAL.c = Null
		}
	case 6:
		//line parser.y:44
		{
			yyVAL.c = yyS[yypt-0].c
			if yyS[yypt-0].c != Null {
				yylex.(*scanner).process(yyS[yypt-0].c)
			}
			goto start
		}
	case 7:
		//line parser.y:52
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-0].s), yyS[yypt-1].c)
		}
	case 8:
		//line parser.y:56
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 9:
		//line parser.y:60
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 10:
		//line parser.y:64
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 11:
		//line parser.y:68
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-0].c, yyS[yypt-2].c)
		}
	case 12:
		//line parser.y:72
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-2].s), yyS[yypt-1].c, yyS[yypt-3].c)
		}
	case 13:
		//line parser.y:76
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 14:
		//line parser.y:78
		{
			yyVAL.c = Null
		}
	case 15:
		//line parser.y:80
		{
			if yyS[yypt-0].c == Null {
				yyVAL.c = yyS[yypt-1].c
			} else {
				yyVAL.c = Cons(NewSymbol("block"), Cons(yyS[yypt-1].c, yyS[yypt-0].c))
			}
		}
	case 20:
		//line parser.y:96
		{
			yyVAL.c = Null
		}
	case 21:
		//line parser.y:98
		{
			yyVAL.c = yyS[yypt-1].c
		}
	case 22:
		//line parser.y:100
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 23:
		//line parser.y:102
		{
			yyVAL.c = AppendTo(yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 24:
		//line parser.y:104
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 25:
		//line parser.y:106
		{
			yyVAL.c = JoinTo(yyS[yypt-1].c, yyS[yypt-0].c)
		}
	case 26:
		//line parser.y:110
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 27:
		//line parser.y:112
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 28:
		//line parser.y:114
		{
			if yyS[yypt-1].c == Null {
				yyVAL.c = yyS[yypt-0].c
			} else {
				yyVAL.c = JoinTo(yyS[yypt-1].c, yyS[yypt-0].c)
			}
		}
	case 29:
		//line parser.y:122
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 30:
		//line parser.y:126
		{
			yyVAL.c = Null
		}
	case 31:
		//line parser.y:128
		{
			yyVAL.c = yyS[yypt-2].c
		}
	case 32:
		//line parser.y:130
		{
			if yyS[yypt-0].c == Null {
				yyVAL.c = yyS[yypt-0].c
			} else {
				yyVAL.c = Cons(yyS[yypt-0].c, Null)
			}
		}
	case 33:
		//line parser.y:138
		{
			if yyS[yypt-2].c == Null {
				if yyS[yypt-0].c == Null {
					yyVAL.c = yyS[yypt-0].c
				} else {
					yyVAL.c = Cons(yyS[yypt-0].c, Null)
				}
			} else {
				if yyS[yypt-0].c == Null {
					yyVAL.c = yyS[yypt-2].c
				} else {
					yyVAL.c = AppendTo(yyS[yypt-2].c, yyS[yypt-0].c)
				}
			}
		}
	case 34:
		//line parser.y:154
		{
			yyVAL.c = Null
		}
	case 35:
		//line parser.y:156
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 36:
		//line parser.y:158
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 37:
		//line parser.y:160
		{
			yyVAL.c = AppendTo(yyS[yypt-1].c, yyS[yypt-0].c)
		}
	case 38:
		//line parser.y:162
		{
			s := Cons(NewString(yylex.(*scanner).task, ""), NewSymbol("join"))
			yyVAL.c = List(s, yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 39:
		//line parser.y:167
		{
			yyVAL.c = List(NewSymbol("splice"), yyS[yypt-0].c)
		}
	case 40:
		//line parser.y:171
		{
			yyVAL.c = List(NewSymbol("quote"), yyS[yypt-0].c)
		}
	case 41:
		//line parser.y:175
		{
			yyVAL.c = List(NewSymbol("backtick"), yyS[yypt-0].c)
		}
	case 42:
		//line parser.y:179
		{
			yyVAL.c = Cons(yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 43:
		//line parser.y:183
		{
			kind := yyS[yypt-2].s
			value, _ := strconv.ParseUint(yyS[yypt-1].s, 0, 64)

			addr := uintptr(value)

			switch {
			case kind == "bound":
				yyVAL.c = (*Bound)(unsafe.Pointer(addr))
			case kind == "builtin":
				yyVAL.c = (*Builtin)(unsafe.Pointer(addr))
			case kind == "channel":
				yyVAL.c = (*Channel)(unsafe.Pointer(addr))
			case kind == "constant":
				yyVAL.c = (*Constant)(unsafe.Pointer(addr))
			case kind == "continuation":
				yyVAL.c = (*Continuation)(unsafe.Pointer(addr))
			case kind == "env":
				yyVAL.c = (*Env)(unsafe.Pointer(addr))
			case kind == "method":
				yyVAL.c = (*Method)(unsafe.Pointer(addr))
			case kind == "object":
				yyVAL.c = (*Object)(unsafe.Pointer(addr))
			case kind == "pipe":
				yyVAL.c = (*Pipe)(unsafe.Pointer(addr))
			case kind == "scope":
				yyVAL.c = (*Scope)(unsafe.Pointer(addr))
			case kind == "syntax":
				yyVAL.c = (*Syntax)(unsafe.Pointer(addr))
			case kind == "task":
				yyVAL.c = (*Task)(unsafe.Pointer(addr))
			case kind == "unbound":
				yyVAL.c = (*Unbound)(unsafe.Pointer(addr))
			case kind == "variable":
				yyVAL.c = (*Variable)(unsafe.Pointer(addr))

			default:
				yyVAL.c = Null
			}

		}
	case 44:
		//line parser.y:225
		{
			yyVAL = yyS[yypt-1]
		}
	case 45:
		//line parser.y:227
		{
			yyVAL.c = Null
		}
	case 46:
		//line parser.y:229
		{
			yyVAL = yyS[yypt-0]
		}
	case 47:
		//line parser.y:231
		{
			yyVAL.c = NewString(yylex.(*scanner).task, yyS[yypt-0].s[1:len(yyS[yypt-0].s)-1])
		}
	case 48:
		//line parser.y:233
		{
			yyVAL.c = NewSymbol(yyS[yypt-0].s)
		}
	}
	goto yystack /* stack new state and value */
}
