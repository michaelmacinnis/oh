/* Released under an MIT-style license. See LICENSE. */

package main

import (
	"fmt"
	"strings"
)

type Binding interface {
	Cell

	Bind(c Context) Binding
	Ref() Closure
	Self() Context
}

type Closure interface {
	Cell

	Applier() Function
	Body() Cell
	Label() Cell
	Params() Cell
	Scope() Context
}

type Context interface {
	Cell

	Access(key Cell) Reference
	Copy() Context
	Complete(line, prefix string) []string
	Define(key, value Cell)
	Expose() Context
	Faces() *Env
	Prev() Context
	Public(key, value Cell)
	Remove(key Cell) bool

	DefineBuiltin(k string, f Function)
	DefineMethod(k string, f Function)
	DefineSyntax(k string, f Function)
	PublicMethod(k string, f Function)
	PublicSyntax(k string, f Function)
}

type Function func(t *Task, args Cell) bool

type NewClosure func(a Function, b, l, p Cell, s Context) Closure

func Resolve(s Context, e *Env, k *Symbol) (v Reference) {
	v = nil

	if v = s.Access(k); v == nil {
		if e == nil {
			return v
		}

		v = e.Access(k)
	}

	return v
}

/* Bound cell definition. */

type Bound struct {
	ref     Closure
	context Context
}

func NewBound(ref Closure, context Context) *Bound {
	return &Bound{ref, context}
}

func (b *Bound) Bool() bool {
	return true
}

func (b *Bound) Equal(c Cell) bool {
	if m, ok := c.(*Bound); ok {
		return b.ref == m.Ref() && b.context == m.Self()
	}
	return false
}

func (b *Bound) String() string {
	return fmt.Sprintf("%%bound %p%%", b)
}

/* Bound-specific functions */

func (b *Bound) Bind(c Context) Binding {
	if c == b.context {
		return b
	}
	return NewBound(b.ref, c)
}

func (b *Bound) Ref() Closure {
	return b.ref
}

func (b *Bound) Self() Context {
	return b.context
}

/* Builtin cell definition. */

type Builtin struct {
	Combiner
}

func NewBuiltin(a Function, b, l, p Cell, s Context) Closure {
	return &Builtin{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (b *Builtin) Equal(c Cell) bool {
	return b == c
}

func (b *Builtin) String() string {
	return fmt.Sprintf("%%builtin %p%%", b)
}

/* Combiner cell definition. */

type Combiner struct {
	applier Function
	body    Cell
	label   Cell
	params  Cell
	scope   Context
}

func (c *Combiner) Bool() bool {
	return true
}

func (c *Combiner) Applier() Function {
	return c.applier
}

func (c *Combiner) Body() Cell {
	return c.body
}

func (c *Combiner) Params() Cell {
	return c.params
}

func (c *Combiner) Label() Cell {
	return c.label
}

func (c *Combiner) Scope() Context {
	return c.scope
}

/* Env cell definition. */

type Env struct {
	hash map[string]Reference
	prev *Env
}

func NewEnv(prev *Env) *Env {
	return &Env{make(map[string]Reference), prev}
}

func (e *Env) Bool() bool {
	return true
}

func (e *Env) Equal(c Cell) bool {
	return e == c
}

func (e *Env) String() string {
	return fmt.Sprintf("%%env %p%%", e)
}

/* Env-specific functions */

func (e *Env) Access(key Cell) Reference {
	for env := e; env != nil; env = env.prev {
		if value, ok := env.hash[key.String()]; ok {
			return value
		}
	}

	return nil
}

func (e *Env) Add(key Cell, value Cell) {
	e.hash[key.String()] = NewVariable(value)
}

func (e *Env) Complete(line, prefix string) []string {
	cl := []string{}

	for k, _ := range e.hash {
		if strings.HasPrefix(k, prefix) {
			cl = append(cl, line+k)
		}
	}

	if e.prev != nil {
		cl = append(cl, e.prev.Complete(line, prefix)...)
	}

	return cl
}

func (e *Env) Copy() *Env {
	if e == nil {
		return nil
	}

	fresh := NewEnv(e.prev.Copy())

	for k, v := range e.hash {
		fresh.hash[k] = v.Copy()
	}

	return fresh
}

func (e *Env) Method(name string, m Function) {
	e.hash[name] =
		NewConstant(NewBound(NewMethod(m, Null, Null, Null, nil), nil))
}

func (e *Env) Prev() *Env {
	return e.prev
}

func (e *Env) Remove(key Cell) bool {
	_, ok := e.hash[key.String()]

	delete(e.hash, key.String())

	return ok
}

/* Method cell definition. */

type Method struct {
	Combiner
}

func NewMethod(a Function, b, l, p Cell, s Context) Closure {
	return &Method{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (m *Method) Equal(c Cell) bool {
	return m == c
}

func (m *Method) String() string {
	return fmt.Sprintf("%%method %p%%", m)
}

/*
 * Object cell definition.
 * (An object cell only allows access to a context's public members).
 */

type Object struct {
	Context
}

func NewObject(v Context) *Object {
	return &Object{v.Expose()}
}

func (o *Object) Equal(c Cell) bool {
	if o == c {
		return true
	}
	if o, ok := c.(*Object); ok {
		return o.Context == o.Expose()
	}
	return false
}

func (o *Object) String() string {
	return fmt.Sprintf("%%object %p%%", o)
}

/* Object-specific functions */

func (o *Object) Access(key Cell) Reference {
	var obj Context
	for obj = o; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().prev.Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (o *Object) Copy() Context {
	return &Object{
		&Scope{o.Expose().Faces().Copy(), o.Context.Prev()},
	}
}

func (o *Object) Expose() Context {
	return o.Context
}

func (o *Object) Define(key Cell, value Cell) {
	panic("Private members cannot be added to an object.")
}

/*
 * Scope cell definition.
 * (A scope cell allows access to a context's public and private members).
 */

type Scope struct {
	env  *Env
	prev Context
}

func NewScope(prev Context, fixed *Env) *Scope {
	return &Scope{NewEnv(NewEnv(fixed)), prev}
}

func (s *Scope) Bool() bool {
	return true
}

func (s *Scope) Equal(c Cell) bool {
	return s == c
}

func (s *Scope) String() string {
	return fmt.Sprintf("%%scope %p%%", s)
}

/* Scope-specific functions */

func (s *Scope) Access(key Cell) Reference {
	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		if value := obj.Faces().Access(key); value != nil {
			return value
		}
	}

	return nil
}

func (s *Scope) Complete(line, prefix string) []string {
	cl := []string{}

	var obj Context
	for obj = s; obj != nil; obj = obj.Prev() {
		cl = append(cl, obj.Faces().Complete(line, prefix)...)
	}

	return cl
}

func (s *Scope) Copy() Context {
	return &Scope{s.env.Copy(), s.prev}
}

func (s *Scope) Expose() Context {
	return s
}

func (s *Scope) Faces() *Env {
	return s.env
}

func (s *Scope) Prev() Context {
	return s.prev
}

func (s *Scope) Define(key Cell, value Cell) {
	s.env.Add(key, value)
}

func (s *Scope) Public(key Cell, value Cell) {
	s.env.prev.Add(key, value)
}

func (s *Scope) Remove(key Cell) bool {
	if !s.env.prev.Remove(key) {
		return s.env.Remove(key)
	}

	return true
}

func (s *Scope) DefineBuiltin(k string, a Function) {
	s.Define(NewSymbol(k),
		NewUnbound(NewBuiltin(a, Null, Null, Null, s)))
}

func (s *Scope) DefineMethod(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, s), s))
}

func (s *Scope) PublicMethod(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewMethod(a, Null, Null, Null, s), s))
}

func (s *Scope) DefineSyntax(k string, a Function) {
	s.Define(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, s), s))
}

func (s *Scope) PublicSyntax(k string, a Function) {
	s.Public(NewSymbol(k),
		NewBound(NewSyntax(a, Null, Null, Null, s), s))
}

/* Syntax cell definition. */

type Syntax struct {
	Combiner
}

func NewSyntax(a Function, b, l, p Cell, s Context) Closure {
	return &Syntax{
		Combiner{applier: a, body: b, label: l, params: p, scope: s},
	}
}

func (m *Syntax) Equal(c Cell) bool {
	return m == c
}

func (m *Syntax) String() string {
	return fmt.Sprintf("%%syntax %p%%", m)
}

/* Unbound cell definition. */

type Unbound struct {
	ref Closure
}

func NewUnbound(Ref Closure) *Unbound {
	return &Unbound{Ref}
}

func (u *Unbound) Bool() bool {
	return true
}

func (u *Unbound) Equal(c Cell) bool {
	if u, ok := c.(*Unbound); ok {
		return u.ref == u.Ref()
	}
	return false
}

func (u *Unbound) String() string {
	return fmt.Sprintf("%%unbound %p%%", u)
}

/* Unbound-specific functions */

func (u *Unbound) Bind(c Context) Binding {
	return u
}

func (u *Unbound) Ref() Closure {
	return u.ref
}

func (u *Unbound) Self() Context {
	return nil
}

