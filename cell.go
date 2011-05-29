/* Released under an MIT-style license. See LICENSE. */

package cell

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

type Atom interface {
    Cell

    Float() float64
    Int() int64
    Status() int64

    Greater(c Cell) bool
    Less(c Cell) bool

    Add(c Cell) Atom
    Multiply(c Cell) Atom
}

type Cell interface {
    Bool() bool
    String() string

    Equal(c Cell) bool
}

type Interface interface {
    Cell

    Access(key Cell) *Reference
    Copy() Interface
        Expose() *Scope
    Faces() *Env
    Prev() Interface
        Private(key, value Cell)
    Public(key, value Cell)
}

type Number interface {
    Atom

    Divide(c Cell) Number
    Modulo(c Cell) Number
    Subtract(c Cell) Number
}

const (
    SaveCode = 1 << iota
    SaveDynamic
    SaveLexical
    SaveScratch
    SaveMax
)

var Null Cell
var False *Boolean
var True *Boolean

var num [512]*Integer
var res [256]*Status
var str map[string] *String
var sym map[string] *Symbol

func init() {
    pair := new(Pair)
    pair.car = pair
    pair.cdr = pair

    Null = Cell(pair)

    F := Boolean(false)
    False = &F

    T := Boolean(true)
    True = &T

    str = make(map[string] *String)
    sym = make(map[string] *Symbol)

    /* Make sure the following symbols are cached. */
    sym["is-boolean"] = NewSymbol("is-boolean")
    sym["is-integer"] = NewSymbol("is-integer")
    sym["is-method"] = NewSymbol("is-method")
    sym["is-number"] = NewSymbol("is-number")
    sym["is-object"] = NewSymbol("is-object")
    sym["is-status"] = NewSymbol("is-status")
    sym["is-string"] = NewSymbol("is-string")
    sym["is-symbol"] = NewSymbol("is-symbol")
    sym["append-stdout"] = NewSymbol("append-stdout")
    sym["append-stderr"] = NewSymbol("append-stderr")
    sym["background"] = NewSymbol("background")
    sym["pipe-stdout"] = NewSymbol("pipe-stdout")
    sym["pipe-stderr"] = NewSymbol("pipe-stderr")
    sym["redirect-stdin"] = NewSymbol("redirect-stdin")
    sym["redirect-stdout"] = NewSymbol("redirect-stdout")
    sym["redirect-stderr"] = NewSymbol("redirect-stderr")
}

func Append(list Cell, elements ...Cell) Cell {
    var pair, prev, start Cell

    index := 0

    start = Null

    if list != nil && list != Null {
        start = Cons(Car(list), Null)
        prev = start

        for list = Cdr(list); list != Null; list = Cdr(list) {
            pair = Cons(Car(list), Null)
            SetCdr(prev, pair)
            prev = pair
        }
    } else if len(elements) > 0 {
        start = Cons(elements[index], Null)
        prev = start
        index++
    }

    for ; index < len(elements); index++ {
        pair = Cons(elements[index], Null)
        SetCdr(prev, pair)
        prev = pair
    }

    return start
}

func AppendTo(list Cell, elements ...Cell) Cell {
    var pair, prev, start Cell

    index := 0

    start = Null

    if list == nil {
        panic("Cannot append to non-existent list.")
    }

    if list != Null {
        start = list

        for prev = list; Cdr(prev) != Null; prev = Cdr(prev) {
        }

    } else if len(elements) > 0 {
        start = Cons(elements[index], Null)
        prev = start
        index++
    }

    for ; index < len(elements); index++ {
        pair = Cons(elements[index], Null)
        SetCdr(prev, pair)
        prev = pair
    }

    return start
}

func Car(c Cell) Cell {
    return c.(*Pair).car
}

func Cdr(c Cell) Cell {
    return c.(*Pair).cdr
}

func Caar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).car
}

func Cadr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).car
}

func Cdar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).cdr
}

func Cddr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).cdr
}

func Caaar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).car.(*Pair).car
}

func Caadr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).car.(*Pair).car
}

func Cadar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).cdr.(*Pair).car
}

func Caddr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).cdr.(*Pair).car
}

func Cdaar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).car.(*Pair).cdr
}

func Cdadr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).car.(*Pair).cdr
}

func Cddar(c Cell) Cell {
    return c.(*Pair).car.(*Pair).cdr.(*Pair).cdr
}

func Cdddr(c Cell) Cell {
    return c.(*Pair).cdr.(*Pair).cdr.(*Pair).cdr
}

func IsAtom(c Cell) bool {
    switch c.(type) {
    case Atom:
        return true
    }
    
    return false
}

func IsCons(c Cell) bool {
    switch c.(type) {
    case *Pair:
        return true
    }

    return false
}

func IsList(c Cell) bool {
    if c == Null {
        return true
    }
    return IsList(Cdr(c))
}

func IsSimple(c Cell) bool {
    return IsAtom(c) || IsCons(c)
}

func Join(list Cell, elements ...Cell) Cell {
    var pair, prev, start Cell

    if list != nil && list != Null {
        start = Cons(Car(list), Null)

        for list = Cdr(list); list != Null; list = Cdr(list) {
            pair = Cons(Car(list), Null)
            SetCdr(prev, pair)
            prev = pair
        }
    } else if len(elements) > 0 {
        return Join(elements[0], elements[1:]...)
    } else {
        return Null
    }

    for index := 0; index < len(elements); index++ {
        for list = elements[index]; list != Null; list = Cdr(list) {
            pair = Cons(Car(list), Null)
            SetCdr(prev, pair)
            prev = pair
        }
    }

    return start
}

func JoinTo(list Cell, elements ...Cell) Cell {
    var pair, prev, start Cell

    start = list

    if list == nil {
        panic("Cannot append to non-existent list.")
    } else if list == Null {
        panic("Cannot destructively modify nil value.")
    }

    for ; list != Null; list = Cdr(list) {
        prev = list
    }

    for index := 0; index < len(elements); index++ {
        for list = elements[index]; list != Null; list = Cdr(list) {
            pair = Cons(Car(list), Null)
            SetCdr(prev, pair)
            prev = pair
        }
    }

    return start
}

func Length(list Cell) int64 {
    var length int64 = 0

    for ; list != nil && list != Null; list = Cdr(list) {
        length++
    }

    return length
}

func List(elements ...Cell) Cell {
    if len(elements) <= 0 {
        return Null
    }

    start := Cons(elements[0], Null)
    prev := start

    for index := 1; index < len(elements); index++ {
        pair := Cons(elements[index], Null)
        SetCdr(prev, pair)
        prev = pair
    }

    return start
}

func Raw(c Cell) string {
    if s, ok := c.(*String); ok {
        return s.Raw()
    }

    return c.String()
}

func Resolve(s Interface, e *Env, k *Symbol) (v *Reference) {
    v = nil

    if v = s.Access(k); v == nil {
        if e == nil {
            return v
        }

        v = e.Access(k)
    } else if m, ok := v.GetValue().(*Method); ok &&
        m.Self != nil && m.Self != s.Expose(){
        v = NewReference(NewMethod(m.Func, s.Expose()))
    }

    return v
}

func Reverse(list Cell) Cell {
    reversed := Null

    for ; list != nil && list != Null; list = Cdr(list) {
        reversed = Cons(Car(list), reversed)
    }

    return reversed
}

func SetCar(c, value Cell) {
    c.(*Pair).car = value
}

func SetCdr(c, value Cell) {
    c.(*Pair).cdr = value
}

func Tail(list Cell, index int64) Cell {
    for ; index > 0 && IsCons(list); index++ {
        list = Cdr(list)
    }

    return list
}


/* Boolean cell definition. */

type Boolean bool

func NewBoolean(v bool) *Boolean {
    if v {
        return True
    }
    return False
}

func (self *Boolean) Bool() bool {
    return bool(self == True)
}

func (self *Boolean) Float() float64 {
    if bool(self == True) {
        return 1.0
    }
    return 0.0
}

func (self *Boolean) Int() int64 {
    if bool(self == True) {
        return 1
    }
    return 0
}

func (self *Boolean) Status() int64 {
    if bool(self == True) {
        return 0
    }
    return 1
}

func (self *Boolean) String() string {
    if bool(self == True) {
        return "true"
    }
    return "false"
}

func (self *Boolean) Equal(c Cell) bool {
    return bool(*self) == c.(Atom).Bool()
}

func (self *Boolean) Greater(c Cell) bool {
    return bool(*self) && !c.(Atom).Bool()
}

func (self *Boolean) Less(c Cell) bool {
    return !bool(*self) && c.(Atom).Bool()
}

func (self *Boolean) Add(c Cell) Atom {
    if bool(*self) || c.(Atom).Bool() {
        return True
    }
    return False
}

func (self *Boolean) Multiply(c Cell) Atom {
    if bool(*self) && c.(Atom).Bool() {
        return True
    }
    return False
}


/* Integer cell definition. */

type Integer int64

func NewInteger(v int64) *Integer {
    if -256 <= v && v <= 255 {
        n := v + 256
        p := num[n]

        if p == nil {
            i := Integer(v)
            p = &i

            num[n] = p
        }

        return p
    }

    i := Integer(v)
    return &i
}

func (self *Integer) Bool() bool {
    return *self != 0
}

func (self *Integer) Float() float64 {
    return float64(*self)
}

func (self *Integer) Int() int64 {
    return int64(*self)
}

func (self *Integer) Status() int64 {
    return int64(*self)
}

func (self *Integer) String() string {
    return strconv.Itoa64(int64(*self))
}

func (self *Integer) Equal(c Cell) bool {
    return int64(*self) == c.(Atom).Int()
}

func (self *Integer) Greater(c Cell) bool {
    return int64(*self) > c.(Atom).Int()
}

func (self *Integer) Less(c Cell) bool {
    return int64(*self) < c.(Atom).Int()
}

func (self *Integer) Add(c Cell) Atom {
    return NewInteger(int64(*self) + c.(Atom).Int())
}

func (self *Integer) Divide(c Cell) Number {
    return NewInteger(int64(*self) / c.(Atom).Int())
}

func (self *Integer) Modulo(c Cell) Number {
    return NewInteger(int64(*self) % c.(Atom).Int())
}

func (self *Integer) Multiply(c Cell) Atom {
    return NewInteger(int64(*self) * c.(Atom).Int())
}

func (self *Integer) Subtract(c Cell) Number {
    return NewInteger(int64(*self) - c.(Atom).Int())
}


/* Status cell definition. */

type Status int64

func NewStatus(v int64) *Status {
    if 0 <= v && v <= 255 {
        p := res[v]

        if p == nil {
            s := Status(v)
            p = &s

            res[v] = p
        }

        return p
    }

    s := Status(v)
    return &s
}

func (self *Status) Bool() bool {
    return int64(*self) == 0
}

func (self *Status) Float() float64 {
    return float64(*self)
}

func (self *Status) Int() int64 {
    return int64(*self)
}

func (self *Status) Status() int64 {
    return int64(*self)
}

func (self *Status) String() string {
    return strconv.Itoa64(int64(*self))
}

func (self *Status) Equal(c Cell) bool {
    return int64(*self) == c.(Atom).Status()
}

func (self *Status) Greater(c Cell) bool {
    return int64(*self) > c.(Atom).Status()
}

func (self *Status) Less(c Cell) bool {
    return int64(*self) < c.(Atom).Status()
}

func (self *Status) Add(c Cell) Atom {
    return NewStatus(int64(*self) + c.(Atom).Status())
}

func (self *Status) Divide(c Cell) Number {
    return NewStatus(int64(*self) / c.(Atom).Status())
}

func (self *Status) Modulo(c Cell) Number {
    return NewStatus(int64(*self) % c.(Atom).Status())
}

func (self *Status) Multiply(c Cell) Atom {
    return NewStatus(int64(*self) * c.(Atom).Status())
}

func (self *Status) Subtract(c Cell) Number {
    return NewStatus(int64(*self) - c.(Atom).Status())
}


/* Float cell definition. */

type Float float64

func NewFloat(v float64) *Float {
    f := Float(v)
    return &f
}

func (self *Float) Bool() bool {
    return *self != 0
}

func (self *Float) Float() float64 {
    return float64(*self)
}

func (self *Float) Int() int64 {
    return int64(*self)
}

func (self *Float) Status() int64 {
    return int64(*self)
}

func (self *Float) String() string {
    return strconv.Ftoa64(float64(*self), 'g', -1)
}

func (self *Float) Equal(c Cell) bool {
    return float64(*self) == c.(Atom).Float()
}

func (self *Float) Greater(c Cell) bool {
    return float64(*self) > c.(Atom).Float()
}

func (self *Float) Less(c Cell) bool {
    return float64(*self) < c.(Atom).Float()
}

func (self *Float) Add(c Cell) Atom {
    return NewFloat(float64(*self) + c.(Atom).Float())
}

func (self *Float) Divide(c Cell) Number {
    return NewFloat(float64(*self) / c.(Atom).Float())
}

func (self *Float) Modulo(c Cell) Number {
    panic("Type 'float' does not implement 'modulo'.")
}

func (self *Float) Multiply(c Cell) Atom {
    return NewFloat(float64(*self) * c.(Atom).Float())
}

func (self *Float) Subtract(c Cell) Number {
    return NewFloat(float64(*self) - c.(Atom).Float())
}


/* Symbol cell definition. */

type Symbol string

func NewSymbol(v string) *Symbol {
    p, ok := sym[v]

    if ok {
        return p
    }

    s := Symbol(v)
    p = &s

    if len(v) <= 8 {
        sym[v] = p
    }

    return p
}

func (self *Symbol) Bool() bool {
    if string(*self) == "false" {
        return false
    }

    return true
}

func (self *Symbol) Float() (f float64) {
    var err os.Error
    if f, err = strconv.Atof64(string(*self)); err != nil {
        panic(err)
    }
    return f
}

func (self *Symbol) Int() (i int64) {
    var err os.Error
    if i, err = strconv.Btoi64(string(*self), 0); err != nil {
        panic(err)
    }
    return i
}

func (self *Symbol) Status() (i int64) {
    var err os.Error
    if i, err = strconv.Btoi64(string(*self), 0); err != nil {
        panic(err)
    }
    return i
}

func (self *Symbol) String() string {
    return string(*self)
}

func (self *Symbol) Equal(c Cell) bool {
    return string(*self) == c.(Atom).String()
}

func (self *Symbol) Greater(c Cell) bool {
    return string(*self) > c.(Atom).String()
}

func (self *Symbol) Less(c Cell) bool {
    return string(*self) < c.(Atom).String()
}

func (self *Symbol) isFloat() bool {
    _, err := strconv.Atof64(string(*self))
    return err == nil
}

func (self *Symbol) isInt() bool {
    _, err := strconv.Btoi64(string(*self), 0)
    return err == nil
}

func (self *Symbol) Add(c Cell) Atom {
    if self.isInt() {
        return NewInteger(self.Int() + c.(Atom).Int())
    } else if self.isFloat() {
        return NewFloat(self.Float() + c.(Atom).Float())
    }

    return NewSymbol(string(*self) + Raw(c))
}

func (self *Symbol) Divide(c Cell) Number {
    if self.isInt() {
        return NewInteger(self.Int() / c.(Atom).Int())
    } else if self.isFloat() {
        return NewFloat(self.Float() / c.(Atom).Float())
    }

    panic("Type 'symbol' does not implement 'divide'.")
}

func (self *Symbol) Modulo(c Cell) Number {
    if self.isInt() {
        return NewInteger(self.Int() % c.(Atom).Int())
    }

    panic("Type 'symbol' does not implement 'modulo'.")
}

func (self *Symbol) Multiply(c Cell) Atom {
    if self.isInt() {
        return NewInteger(self.Int() * c.(Atom).Int())
    } else if self.isFloat() {
        return NewFloat(self.Float() * c.(Atom).Float())
    }

    var i int64
    var r string

    for r, i = string(*self), c.(Atom).Int(); i > 0; i-- {
        r += string(*self)
    }

    return NewSymbol(r)
}

func (self *Symbol) Subtract(c Cell) Number {
    if self.isInt() {
        return NewInteger(self.Int() - c.(Atom).Int())
    } else if self.isFloat() {
        return NewFloat(self.Float() - c.(Atom).Float())
    }

    panic("Type 'symbol' does not implement 'subtract'.")
}


/* String cell definition. */

type String string

func NewString(q string) *String {
    v, _ := strconv.Unquote("\"" + q + "\"")

    p, ok := str[v]

    if ok {
        return p
    }

    s := String(v)
    p = &s

    if len(v) <= 8 {
        str[v] = p
    }

    return p
}

func (self *String) Bool() bool {
    return true
}

func (self *String) Float() (f float64) {
    var err os.Error
    if f, err = strconv.Atof64(string(*self)); err != nil {
        panic(err)
    }
    return f
}

func (self *String) Int() (i int64) {
    var err os.Error
    if i, err = strconv.Btoi64(string(*self), 0); err != nil {
        panic(err)
    }
    return i
}

func (self *String) Raw() string {
    return string(*self)
}

func (self *String) Status() (i int64) {
    var err os.Error
    if i, err = strconv.Btoi64(string(*self), 0); err != nil {
        panic(err)
    }
    return i
}

func (self *String) String() string {
    return strconv.Quote(string(*self))
}

func (self *String) Equal(c Cell) bool {
    return string(*self) == c.(Atom).String()
}

func (self *String) Greater(c Cell) bool {
    return string(*self) > c.(Atom).String()
}

func (self *String) Less(c Cell) bool {
    return string(*self) < c.(Atom).String()
}

func (self *String) Add(c Cell) Atom {
    return NewString(string(*self) + Raw(c))
}

func (self *String) Multiply(c Cell) Atom {
    var i int64
    var r string

    for r, i = string(*self), c.(Atom).Int(); i > 0; i-- {
        r += string(*self)
    }

    return NewSymbol(r)
}


/* Pair cell definition. */

type Pair struct {
    car Cell
    cdr Cell
}

func Cons(h, t Cell) Cell {
    return &Pair{h, t}
}

func (self *Pair) Bool() bool {
    return self != Null
}

func (self *Pair) String() (s string) {
    s = ""

    if IsCons(self.car) && IsCons(Cdr(self.car)) {
        s += "("
    }

    if self.car != Null {
        s += self.car.String()
    }

    if IsCons(self.car) && IsCons(Cdr(self.car)) {
        s += ")"
    }

    if IsCons(self.cdr) {
        if self.cdr == Null {
            return s
        }

        s += " "
    } else {
        s += "::"
    }

    s += self.cdr.String()

    return s
}

func (self *Pair) Equal(c Cell) bool {
    if self == Null && c == Null {
        return true
    }
    return self.car.Equal(Car(c)) && self.cdr.Equal(Cdr(c))
}


/* Channel cell definition. */

type Channel struct {
    b *bufio.Reader
    c chan Cell
    d chan bool
    r *os.File
    w *os.File
    Implicit bool
}

func NewChannel(r *os.File, w *os.File) *Channel {
    ch := &Channel{nil, nil, nil, r, w, false}

    if r == nil && w == nil {
        var err os.Error

        if ch.r, ch.w, err = os.Pipe(); err != nil {
            ch.r, ch.w = nil, nil
        }
    }

    return ch
}

func (self *Channel) Bool() bool {
    return true
}

func (self *Channel) String() string {
    return fmt.Sprintf("%%channel %p%%", self)
}

func (self *Channel) Equal(c Cell) bool {
    return c.(*Channel) == self
}

/* Channel-specific functions */

func (self *Channel) Close() {
    if self.r != nil && len(self.r.Name()) > 0 {

        self.r.Close()
        self.r = nil
    }

    if self.w != nil && len(self.w.Name()) > 0 {

        self.w.Close()
        self.w = nil
    }

    return
}

func (self *Channel) reader() *bufio.Reader {
    if self.b == nil {
        self.b = bufio.NewReader(self.r)
    }

    return self.b
}

func (self *Channel) Read() Cell {
    if self.r == nil {
        return Null
    }

    if self.c == nil {
        self.c = make(chan Cell)
        self.d = make(chan bool)
        go Parse(self.reader(), func (c Cell) { self.c <- c; <-self.d })
    } else {
        self.d <- true
    }

    return <-self.c
}

func (self *Channel) ReadLine() Cell {
    if self.r == nil {
        return Null
    }

    s, err := self.reader().ReadString('\n')
    if err != nil && len(s) == 0 {
        self.b = nil
        return Null
    }

    return NewString(strings.TrimRight(s, "\n"))
}

func (self *Channel) ReadEnd() *os.File {
    return self.r
}

func (self *Channel) Write(c Cell) {
    if self.w == nil || c == Null {
        return
    }
    
    fmt.Fprintln(self.w, c)
}

func (self *Channel) WriteEnd() *os.File {
    return self.w
}


/* Closure cell definition. */

type Closure struct {
    Body Cell
    Param Cell
    Lexical *Scope
}

func NewClosure(Body, Param Cell, Lexical *Scope) *Closure {
    return &Closure{Body, Param, Lexical}
}

func (self *Closure) Bool() bool {
    return true
}

func (self *Closure) String() string {
    return fmt.Sprintf("%%closure %p%%", self)
}

func (self *Closure) Equal(c Cell) bool {
    return c.(*Closure) == self
}


/* Env cell definition. */

type Env struct {
    hash map[string] *Reference
    prev *Env
}

func NewEnv(prev *Env) *Env {
    return &Env{make(map[string] *Reference), prev}
}

func (self *Env) Bool() bool {
    return true
}

func (self *Env) String() string {
    return fmt.Sprintf("%%env %p%%", self)
}

func (self *Env) Equal(c Cell) bool {
    return c.(*Env) == self
}


/* Env-specific functions */

func (self *Env) Access(key Cell) *Reference {
    for env := self; env != nil; env = env.prev {
        if value, ok := env.hash[key.String()]; ok {
            return value
        }
    }

    return nil
}

func (self *Env) Add(key Cell, value Cell) {
    self.hash[key.String()] = NewReference(value)
}

func (self *Env) Copy() *Env {
    if self == nil {
        return nil
    }

    fresh := NewEnv(self.prev.Copy())

    for k, v := range self.hash {
        fresh.hash[k] = NewReference(v.GetValue())
    }

    return fresh
}

func (self *Env) Prev() *Env {
    return self.prev
}


/* Function cell definition. */

type Function func(p *Process, args Cell) bool

func (self Function) Bool() bool {
    return true
}

func (self Function) String() string {
    return fmt.Sprintf("%%function %p%%", self)
}

func (self Function) Equal(c Cell) bool {
    return c.(Function) == self
}


/* Method cell definition. */

type Method struct {
    Func *Closure
    Self *Scope
}

func NewMethod(Func *Closure, Self *Scope) *Method {
    return &Method{Func, Self}
}

func (self *Method) Bool() bool {
    return true
}

func (self *Method) String() string {
    return fmt.Sprintf("%%method %p%%", self)
}

func (self *Method) Equal(c Cell) bool {
    m := c.(*Method)
    return m.Func == self.Func && m.Self == self.Self
}


/* Object cell definition. (An object cell is an object's public face). */

type Object struct {
    *Scope
}

func NewObject(v Interface) *Object {
    return &Object{v.Expose()}
}

func (self *Object) String() string {
    return fmt.Sprintf("%%object %p%%", self)
}

func (self *Object) Equal(c Cell) bool {
    return c.(*Object) == self || c.(*Scope) == self.Scope
}

/* Object-specific functions */

func (self *Object) Access(key Cell) *Reference {
    var obj Interface
        for obj = self.Scope; obj != nil; obj = obj.Prev() {
        if value := obj.Faces().prev.Access(key); value != nil {
            return value
        }
    }

    return nil
}

func (self *Object) Copy() Interface {
    return &Object{&Scope{self.Scope.env.Copy(), self.Scope.prev}}
}

func (self *Object) Expose() *Scope {
    return self.Scope
}

func (self *Object) Faces() *Env {
    return self.env.prev
}

func (self *Object) Prev() Interface {
    return self.prev
}

func (self *Object) Private(key Cell, value Cell) {
    panic("Private members cannot be added to an object.")
}


/* Process cell definition. */

type Process struct {
    Code Cell
    Dynamic *Env
    Lexical Interface
        Scratch, Stack Cell
}

func NewProcess(state int64, env *Env, scope Interface) *Process {
    return &Process{
        Null,
        NewEnv(env),
        NewScope(scope),
        Null,
        List(NewInteger(state)),
    }
}

func (self *Process) Bool() bool {
    return true
}

func (self *Process) String() string {
    return fmt.Sprintf("%%process %p%%", self)
}

func (self *Process) Equal(c Cell) bool {
    return c.(*Process) == self
}

/* Process-specific functions. */

func (self *Process) Arguments() Cell {
    e := Car(self.Scratch)
    l := Null

    for e != nil {
        l = Cons(e, l)

        self.Scratch = Cdr(self.Scratch)
        e = Car(self.Scratch)
    }

    self.Scratch = Cdr(self.Scratch)

    return l
}

func (self *Process) Continuation(state int64) *Method {
    return NewMethod(NewClosure(
        NewInteger(state),
        List(Cdr(self.Scratch), self.Stack),
        nil),
        nil)
}

func (self *Process) GetState() int64 {
    if self.Stack == Null {
        return 0
    }
    return Car(self.Stack).(Atom).Int()
}

func (self *Process) NewState(state int64) {
    self.Stack = Cons(NewInteger(state), self.Stack)
}

func (self *Process) RemoveState() {
    self.Stack = Cdr(self.Stack)
}

func (self *Process) ReplaceState(state int64) {
    SetCar(self.Stack, NewInteger(state))
}

func (self *Process) RestoreState() {
    f := self.GetState()

    if f >= SaveMax {
        return
    }

    if f & SaveScratch > 0 {
        self.Stack = Cdr(self.Stack)
        self.Scratch = Car(self.Stack)
    }

    if f & SaveLexical > 0 {
        self.Stack = Cdr(self.Stack)
        self.Lexical = Car(self.Stack).(Interface)
    }

    if f & SaveDynamic > 0 {
        self.Stack = Cdr(self.Stack)
        self.Dynamic = Car(self.Stack).(*Env)
    }

    if f & SaveCode > 0 {
        self.Stack = Cdr(self.Stack)
        self.Code = Car(self.Stack)
    }
}

func (self *Process) SaveState(f int64, c... Cell) bool {
    if s := self.GetState(); s < SaveMax && f & s == f {
        return false
    }

    if f & SaveCode > 0 {
        self.Stack = Cons(c[0], self.Stack)
    }

    if f & SaveDynamic > 0 {
        self.Stack = Cons(self.Dynamic, self.Stack)
    }

    if f & SaveLexical > 0 {
        self.Stack = Cons(self.Lexical, self.Stack)
    }

    if f & SaveScratch > 0 {
        self.Stack = Cons(self.Scratch, self.Stack)
    }

    self.NewState(f)
    return true
}


/* Scope cell definition. A scope cell is an object public + private faces. */

type Scope struct {
    env *Env
    prev Interface
    }

func NewScope(prev Interface) *Scope {
    return &Scope{NewEnv(NewEnv(nil)), prev}
}

func (self *Scope) Bool() bool {
    return true
}

func (self *Scope) String() string {
    return fmt.Sprintf("%%scope %p%%", self)
}

func (self *Scope) Equal(c Cell) bool {
    return c.(*Scope) == self
}

/* Scope-specific functions */

func (self *Scope) Access(key Cell) *Reference {
    var obj Interface
        for obj = self; obj != nil; obj = obj.Prev() {
        if value := obj.Faces().Access(key); value != nil {
            return value
        }
    }

    return nil
}

func (self *Scope) Copy() Interface {
    return &Scope{self.env.Copy(), self.prev}
}

func (self *Scope) Expose() *Scope {
    return self
}

func (self *Scope) Faces() *Env {
    return self.env
}

func (self *Scope) Prev() Interface {
    return self.prev
}

func (self *Scope) Private(key Cell, value Cell) {
    self.env.Add(key, value)
}

func (self *Scope) Public(key Cell, value Cell) {
    self.env.prev.Add(key, value)
}

func (self *Scope) PrivateFunction(k string, f Function) {
    self.Private(NewSymbol(k), NewMethod(NewClosure(f, Null, self), nil))
}

func (self *Scope) PrivateMethod(k string, f Function) {
    self.Private(NewSymbol(k), NewMethod(NewClosure(f, Null, self), self))
}

func (self *Scope) PublicMethod(k string, f Function) {
    self.Public(NewSymbol(k), NewMethod(NewClosure(f, Null, self), self))
}

func (self *Scope) PrivateState(k string, v int64) {
    self.Private(NewSymbol(k),
        NewMethod(NewClosure(NewInteger(v), Null, self), self))
}

func (self *Scope) PublicState(k string, v int64) {
    self.Public(NewSymbol(k),
        NewMethod(NewClosure(NewInteger(v), Null, self), self))
}


/* Reference cell definition. */

type Reference struct {
    v Cell
}

func NewReference(v Cell) *Reference {
    return &Reference{v}
}

func (self *Reference) Bool() bool {
    return self.v.Bool()
}

func (self *Reference) String() string {
    return self.v.String()
}

func (self *Reference) Equal(c Cell) bool {
    return self.v.Equal(c)
}

func (self *Reference) GetValue() Cell {
    return self.v
}

func (self *Reference) SetValue(c Cell) {
    self.v = c
}
