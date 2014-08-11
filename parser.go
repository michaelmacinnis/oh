//line parser.y:15
package main

import __yyfmt__ "fmt"

//line parser.y:15
import (
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
const CONS = 57357

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
	" @",
	" '",
	" `",
	"CONS",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line parser.y:224

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
		">":   "redirect-stdout",
		">>":  "append-stdout",
		"|":   "pipe-stdout",
		"|+":  "channel-stdout",
		"||":  "or",
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
			} else if len(line) > 1 && line[len(line)-2:] == "\x03\n" {
				s.start = 0
				s.token = ERROR
				break
			}

			if s.start < s.cursor-1 {
				s.line = append(s.line[s.start:s.cursor], []rune(line)...)
				s.cursor -= s.start
			} else {
				s.cursor = 0
				s.line = []rune(line)
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
			for s.line[s.cursor+1] != '\n' ||
				s.line[s.cursor] == '\\' {
				s.cursor++

				if s.cursor+1 >= len(s.line) {
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
				'<', '@', '`', '{', '|', '}',
				'\t', ' ', '"', '#', ':', '>':
				if s.line[s.cursor-1] != '\\' {
					s.token = SYMBOL
					continue main
				}
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
	15, 16,
	16, 16,
	17, 16,
	19, 6,
	22, 16,
	23, 16,
	25, 16,
	26, 16,
	-2, 0,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 8,
	10, 14,
	11, 14,
	12, 14,
	13, 14,
	14, 14,
	19, 14,
	27, 14,
	-2, 17,
	-1, 11,
	1, 1,
	8, 16,
	9, 16,
	15, 16,
	16, 16,
	17, 16,
	19, 6,
	22, 16,
	23, 16,
	25, 16,
	26, 16,
	-2, 0,
	-1, 46,
	19, 34,
	-2, 16,
	-1, 65,
	19, 34,
	-2, 16,
}

const yyNprod = 48
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 102

var yyAct = []int{

	6, 9, 19, 59, 8, 31, 32, 22, 10, 66,
	10, 68, 25, 26, 27, 34, 35, 36, 18, 23,
	24, 39, 28, 29, 40, 37, 44, 18, 42, 10,
	51, 10, 57, 47, 48, 49, 65, 52, 5, 46,
	21, 11, 43, 54, 17, 31, 32, 60, 56, 61,
	50, 55, 25, 26, 27, 63, 4, 10, 64, 23,
	24, 41, 28, 29, 16, 17, 60, 67, 3, 69,
	13, 14, 15, 16, 17, 31, 32, 15, 16, 17,
	33, 12, 25, 26, 27, 30, 58, 62, 45, 20,
	53, 38, 28, 29, 13, 14, 15, 16, 17, 7,
	2, 1,
}
var yyPact = []int{

	36, -1000, 22, -1000, 75, -1000, 84, -1000, 6, 37,
	-1000, 36, -1000, -1000, -11, -11, -11, 67, -1000, -11,
	37, -1000, 24, 37, 20, 67, 67, 67, 41, 10,
	-1000, -1000, -1000, -1000, 65, 51, 30, 24, -1000, -1000,
	-3, -1000, 24, 67, -1000, 37, 8, 24, 24, 24,
	40, 60, -1000, -11, -1000, -1000, -1000, -1000, 17, -1000,
	84, -16, -1000, -1000, -3, -13, -1000, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 101, 100, 68, 0, 7, 99, 4, 1, 2,
	91, 90, 89, 40, 88, 86, 3, 85,
}
var yyR1 = []int{

	0, 1, 2, 2, 3, 3, 3, 3, 4, 4,
	4, 4, 4, 4, 6, 6, 8, 8, 7, 7,
	10, 10, 11, 11, 9, 9, 9, 13, 13, 13,
	14, 14, 15, 15, 16, 16, 12, 12, 5, 5,
	5, 5, 5, 5, 5, 5, 17, 17,
}
var yyR2 = []int{

	0, 2, 1, 3, 2, 1, 0, 1, 2, 3,
	3, 3, 3, 1, 1, 3, 0, 1, 1, 2,
	1, 3, 1, 3, 1, 2, 1, 2, 3, 2,
	2, 4, 1, 3, 0, 1, 1, 2, 2, 2,
	2, 3, 4, 3, 2, 1, 1, 1,
}
var yyChk = []int{

	-1000, -1, -2, -3, 20, 2, -4, -6, -7, -8,
	21, 19, 6, 10, 11, 12, 13, 14, 21, -9,
	-12, -13, -5, 22, 23, 15, 16, 17, 25, 26,
	-17, 8, 9, -3, -4, -4, -4, -5, -10, -8,
	-7, -13, -5, 18, -9, -14, 19, -5, -5, -5,
	9, -4, 27, -11, -9, -5, -9, 24, -15, -16,
	-4, 9, 27, -8, -7, 19, 25, -9, 24, -16,
}
var yyDef = []int{

	-2, -2, 0, 2, 0, 5, 7, 13, -2, 0,
	18, -2, 4, 8, 16, 16, 16, 0, 19, 16,
	24, 26, 36, 0, 0, 0, 0, 0, 0, 16,
	45, 46, 47, 3, 9, 10, 11, 12, 15, 20,
	17, 25, 37, 0, 27, 29, -2, 38, 39, 40,
	0, 0, 44, 16, 22, 41, 28, 30, 0, 32,
	35, 0, 43, 21, 17, -2, 42, 23, 31, 33,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 20,
	19, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 25, 3, 16,
	26, 27, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 22, 21,
	3, 3, 3, 3, 15, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 17, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 23, 3, 24,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 18,
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

start:
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
                if yychar == 6 {
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
                	if yychar == 6 {
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

	case 6:
		//line parser.y:41
		{
			yyVAL.c = Null
		}
	case 7:
		//line parser.y:43
		{
			yyVAL.c = yyS[yypt-0].c
			if yyS[yypt-0].c != Null {
				yylex.(*scanner).process(yyS[yypt-0].c)
			}
		}
	case 8:
		//line parser.y:50
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-0].s), yyS[yypt-1].c)
		}
	case 9:
		//line parser.y:54
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 10:
		//line parser.y:58
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 11:
		//line parser.y:62
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 12:
		//line parser.y:66
		{
			yyVAL.c = List(NewSymbol(yyS[yypt-1].s), yyS[yypt-0].c, yyS[yypt-2].c)
		}
	case 13:
		//line parser.y:70
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 14:
		//line parser.y:72
		{
			yyVAL.c = Null
		}
	case 15:
		//line parser.y:74
		{
			if yyS[yypt-0].c == Null {
				yyVAL.c = yyS[yypt-1].c
			} else {
				yyVAL.c = Cons(NewSymbol("block"), Cons(yyS[yypt-1].c, yyS[yypt-0].c))
			}
		}
	case 20:
		//line parser.y:90
		{
			yyVAL.c = Null
		}
	case 21:
		//line parser.y:92
		{
			yyVAL.c = yyS[yypt-1].c
		}
	case 22:
		//line parser.y:94
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 23:
		//line parser.y:96
		{
			yyVAL.c = AppendTo(yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 24:
		//line parser.y:98
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 25:
		//line parser.y:100
		{
			yyVAL.c = JoinTo(yyS[yypt-1].c, yyS[yypt-0].c)
		}
	case 26:
		//line parser.y:104
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 27:
		//line parser.y:106
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 28:
		//line parser.y:108
		{
			if yyS[yypt-1].c == Null {
				yyVAL.c = yyS[yypt-0].c
			} else {
				yyVAL.c = JoinTo(yyS[yypt-1].c, yyS[yypt-0].c)
			}
		}
	case 29:
		//line parser.y:116
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 30:
		//line parser.y:120
		{
			yyVAL.c = Null
		}
	case 31:
		//line parser.y:122
		{
			yyVAL.c = yyS[yypt-2].c
		}
	case 32:
		//line parser.y:124
		{
			if yyS[yypt-0].c == Null {
				yyVAL.c = yyS[yypt-0].c
			} else {
				yyVAL.c = Cons(yyS[yypt-0].c, Null)
			}
		}
	case 33:
		//line parser.y:132
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
		//line parser.y:148
		{
			yyVAL.c = Null
		}
	case 35:
		//line parser.y:150
		{
			yyVAL.c = yyS[yypt-0].c
		}
	case 36:
		//line parser.y:152
		{
			yyVAL.c = Cons(yyS[yypt-0].c, Null)
		}
	case 37:
		//line parser.y:154
		{
			yyVAL.c = AppendTo(yyS[yypt-1].c, yyS[yypt-0].c)
		}
	case 38:
		//line parser.y:156
		{
			yyVAL.c = List(NewSymbol("splice"), yyS[yypt-0].c)
		}
	case 39:
		//line parser.y:160
		{
			yyVAL.c = List(NewSymbol("quote"), yyS[yypt-0].c)
		}
	case 40:
		//line parser.y:164
		{
			yyVAL.c = List(NewSymbol("backtick"), yyS[yypt-0].c)
		}
	case 41:
		//line parser.y:168
		{
			yyVAL.c = Cons(yyS[yypt-2].c, yyS[yypt-0].c)
		}
	case 42:
		//line parser.y:172
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
	case 43:
		//line parser.y:214
		{
			yyVAL = yyS[yypt-1]
		}
	case 44:
		//line parser.y:216
		{
			yyVAL.c = Null
		}
	case 45:
		//line parser.y:218
		{
			yyVAL = yyS[yypt-0]
		}
	case 46:
		//line parser.y:220
		{
			yyVAL.c = NewString(yylex.(*scanner).task, yyS[yypt-0].s[1:len(yyS[yypt-0].s)-1])
		}
	case 47:
		//line parser.y:222
		{
			yyVAL.c = NewSymbol(yyS[yypt-0].s)
		}
	}
	goto yystack /* stack new state and value */
}
