// Released under an MIT license. See LICENSE.

package cell

import (
	"bufio"
	"fmt"
	"github.com/michaelmacinnis/adapted"
	"math/big"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Cell interface {
	Bool() bool
	Equal(c Cell) bool
	String() string
}

type Conduit interface {
	Close()
	ReaderClose()
	ReadLine() Cell
	Read(bool, Parser, Thrower) Cell
	WriterClose()
	Write(c Cell)
}

type Number interface {
	Atom

	Greater(c Cell) bool
	Less(c Cell) bool

	Add(c Cell) Number
	Divide(c Cell) Number
	Modulo(c Cell) Number
	Multiply(c Cell) Number
	Subtract(c Cell) Number
}

type Parser func(
	ReadStringer, Thrower, *os.File, string,
	func(Cell, string, int, string) (Cell, bool),
) bool

type ReadStringer interface {
	ReadString(delim byte) (line string, err error)
}

type Reference interface {
	Copy() Reference
	Get() Cell
	Set(c Cell)
}

type Thrower interface {
	Throw(filename string, lineno int, message string)
	SetFile(filename string)
	SetLine(lineno int)
}

var (
	Null  Cell
	False *Boolean
	True  *Boolean

	max  *big.Rat
	min  *big.Rat
	num  [512]*Integer
	numl = &sync.RWMutex{}
	one  *big.Rat
	rat  [512]*Rational
	ratl = &sync.RWMutex{}
	sym  = map[string]*Symbol{}
	syml = &sync.RWMutex{}
	zip  *big.Rat
)

func CacheSymbols(symbols ...string) {
	for _, v := range symbols {
		sym[v] = NewSymbol(v)
	}
}

/* Convert Cell into a Conduit. (Return nil if not possible). */
func asConduit(o Cell) Conduit {
	if c, ok := o.(Conduit); ok {
		return c
	}

	return nil
}

func init() {
	max = big.NewRat(255, 1)
	min = big.NewRat(-256, 1)

	one = big.NewRat(1, 1)
	zip = big.NewRat(0, 1)
	rat[257] = NewRational(one)
	rat[256] = NewRational(zip)

	pair := new(Pair)
	pair.car = pair
	pair.cdr = pair

	Null = Cell(pair)

	F := Boolean(false)
	False = &F

	T := Boolean(true)
	True = &T
}

func ratmod(x, y *big.Rat) *big.Rat {
	if x.IsInt() && y.IsInt() {
		return new(big.Rat).SetInt(new(big.Int).Mod(x.Num(), y.Num()))
	}

	panic("operation not permitted")
}

/* Boolean cell definition. */

type Boolean bool

func IsBoolean(c Cell) bool {
	switch c.(type) {
	case *Boolean:
		return true
	}
	return false
}

func NewBoolean(v bool) *Boolean {
	if v {
		return True
	}
	return False
}

func (b *Boolean) Bool() bool {
	return b == True
}

func (b *Boolean) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return bool(*b) == a.Bool()
	}
	return false
}

func (b *Boolean) String() string {
	if b == True {
		return "true"
	}
	return "false"
}

func (b *Boolean) Float() float64 {
	if b == True {
		return 1.0
	}
	return 0.0
}

func (b *Boolean) Int() int64 {
	if b == True {
		return 1
	}
	return 0
}

func (b *Boolean) Rat() *big.Rat {
	if b == True {
		return one
	}
	return zip
}

/* Channel cell definition. */

type Channel chan Cell

func IsChannel(c Cell) bool {
	conduit := asConduit(c)
	if conduit == nil {
		return false
	}

	switch conduit.(type) {
	case *Channel:
		return true
	}
	return false
}

func NewChannel(cap int) *Channel {
	c := Channel(make(chan Cell, cap))
	return &c
}

func (ch *Channel) Bool() bool {
	return true
}

func (ch *Channel) Equal(c Cell) bool {
	return ch == c
}

func (ch *Channel) String() string {
	return fmt.Sprintf("%%channel %p%%", ch)
}

func (ch *Channel) Close() {
	ch.WriterClose()
}

func (ch *Channel) ReaderClose() {
	return
}

func (ch *Channel) Read(interactive bool, p Parser, t Thrower) Cell {
	v := <-(chan Cell)(*ch)
	if v == nil {
		return Null
	}
	return v
}

func (ch *Channel) ReadLine() Cell {
	v := <-(chan Cell)(*ch)
	if v == nil {
		return False
	}
	return NewString(v.String())
}

func (ch *Channel) WriterClose() {
	close((chan Cell)(*ch))
}

func (ch *Channel) Write(c Cell) {
	(chan Cell)(*ch) <- c
}

/* Constant cell definition. */

type Constant struct {
	v Cell
}

func NewConstant(v Cell) *Constant {
	ct := new(Constant)

	ct.v = v

	return ct
}

func (ct *Constant) Copy() Reference {
	return NewConstant(ct.Get())
}

func (ct *Constant) Get() Cell {
	return ct.v
}

func (ct *Constant) Set(c Cell) {
	panic("constant cannot be set")
}

/* Env definition. */

type Env struct {
	*sync.RWMutex
	hash map[string]Reference
	prev *Env
}

func NewEnv(prev *Env) *Env {
	return &Env{&sync.RWMutex{}, make(map[string]Reference), prev}
}

func (e *Env) Access(key Cell) Reference {
	for env := e; env != nil; env = env.prev {
		env.RLock()
		if value, ok := env.hash[key.String()]; ok {
			env.RUnlock()
			return value
		}
		env.RUnlock()
	}

	return nil
}

func (e *Env) Add(key Cell, value Cell) {
	e.Lock()
	defer e.Unlock()

	e.hash[key.String()] = NewVariable(value)
}

func (e *Env) Complete(simple bool, word string) []string {
	p := e.Prefixed(simple, word)

	cl := make([]string, 0, len(p))

	for k := range p {
		cl = append(cl, k)
	}

	if e.prev != nil {
		cl = append(cl, e.prev.Complete(simple, word)...)
	}

	return cl
}

func (e *Env) Copy() *Env {
	e.RLock()
	defer e.RUnlock()

	if e == nil {
		return nil
	}

	fresh := NewEnv(e.prev.Copy())

	for k, v := range e.hash {
		fresh.hash[k] = v.Copy()
	}

	return fresh
}

func (e *Env) Empty() bool {
	return len(e.hash) == 0
}

func (e *Env) Prefixed(simple bool, prefix string) map[string]Cell {
	e.RLock()
	defer e.RUnlock()

	r := map[string]Cell{}

	for k, ref := range e.hash {
		if strings.HasPrefix(k, prefix) {
			v := ref.Get()
			if !simple || IsSimple(v) {
				r[k] = ref.Get()
			}
		}
	}

	return r
}

func (e *Env) Prev() *Env {
	return e.prev
}

func (e *Env) Remove(key Cell) bool {
	e.Lock()
	defer e.Unlock()

	k := key.String()
	_, ok := e.hash[k]
	if ok {
		delete(e.hash, k)
	}
	return ok
}

/* Float cell definition. */

type Float float64

func IsFloat(c Cell) bool {
	switch c.(type) {
	case *Float:
		return true
	}
	return false
}

func NewFloat(v float64) *Float {
	f := Float(v)
	return &f
}

func (f *Float) Bool() bool {
	return *f != 0
}

func (f *Float) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return f.Rat().Cmp(a.Rat()) == 0
	}
	return false
}

func (f *Float) String() string {
	return strconv.FormatFloat(float64(*f), 'g', -1, 64)
}

func (f *Float) Float() float64 {
	return float64(*f)
}

func (f *Float) Int() int64 {
	return int64(*f)
}

func (f *Float) Rat() *big.Rat {
	return new(big.Rat).SetFloat64(float64(*f))
}

func (f *Float) Greater(c Cell) bool {
	return f.Rat().Cmp(c.(Atom).Rat()) > 0
}

func (f *Float) Less(c Cell) bool {
	return f.Rat().Cmp(c.(Atom).Rat()) < 0
}

func (f *Float) Add(c Cell) Number {
	return NewRational(new(big.Rat).Add(f.Rat(), c.(Atom).Rat()))
}

func (f *Float) Divide(c Cell) Number {
	return NewRational(new(big.Rat).Quo(f.Rat(), c.(Atom).Rat()))
}

func (f *Float) Modulo(c Cell) Number {
	return NewRational(ratmod(f.Rat(), c.(Atom).Rat()))
}

func (f *Float) Multiply(c Cell) Number {
	return NewRational(new(big.Rat).Mul(f.Rat(), c.(Atom).Rat()))
}

func (f *Float) Subtract(c Cell) Number {
	return NewRational(new(big.Rat).Sub(f.Rat(), c.(Atom).Rat()))
}

/* Integer cell definition. */

type Integer int64

func IsInteger(c Cell) bool {
	switch c.(type) {
	case *Integer:
		return true
	}
	return false
}

func NewInteger(v int64) *Integer {
	if -256 <= v && v <= 255 {
		n := v + 256

		numl.RLock()
		p := num[n]
		numl.RUnlock()

		if p == nil {
			i := Integer(v)
			p = &i

			numl.Lock()
			num[n] = p
			numl.Unlock()
		}

		return p
	}

	i := Integer(v)
	return &i
}

func (i *Integer) Bool() bool {
	return *i != 0
}

func (i *Integer) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return i.Rat().Cmp(a.Rat()) == 0
	}
	return false
}

func (i *Integer) String() string {
	return strconv.FormatInt(int64(*i), 10)
}

func (i *Integer) Float() float64 {
	return float64(*i)
}

func (i *Integer) Int() int64 {
	return int64(*i)
}

func (i *Integer) Rat() *big.Rat {
	return big.NewRat(int64(*i), 1)
}

func (i *Integer) Greater(c Cell) bool {
	return i.Rat().Cmp(c.(Atom).Rat()) > 0
}

func (i *Integer) Less(c Cell) bool {
	return i.Rat().Cmp(c.(Atom).Rat()) < 0
}

func (i *Integer) Add(c Cell) Number {
	return NewRational(new(big.Rat).Add(i.Rat(), c.(Atom).Rat()))
}

func (i *Integer) Divide(c Cell) Number {
	return NewRational(new(big.Rat).Quo(i.Rat(), c.(Atom).Rat()))
}

func (i *Integer) Modulo(c Cell) Number {
	return NewRational(ratmod(i.Rat(), c.(Atom).Rat()))
}

func (i *Integer) Multiply(c Cell) Number {
	return NewRational(new(big.Rat).Mul(i.Rat(), c.(Atom).Rat()))
}

func (i *Integer) Subtract(c Cell) Number {
	return NewRational(new(big.Rat).Sub(i.Rat(), c.(Atom).Rat()))
}

/* Pair cell definition. */

type Pair struct {
	car Cell
	cdr Cell
}

func IsCons(c Cell) bool {
	switch c.(type) {
	case *Pair:
		return true
	}
	return false
}

func Cons(h, t Cell) Cell {
	return &Pair{car: h, cdr: t}
}

func (p *Pair) Bool() bool {
	return p != Null
}

func (p *Pair) Equal(c Cell) bool {
	if p == Null && c == Null {
		return true
	}
	return p.car.Equal(Car(c)) && p.cdr.Equal(Cdr(c))
}

func (p *Pair) String() (s string) {
	s = ""

	if IsCons(p.car) && IsCons(Cdr(p.car)) {
		s += "("
	}

	if p.car != Null {
		s += p.car.String()
	}

	if IsCons(p.car) && IsCons(Cdr(p.car)) {
		s += ")"
	}

	if IsCons(p.cdr) {
		if p.cdr == Null {
			return s
		}

		s += " "
	} else {
		s += "::"
	}

	s += p.cdr.String()

	return s
}

/* Pipe cell definition. */

type Pipe struct {
	b *bufio.Reader
	c chan Cell
	d chan bool
	r *os.File
	w *os.File
}

func IsPipe(c Cell) bool {
	conduit := asConduit(c)
	if conduit == nil {
		return false
	}

	switch conduit.(type) {
	case *Pipe:
		return true
	}
	return false
}

func NewPipe(r *os.File, w *os.File) *Pipe {
	p := &Pipe{
		b: nil,
		c: nil,
		d: nil,
		r: r,
		w: w,
	}

	if r == nil && w == nil {
		var err error

		if p.r, p.w, err = os.Pipe(); err != nil {
			p.r, p.w = nil, nil
		}
	}

	runtime.SetFinalizer(p, (*Pipe).Close)

	return p
}

func (p *Pipe) Bool() bool {
	return true
}

func (p *Pipe) Equal(c Cell) bool {
	return p == c
}

func (p *Pipe) String() string {
	return fmt.Sprintf("%%pipe %p%%", p)
}

func (p *Pipe) Close() {
	if p.r != nil && len(p.r.Name()) > 0 {
		p.ReaderClose()
	}

	if p.w != nil && len(p.w.Name()) > 0 {
		p.WriterClose()
	}
}

func (p *Pipe) reader() *bufio.Reader {
	if p.b == nil {
		p.b = bufio.NewReader(p.r)
	}

	return p.b
}

func (p *Pipe) ReaderClose() {
	if p.r != nil {
		p.r.Close()
		p.r = nil
	}
}

func (p *Pipe) Read(interactive bool, parse Parser, t Thrower) Cell {
	if p.r == nil {
		return Null
	}

	if p.d == nil {
		p.d = make(chan bool)
	} else {
		p.d <- true
	}

	if p.c == nil {
		p.c = make(chan Cell)
		go func() {
			var f *os.File = nil
			if interactive && p.r == os.Stdin {
				f = p.r
			}
			parse(
				p.reader(), t, f, p.r.Name(),
				func(c Cell, f string, l int, u string) (Cell, bool) {
					t.SetLine(l)
					p.c <- c
					<-p.d
					return nil, true
				},
			)
			p.d = nil
			p.c <- Null
			p.c = nil
		}()
	}

	return <-p.c
}

func (p *Pipe) ReadLine() Cell {
	s, err := p.reader().ReadString('\n')
	if err != nil && len(s) == 0 {
		p.b = nil
		return Null
	}

	return NewString(strings.TrimRight(s, "\n"))
}

func (p *Pipe) WriterClose() {
	if p.w != nil {
		p.w.Close()
		p.w = nil
	}
}

func (p *Pipe) Write(c Cell) {
	if p.w == nil {
		panic("write to closed pipe")
	}

	fmt.Fprintln(p.w, c)
}

/* Pipe-specific functions */

func (p *Pipe) ReadFd() *os.File {
	return p.r
}

func (p *Pipe) WriteFd() *os.File {
	return p.w
}

/* Rational cell definition. */

type Rational big.Rat

func IsRational(c Cell) bool {
	switch c.(type) {
	case *Rational:
		return true
	}
	return false
}

func NewRational(r *big.Rat) *Rational {
	if !r.IsInt() || r.Cmp(min) < 0 || r.Cmp(max) > 0 {
		return (*Rational)(r)
	}

	n := r.Num().Int64()
	i := n + 256

	ratl.RLock()
	p := rat[i]
	ratl.RUnlock()

	if p == nil {
		p = (*Rational)(r)

		ratl.Lock()
		rat[i] = p
		ratl.Unlock()
	}

	return p
}

func (r *Rational) Bool() bool {
	return (*big.Rat)(r).Cmp(zip) != 0
}

func (r *Rational) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return (*big.Rat)(r).Cmp(a.Rat()) == 0
	}
	return false
}

func (r *Rational) String() string {
	return (*big.Rat)(r).RatString()
}

func (r *Rational) Float() float64 {
	f, _ := (*big.Rat)(r).Float64()
	return f
}

func (r *Rational) Int() int64 {
	n := (*big.Rat)(r).Num()
	d := (*big.Rat)(r).Denom()
	return new(big.Int).Div(n, d).Int64()
}

func (r *Rational) Rat() *big.Rat {
	return (*big.Rat)(r)
}

func (r *Rational) Greater(c Cell) bool {
	return (*big.Rat)(r).Cmp(c.(Atom).Rat()) > 0
}

func (r *Rational) Less(c Cell) bool {
	return (*big.Rat)(r).Cmp(c.(Atom).Rat()) < 0
}

func (r *Rational) Add(c Cell) Number {
	return NewRational(new(big.Rat).Add((*big.Rat)(r), c.(Atom).Rat()))
}

func (r *Rational) Divide(c Cell) Number {
	return NewRational(new(big.Rat).Quo((*big.Rat)(r), c.(Atom).Rat()))
}

func (r *Rational) Modulo(c Cell) Number {
	return NewRational(ratmod((*big.Rat)(r), c.(Atom).Rat()))
}

func (r *Rational) Multiply(c Cell) Number {
	return NewRational(new(big.Rat).Mul((*big.Rat)(r), c.(Atom).Rat()))
}

func (r *Rational) Subtract(c Cell) Number {
	return NewRational(new(big.Rat).Sub((*big.Rat)(r), c.(Atom).Rat()))
}

/* String cell definition. */

type String string

func IsString(c Cell) bool {
	switch c.(type) {
	case *String:
		return true
	}
	return false
}

func NewString(v string) *String {
	s := String(v)
	return &s
}

func (s *String) Bool() bool {
	return true
}

func (s *String) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*s) == a.String()
	}
	return false
}

func (s *String) String() string {
	return adapted.Quote(string(*s))
}

func (s *String) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*s), 64); err != nil {
		panic(err)
	}
	return f
}

func (s *String) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *String) Rat() *big.Rat {
	r := new(big.Rat)
	if _, err := fmt.Sscan(string(*s), r); err != nil {
		panic(err)
	}
	return r
}

/* String-specific functions. */

func (s *String) Raw() string {
	return string(*s)
}

/* Symbol cell definition. */

type Symbol string

func IsSymbol(c Cell) bool {
	switch c.(type) {
	case *Symbol:
		return true
	}
	return false
}

func NewSymbol(v string) *Symbol {
	syml.RLock()
	p, ok := sym[v]
	syml.RUnlock()

	if ok {
		return p
	}

	s := Symbol(v)
	p = &s

	if len(v) <= 3 {
		syml.Lock()
		sym[v] = p
		syml.Unlock()
	}

	return p
}

func (s *Symbol) Bool() bool {
	if string(*s) == "false" {
		return false
	}

	return true
}

func (s *Symbol) Equal(c Cell) bool {
	if a, ok := c.(Atom); ok {
		return string(*s) == a.String()
	}
	return false
}

func (s *Symbol) String() string {
	return string(*s)
}

func (s *Symbol) Float() (f float64) {
	var err error
	if f, err = strconv.ParseFloat(string(*s), 64); err != nil {
		panic(err)
	}
	return f
}

func (s *Symbol) Int() (i int64) {
	var err error
	if i, err = strconv.ParseInt(string(*s), 0, 64); err != nil {
		panic(err)
	}
	return i
}

func (s *Symbol) Rat() *big.Rat {
	r := new(big.Rat)
	if _, err := fmt.Sscan(string(*s), r); err != nil {
		panic(err)
	}
	return r
}

func (s *Symbol) Greater(c Cell) bool {
	return string(*s) > c.(Atom).String()
}

func (s *Symbol) Less(c Cell) bool {
	return string(*s) < c.(Atom).String()
}

func (s *Symbol) Add(c Cell) Number {
	return NewRational(new(big.Rat).Add(s.Rat(), c.(Atom).Rat()))
}

func (s *Symbol) Divide(c Cell) Number {
	return NewRational(new(big.Rat).Quo(s.Rat(), c.(Atom).Rat()))
}

func (s *Symbol) Modulo(c Cell) Number {
	return NewRational(ratmod(s.Rat(), c.(Atom).Rat()))
}

func (s *Symbol) Multiply(c Cell) Number {
	return NewRational(new(big.Rat).Mul(s.Rat(), c.(Atom).Rat()))
}

func (s *Symbol) Subtract(c Cell) Number {
	return NewRational(new(big.Rat).Sub(s.Rat(), c.(Atom).Rat()))
}

/* Symbol-specific functions. */

func (s *Symbol) isNumeric() bool {
	r := new(big.Rat)
	_, err := fmt.Sscan(string(*s), r)
	return err == nil
}

/* Variable definition. */

type Variable struct {
	sync.RWMutex

	v Cell
}

func NewVariable(v Cell) *Variable {
	vr := new(Variable)

	vr.v = v

	return vr
}

func (vr *Variable) Copy() Reference {
	return NewVariable(vr.Get())
}

func (vr *Variable) Get() Cell {
	vr.RLock()
	defer vr.RUnlock()

	return vr.v
}

func (vr *Variable) Set(v Cell) {
	vr.Lock()
	defer vr.Unlock()

	vr.v = v
}
