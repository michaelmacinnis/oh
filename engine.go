/* released under an MIT-style license. See LICENSE. */

package main

import (
    "bufio"
    "exec"
    "fmt"
    "os"
    "path"
    "path/filepath"
    "strings"
    "strconv"
)

const (
    psNone = 0

    psChangeScope = SaveMax + iota
    psCreateModule

    psDoEvalArguments
    psDoEvalCommand

    psEvalAccess
    psEvalAnd
    psEvalAndf
    psEvalArguments
    psEvalBlock
    psEvalCommand
    psEvalElement
    psEvalFor
    psEvalOr
    psEvalOrf
    psEvalReference
    psEvalWhileBody
    psEvalWhileTest

    psExecAccess
    psExecApplication
    psExecBuiltin
    psExecDefine
    psExecDynamic
    psExecExternal
    psExecFor
    psExecIf
    psExecImport
    psExecSource
    psExecObject
    psExecPublic
    psExecReference
    psExecSet
    psExecSetenv
    psExecSplice

    /* Commands. */
    psBlock
    psBuiltin
    psDefine
    psDynamic
    psFor
    psIf
    psImport
    psMethod
    psObject
    psPublic
    psQuote
    psReturn
    psSet
    psSetenv
    psSource
    psSpawn
    psSplice
    psWhile

    /* Operators. */
    psAnd
    psAndf
    psBackground
    psBacktick
    psOr
    psOrf

    psAppendStdout
    psAppendStderr
    psPipeChild
    psPipeParent
    psPipeStderr
    psPipeStdout
    psRedirectCleanup
    psRedirectSetup
    psRedirectStderr
    psRedirectStdin
    psRedirectStdout

    psMax
)

var proc0 *Process

func channel(p *Process, r, w *os.File) Interface {
    c, ch := NewScope(p.Lexical), NewChannel(r, w)

    var read Function = func (p *Process, args Cell) bool {
        SetCar(p.Scratch, ch.Read())                                   
        return false
    }

    var readline Function = func (p *Process, args Cell) bool {
        SetCar(p.Scratch, ch.ReadLine())
        return false
    }

    var write Function = func (p *Process, args Cell) bool {
        ch.Write(args)
        SetCar(p.Scratch, True)
        return false
    }

    c.Public(NewSymbol("guts"), ch)
    c.Public(NewSymbol("read"), method(read, Null, c))
    c.Public(NewSymbol("readline"), method(readline, Null, c))
    c.Public(NewSymbol("write"), method(write, Null, c))

    return NewObject(c)
}

func debug(p *Process, s string) {
    fmt.Printf("%s: p.Code = %v, p.Scratch = %v\n", s, p.Code, p.Scratch)
}

func expand(args Cell) Cell {
    list := Null

    for args != Null {
        c := Car(args)

        s := Raw(c)
        if _, ok := c.(*Symbol); ok {
            if s[:1] == "~" {
                s = filepath.Join(os.Getenv("HOME"), s[1:])
            }

            if strings.IndexAny(s, "*?[") != -1 {
                m, err := filepath.Glob(s)
                if err != nil || len(m) == 0 {
                    panic("no matches found: " + s)
                }

                for _, e := range m {
                    if e[0] != '.' || s[0] == '.' {
                        list = AppendTo(list, NewSymbol(e))
                    }
                }
            } else {
                list = AppendTo(list, NewSymbol(s))
            }
        } else {
            list = AppendTo(list, NewSymbol(s))
        }
        args = Cdr(args)
    }   

    return list
}

func external(p *Process, args Cell) bool {
    name, err := exec.LookPath(Raw(Car(p.Scratch)))

    SetCar(p.Scratch, False)

    if err != nil {
        panic(err)
    }

    argv := []string{name}

    for args = expand(args); args != Null; args = Cdr(args) {
        argv = append(argv, Car(args).String())
    }

    c := Resolve(p.Lexical, p.Dynamic, NewSymbol("$cwd"))
    dir := c.GetValue().String()

    var fd[]*os.File //{os.Stdin, os.Stdout, os.Stderr}

    fd = append(fd,
		rpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).GetValue()),
		wpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdout")).GetValue()),
		wpipe(Resolve(p.Lexical, p.Dynamic, NewSymbol("$stderr")).GetValue()))

    proc, err := os.StartProcess(name, argv, &os.ProcAttr{dir, nil, fd, nil})
    if err != nil {
        panic(err)
    }

    var status int64 = 0

    msg, err := proc.Wait(0)
    if err != nil {
        status = int64(err.(*os.SyscallError).Errno)
    } else {
        status = int64(msg.ExitStatus())
    }

    SetCar(p.Scratch, NewStatus(status))

    return false
}

func function(body, param Cell, scope *Scope) *Method {
    return NewMethod(NewClosure(body, param, scope), nil)
}

func method(body, param Cell, scope *Scope) *Method {
    return NewMethod(NewClosure(body, param, scope), scope)
}

func module(f string) (string, os.Error) {
    i, err := os.Stat(f)
    if err != nil {
        return "", err
    }
    
    m := "$" + f + "-" + strconv.Uitoa64(i.Dev) + "-" +
        strconv.Uitoa64(i.Ino) + "-" + strconv.Itoa64(i.Blocks) + "-" +
        strconv.Itoa64(i.Mtime_ns)

    return m, nil
}

func next(p *Process) bool {
    body := Car(p.Scratch).(*Method).Func.Body
    
    switch t := body.(type) {
    case Function:
        p.ReplaceState(psExecBuiltin)
        
    case *Integer:
        p.ReplaceState(t.Int())
        return true
        
    default:
        p.ReplaceState(psExecApplication)
    }

    return false
}

func rpipe(c Cell) *os.File {
    r := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
    return r.GetValue().(*Channel).ReadEnd()
}

func run(p *Process) {
    defer func(saved Process) {
        r := recover()
        if r == nil {
            return
        }

        fmt.Printf("oh: %v\n", r)

        *p = saved

        p.Code = Null
        p.Scratch = Cons(False, p.Scratch)
        p.Stack = Cdr(p.Stack)
    }(*p)

    for p.Stack != Null {
        switch state := p.GetState(); state {
        case psNone:
            return

        case psDoEvalCommand:
            switch Car(p.Scratch).(type) {
            case *String, *Symbol:
                p.ReplaceState(psExecExternal)

            default:
                if next(p) {
                    continue
                }
            }

            p.NewState(psEvalArguments)

            fallthrough
        case psEvalArguments:
            p.Scratch = Cons(nil, p.Scratch)

            p.ReplaceState(psDoEvalArguments)

            fallthrough
        case psDoEvalArguments:
            if p.Code == Null {
                break
            }

            p.SaveState(SaveCode, Cdr(p.Code))

            p.Code = Car(p.Code)

            p.NewState(psEvalElement)

            fallthrough
        case psEvalElement:
            if p.Code != Null && IsCons(p.Code) {
                if IsAtom(Cdr(p.Code)) {
                    p.ReplaceState(psEvalAccess)
                } else {
                    p.ReplaceState(psEvalCommand)
                    continue
                }
            } else if sym, ok := p.Code.(*Symbol); ok {
                if c := Resolve(p.Lexical, p.Dynamic, sym); c != nil {
                    p.Scratch = Cons(c.GetValue(), p.Scratch)
                } else {
                    p.Scratch = Cons(sym, p.Scratch)
                }
                break
            } else {
                p.Scratch = Cons(p.Code, p.Scratch)
                break
            }

            fallthrough
        case psEvalAccess:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.NewState(psExecAccess)
            p.SaveState(SaveCode, Cdr(p.Code))

            p.Code = Car(p.Code)

            p.NewState(psEvalElement)
            continue

        case psBlock:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.Dynamic = NewEnv(p.Dynamic)
            p.Lexical = NewScope(p.Lexical)

            p.NewState(psEvalBlock)

            fallthrough
        case psEvalBlock:
            if !IsCons(p.Code) || !IsCons(Car(p.Code)) {
                break
            }

            if Cdr(p.Code) == Null ||
                !IsCons(Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(SaveCode, Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)

            fallthrough
        case psEvalCommand:
            p.ReplaceState(psDoEvalCommand)
            p.SaveState(SaveCode, Cdr(p.Code))

            p.Code = Car(p.Code)

            p.NewState(psEvalElement)
            continue

        case psEvalFor:
            p.ReplaceState(psExecFor)
            args := p.Arguments()

            /* Second argument to for is a method. First argument is a list. */
            p.Code = Car(args)
            SetCar(p.Scratch, Cadr(args))
            p.Scratch = Cons(Null, p.Scratch)

            fallthrough
        case psExecFor:
            r := Car(p.Scratch)
            p.Scratch = Cdr(p.Scratch)

            if p.Code == Null {
                SetCar(p.Scratch, r)
                break
            }

            p.SaveState(SaveCode, Cdr(p.Code))

            p.Scratch = Cons(Car(p.Scratch), p.Scratch)
            p.Scratch = Cons(nil, p.Scratch)
            p.Scratch = Cons(Car(p.Code), p.Scratch)

            p.NewState(psExecApplication)

            fallthrough
        case psExecApplication:
            args := p.Arguments()

            m := Car(p.Scratch).(*Method)
            if m.Self == nil {
                args = expand(args)
            }

            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.Code = m.Func.Body
            p.Dynamic = NewEnv(p.Dynamic)
            p.Lexical = NewScope(m.Func.Lexical)

            param := m.Func.Param
            for args != Null && param != Null {
                p.Lexical.Public(Car(param), Car(args))
                args, param = Cdr(args), Cdr(param)
            }
            p.Lexical.Public(NewSymbol("$args"), args)
            p.Lexical.Public(NewSymbol("$self"), m.Self)
            p.Lexical.Public(NewSymbol("return"),
                p.Continuation(psReturn))

            p.NewState(psEvalBlock)
            continue

        case psSet:
            p.ReplaceState(psExecSet)
            p.NewState(psEvalArguments)
            p.SaveState(SaveCode, Cdr(p.Code))

            p.Code = Car(p.Code)

            p.NewState(psEvalReference)

            fallthrough
        case psEvalReference:
            p.RemoveState()

            p.Scratch = Cdr(p.Scratch)

            if p.Code != Null && IsCons(p.Code) {
                p.SaveState(SaveLexical)
                p.NewState(psExecReference)
                p.SaveState(SaveCode, Cdr(p.Code))

                p.Code = Car(p.Code)
                
                p.NewState(psChangeScope)
                p.NewState(psEvalElement)
                continue
            }

            p.NewState(psExecReference)

            fallthrough
        case psExecReference:
            k := p.Code.(*Symbol)
            v := Resolve(p.Lexical, p.Dynamic, k)
            if v == nil {
                panic("'" + k.String() + "' is not defined")
            }

            p.Scratch = Cons(v, p.Scratch)

        case psDefine, psPublic:
            p.RemoveState()

            l := Car(p.Scratch).(*Method).Self
            if p.Lexical != l {
                p.SaveState(SaveLexical)
                p.Lexical = l
            }

            if state == psDefine {
                p.NewState(psExecDefine)
            } else {
                p.NewState(psExecPublic)
            }

            k := Car(p.Code)

            p.Code = Cadr(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.SaveState(SaveCode | SaveLexical, k)
            p.NewState(psEvalElement)
            continue

        case psExecDefine, psExecPublic:
            if state == psDefine {
                p.Lexical.Private(p.Code, Car(p.Scratch))
            } else {
                p.Lexical.Public(p.Code, Car(p.Scratch))
            }

        case psDynamic, psSetenv:
            k := Car(p.Code)

            if state == psSetenv {
                if !strings.HasPrefix(k.String(), "$") {
                    break
                }

                p.ReplaceState(psExecSetenv)
            } else {
                p.ReplaceState(psExecDynamic)
            }

            p.Code = Cadr(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.SaveState(SaveCode | SaveDynamic, k)
            p.NewState(psEvalElement)
            continue

        case psExecDynamic, psExecSetenv:
            k := p.Code
            v := Car(p.Scratch)

            if state == psExecSetenv {
                s := Raw(v)
                os.Setenv(strings.TrimLeft(k.String(), "$"), s)
            }

            p.Dynamic.Add(k, v)

        case psWhile:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.NewState(psEvalWhileTest)

            fallthrough
        case psEvalWhileTest:
            p.ReplaceState(psEvalWhileBody)
            p.SaveState(SaveCode, p.Code)

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psEvalWhileBody:
            if !Car(p.Scratch).Bool() {
                break
            }

            p.ReplaceState(psEvalWhileTest)
            p.SaveState(SaveCode, p.Code)

            p.Code = Cdr(p.Code)

            p.NewState(psEvalBlock)
            continue

        case psAnd:
            SetCar(p.Scratch, True)
            p.ReplaceState(psEvalAnd)

            fallthrough
        case psEvalAnd:
            prev := Car(p.Scratch).Bool()
            SetCar(p.Scratch, NewBoolean(prev))

            if p.Code == Null || !prev {
                break
            }

            if Cdr(p.Code) == Null {
                p.ReplaceState(psEvalElement)
            } else {
                p.SaveState(SaveCode, Cdr(p.Code))
                p.NewState(psEvalElement)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)
            continue

        case psAndf:
            p.Scratch = Cons(True, p.Scratch)
            p.ReplaceState(psEvalAndf)

            fallthrough
        case psEvalAndf:
            if !IsCons(p.Code) || !IsCons(Car(p.Code)) {
                break
            }

            if !Car(p.Scratch).Bool() {
                break
            }

            if Cdr(p.Code) == Null ||
                !IsCons(Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(SaveCode, Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)
            continue

        case psOr:
            SetCar(p.Scratch, False)
            p.ReplaceState(psEvalOr)

            fallthrough
        case psEvalOr:
            prev := Car(p.Scratch).Bool()
            SetCar(p.Scratch, NewBoolean(prev))

            if p.Code == Null || prev {
                break
            }

            if Cdr(p.Code) == Null {
                p.ReplaceState(psEvalElement)
            } else {
                p.SaveState(SaveCode, Cdr(p.Code))
                p.NewState(psEvalElement)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)
            continue

        case psOrf:
            p.Scratch = Cons(False, p.Scratch)
            p.ReplaceState(psEvalOrf)

            fallthrough
        case psEvalOrf:
            if !IsCons(p.Code) || !IsCons(Car(p.Code)) {
                break
            }

            if Car(p.Scratch).Bool() {
                break
            }

            if Cdr(p.Code) == Null ||
                !IsCons(Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(SaveCode, Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)
            continue

        case psChangeScope:
            p.Lexical = Car(p.Scratch).(Interface)
            p.Scratch = Cdr(p.Scratch)

        case psExecAccess:
            p.Dynamic = nil
            p.Lexical = Car(p.Scratch).(Interface)
            p.Scratch = Cdr(p.Scratch)
            p.ReplaceState(psEvalElement)
            continue

        case psExecBuiltin:
            args := p.Arguments()

            m := Car(p.Scratch).(*Method)
            if m.Self == nil {
                args = expand(args)
            }

            if m.Func.Body.(Function)(p, args) {
                continue
            }

        case psExecExternal:
            args := p.Arguments()

            if external(p, args) {
                continue
            }

        case psExecIf:
            if !Car(p.Scratch).Bool() {
                p.Code = Cdr(p.Code)

                for Car(p.Code) != Null &&
                    !IsAtom(Car(p.Code)) {
                    p.Code = Cdr(p.Code)
                }

                p.Code = Cdr(p.Code)
            }

            if p.Code == Null {
                break
            }

            p.ReplaceState(psEvalBlock)
            continue

        case psExecImport:
            n := Raw(Car(p.Scratch))

            k, err := module(n)
            if err != nil {
                SetCar(p.Scratch, False)
                break
            }

            v := Resolve(p.Lexical, p.Dynamic, NewSymbol(k))
            if v != nil {
                SetCar(p.Scratch, v.GetValue())
                break
            }

            p.ReplaceState(psCreateModule)
            p.SaveState(SaveCode, NewSymbol(n))
            p.NewState(psExecSource)

            fallthrough
        case psExecSource:
            f, err := os.OpenFile(
                Raw(Car(p.Scratch)),
                os.O_RDONLY, 0666)
            if err != nil {
                panic(err)
            }

            p.Code = Null
            ParseFile(f, func (c Cell) {
                p.Code = AppendTo(p.Code, c)
            })

            if state == psExecImport {
                p.RemoveState()
                p.SaveState(SaveDynamic | SaveLexical)

                p.Dynamic = NewEnv(p.Dynamic)
                p.Lexical = NewScope(p.Lexical)

                p.NewState(psExecObject)
                p.NewState(psEvalBlock)
            } else {
                if p.Code == Null {
                    break
                }

                p.ReplaceState(psEvalBlock)
            }
            continue

        case psCreateModule:
            k, _ := module(p.Code.String())

            s := p.Lexical
            for s.Prev() != nil {
                s = s.Prev()
            }
            p.Lexical.Private(NewSymbol(k), Car(p.Scratch))

        case psExecObject:
            SetCar(p.Scratch, NewObject(p.Lexical))

        case psExecSet:
            args := p.Arguments()

            r := Car(p.Scratch).(*Reference)

            r.SetValue(Car(args))
            SetCar(p.Scratch, r.GetValue())

        case psExecSplice:
            l := Car(p.Scratch)
            p.Scratch = Cdr(p.Scratch)

            if !IsCons(l) {
                break
            }

            for l != Null {
                p.Scratch = Cons(Car(l), p.Scratch)
                l = Cdr(l)
            }

            /* Command states */
        case psBackground:
            child := NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psEvalCommand)

            child.Code = Car(p.Code)
            SetCar(p.Scratch, True)

            go run(child)

        case psBacktick:
            c := channel(p, nil, nil)

            child := NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psPipeChild)

            s := NewSymbol("$stdout")
            child.SaveState(SaveCode, s)

            child.Code = Car(p.Code)
            child.Dynamic.Add(s, c)

            child.NewState(psEvalCommand)

            go run(child)

			b := bufio.NewReader(rpipe(c))

//            g := Resolve(c, nil, NewSymbol("guts"))
//            b := bufio.NewReader(g.GetValue().(*Channel).ReadEnd())

            l := Null

            done := false
            line, err := b.ReadString('\n')
            for !done {
                if err != nil {
                    done = true
                }

                line = strings.Trim(line, " \t\n")

                if len(line) > 0 {
                    l = AppendTo(l, NewString(line))
                }

                line, err = b.ReadString('\n')
            }

            SetCar(p.Scratch, l)

        case psBuiltin, psMethod:
            param := Null
            for !IsCons(Car(p.Code)) {
                param = Cons(Car(p.Code), param)
                p.Code = Cdr(p.Code)
            }

            if state == psBuiltin {
                SetCar(
                    p.Scratch,
                    function(p.Code, Reverse(param), p.Lexical.Expose()))
            } else {
                SetCar(
                    p.Scratch,
                    method(p.Code, Reverse(param), p.Lexical.Expose()))
            }

        case psFor:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.NewState(psEvalFor)
            p.NewState(psEvalArguments)
            continue

        case psIf:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.Dynamic = NewEnv(p.Dynamic)
            p.Lexical = NewScope(p.Lexical)

            p.NewState(psExecIf)
            p.SaveState(SaveCode, Cdr(p.Code))
            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psImport, psSource:
            if state == psImport {
                p.ReplaceState(psExecImport)
            } else {
                p.ReplaceState(psExecSource)
            }

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psObject:
            p.RemoveState()
            p.SaveState(SaveDynamic | SaveLexical)

            p.Dynamic = NewEnv(p.Dynamic)
            p.Lexical = NewScope(p.Lexical)

            p.NewState(psExecObject)
            p.NewState(psEvalBlock)
            continue

        case psQuote:
            SetCar(p.Scratch, Car(p.Code))

        case psReturn:
            p.Code = Car(p.Code)

            m := Car(p.Scratch).(*Method)
            p.Scratch = Car(m.Func.Param)
            p.Stack = Cadr(m.Func.Param)

            p.NewState(psEvalElement)
            continue

        case psSpawn:
            child := NewProcess(psNone, p.Dynamic, p.Lexical)

            child.Scratch = Cons(Null, child.Scratch)
            child.NewState(psEvalBlock)

            child.Code = p.Code

            go run(child)

        case psSplice:
            p.ReplaceState(psExecSplice)

            p.Code = Car(p.Code)
            p.Scratch = Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psPipeStderr, psPipeStdout:
            p.RemoveState()
            p.SaveState(SaveDynamic)

            c := channel(p, nil, nil)

            child := NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psPipeChild)

            var s *Symbol
            if state == psPipeStderr {
                s = NewSymbol("$stderr")
            } else {
                s = NewSymbol("$stdout")
            }
            child.SaveState(SaveCode, s)

            child.Code = Car(p.Code)
            child.Dynamic.Add(s, c)

            child.NewState(psEvalCommand)

            go run(child)

            p.Code = Cadr(p.Code)
            p.Dynamic = NewEnv(p.Dynamic)
            p.Scratch = Cdr(p.Scratch)

            p.Dynamic.Add(NewSymbol("$stdin"), c)

            p.NewState(psPipeParent)
            p.NewState(psEvalCommand)
            continue
            
        case psPipeChild:
            c := Resolve(p.Lexical, p.Dynamic, p.Code.(*Symbol)).GetValue()
//            c = Resolve(
//                c.GetValue().(Interface).Expose(), nil,
//                NewSymbol("guts"))
//            c.GetValue().(*Channel).WriteEnd().Close()
			wpipe(c).Close()
            
        case psPipeParent:
            c := Resolve(p.Lexical, p.Dynamic, NewSymbol("$stdin")).GetValue()
//            c = Resolve(
//                c.GetValue().(Interface).Expose(), nil,
//                NewSymbol("guts"))
//            c.GetValue().(*Channel).Close()
			rpipe(c).Close()

        case psAppendStderr, psAppendStdout, psRedirectStderr,
            psRedirectStdin, psRedirectStdout:
            p.RemoveState()
            p.SaveState(SaveDynamic)

            initial := NewInteger(state)

            p.NewState(psRedirectCleanup)
            p.NewState(psEvalCommand)
            p.SaveState(SaveCode, Cadr(p.Code))
            p.NewState(psRedirectSetup)
            p.SaveState(SaveCode, initial)

            p.Code = Car(p.Code)
            p.Dynamic = NewEnv(p.Dynamic)
            p.Scratch = Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psRedirectSetup:
            flags, name := 0, ""
            initial := p.Code.(Atom).Int()
            
            switch initial {
            case psAppendStderr:
                flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
                name = "$stderr"
                
            case psAppendStdout:
                flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
                name = "$stdout"
                
                
            case psRedirectStderr:
                flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
                name = "$stderr"
                
            case psRedirectStdin:
                flags = os.O_RDONLY
                name = "$stdin"
                
            case psRedirectStdout:
                flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
                name = "$stdout"
            }

            c, ok := Car(p.Scratch).(Interface)
            if !ok {
                n := Raw(Car(p.Scratch))
                
                f, err := os.OpenFile(n, flags, 0666)
                if err != nil {
                    panic(err)
                }

                if name == "$stdin" {
                    c = channel(p, f, nil)
                } else {
                    c = channel(p, nil, f)
                }
                SetCar(p.Scratch, c)

                r := Resolve(c, nil, NewSymbol("guts"))
                ch := r.GetValue().(*Channel)

                ch.Implicit = true
            }

            p.Dynamic.Add(NewSymbol(name), c)

        case psRedirectCleanup:
            c := Cadr(p.Scratch).(Interface)
            r := Resolve(c, nil, NewSymbol("guts"))
            ch := r.GetValue().(*Channel)

            if ch.Implicit {
                ch.Close()
            }

            SetCdr(p.Scratch, Cddr(p.Scratch))

        default:
            if state >= SaveMax {
                panic(fmt.Sprintf("command not found: %s", p.Code))
            } else {
                p.RestoreState()
            }
        }

        p.RemoveState()
    }
}

func wpipe(c Cell) *os.File {
    w := Resolve(c.(Interface).Expose(), nil, NewSymbol("guts"))
    return w.GetValue().(*Channel).WriteEnd()
}

func Evaluate(c Cell) {
    proc0.NewState(psEvalCommand)
    proc0.Code = c
    
    run(proc0)

    if proc0.Stack == Null {
        os.Exit(ExitStatus())
    }

    proc0.Scratch = Cdr(proc0.Scratch)
}

func ExitStatus() int {
    s, ok := Car(proc0.Scratch).(*Status)
    if !ok {
        return 0
    }
    return int(s.Int())
}

func Start() {
    proc0 = NewProcess(psNone, nil, nil)

    proc0.Scratch = Cons(NewStatus(0), proc0.Scratch)

    e, s := proc0.Dynamic, proc0.Lexical.Expose()

    e.Add(NewSymbol("$stdin"), channel(proc0, os.Stdin, nil))
    e.Add(NewSymbol("$stdout"), channel(proc0, nil, os.Stdout))
    e.Add(NewSymbol("$stderr"), channel(proc0, nil, os.Stderr))

    if wd, err := os.Getwd(); err == nil {
        e.Add(NewSymbol("$cwd"), NewSymbol(wd))
    }

    s.PrivateState("and", psAnd)
    s.PrivateState("block", psBlock)
    s.PrivateState("backtick", psBacktick)
    s.PrivateState("define", psDefine)
    s.PrivateState("dynamic", psDynamic)
    s.PrivateState("for", psFor)
    s.PrivateState("builtin", psBuiltin)
    s.PrivateState("if", psIf)
    s.PrivateState("import", psImport)
    s.PrivateState("source", psSource)
    s.PrivateState("method", psMethod)
    s.PrivateState("object", psObject)
    s.PrivateState("or", psOr)
    s.PrivateState("quote", psQuote)
    s.PrivateState("set", psSet)
    s.PrivateState("setenv", psSetenv)
    s.PrivateState("spawn", psSpawn)
    s.PrivateState("splice", psSplice)
    s.PrivateState("while", psWhile)

    s.PublicState("public", psPublic)

    s.PrivateState("background", psBackground)
    s.PrivateState("pipe-stdout", psPipeStdout)
    s.PrivateState("pipe-stderr", psPipeStderr)
    s.PrivateState("redirect-stdin", psRedirectStdin)
    s.PrivateState("redirect-stdout", psRedirectStdout)
    s.PrivateState("redirect-stderr", psRedirectStderr)
    s.PrivateState("append-stdout", psAppendStdout)
    s.PrivateState("append-stderr", psAppendStderr)
    s.PrivateState("andf", psAndf)
    s.PrivateState("orf", psOrf)

    /* Builtins. */
    s.PrivateFunction("cd", func(p *Process, args Cell) bool {
        err, status := os.Chdir(Raw(Car(args))), 0
        if err != nil {
            status = int(err.(*os.PathError).Error.(os.Errno))
        }
        SetCar(p.Scratch, NewStatus(int64(status)))

        if wd, err := os.Getwd(); err == nil {
            p.Dynamic.Add(NewSymbol("$cwd"), NewSymbol(wd))
        }

        return false
    })
    s.PrivateFunction("debug", func(p *Process, args Cell) bool {
        debug(p, "debug")

        return false
    })
    s.PrivateFunction("exit", func(p *Process, args Cell) bool {
        var status int64 = 0

        a, ok := Car(args).(Atom)
        if ok {
            status = a.Status()
        }

        p.Scratch = List(NewStatus(status))
        p.Stack = Null

        return true
    })

    s.PublicMethod("child", func(p *Process, args Cell) bool {
        o := Car(p.Scratch).(*Method).Self.Expose()

        SetCar(p.Scratch, NewObject(NewScope(o)))

        return false
    })
    s.PublicMethod("clone", func(p *Process, args Cell) bool {
        o := Car(p.Scratch).(*Method).Self.Expose()

        SetCar(p.Scratch, NewObject(o.Copy()))

        return false
    })

    s.PrivateMethod("apply", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Car(args))
        next(p)
        
        p.Scratch = Cons(nil, p.Scratch)
        for args = Cdr(args); args != Null; args = Cdr(args) {
            p.Scratch = Cons(Car(args), p.Scratch)
        }
        
        return true
    })
    s.PrivateMethod("append", func(p *Process, args Cell) bool {
        /*
         * NOTE: Our append works differently than Scheme's append.
         *       To mimic Scheme's behavior used append l1 @l2 ... @ln
         */

        /* TODO: We should just copy this list: ... */
        l := Car(args)

        /* TODO: ... and then set it's cdr to cdr(args). */
        argv := make([]Cell, 0)
        for args = Cdr(args); args != Null; args = Cdr(args) {
            argv = append(argv, Car(args))
        }
        
        SetCar(p.Scratch, Append(l, argv...))
        
        return false
    })
    s.PrivateMethod("car", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caar(args))

        return false
    })
    s.PrivateMethod("cdr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdar(args))

        return false
    })
    s.PrivateMethod("caar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caaar(args))

        return false
    })
    s.PrivateMethod("cadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cadar(args))

        return false
    })
    s.PrivateMethod("cdar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdaar(args))

        return false
    })
    s.PrivateMethod("cddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cddar(args))

        return false
    })
    s.PrivateMethod("caaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Car(Caaar(args)))

        return false
    })
    s.PrivateMethod("caadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Car(Cadar(args)))

        return false
    })
    s.PrivateMethod("cadar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Car(Cdaar(args)))

        return false
    })
    s.PrivateMethod("caddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Car(Cddar(args)))

        return false
    })
    s.PrivateMethod("cdaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdr(Caaar(args)))

        return false
    })
    s.PrivateMethod("cdadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdr(Cadar(args)))

        return false
    })
    s.PrivateMethod("cddar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdr(Cdaar(args)))

        return false
    })
    s.PrivateMethod("cdddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdr(Cddar(args)))

        return false
    })
    s.PrivateMethod("caaaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caar(Caaar(args)))

        return false
    })
    s.PrivateMethod("caaadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caar(Cadar(args)))

        return false
    })
    s.PrivateMethod("caadar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caar(Cdaar(args)))

        return false
    })
    s.PrivateMethod("caaddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Caar(Cddar(args)))

        return false
    })
    s.PrivateMethod("cadaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cadr(Caaar(args)))

        return false
    })
    s.PrivateMethod("cadadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cadr(Cadar(args)))

        return false
    })
    s.PrivateMethod("caddar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cadr(Cdaar(args)))

        return false
    })
    s.PrivateMethod("cadddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cadr(Cddar(args)))

        return false
    })
    s.PrivateMethod("cdaaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdar(Caaar(args)))

        return false
    })
    s.PrivateMethod("cdaadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdar(Cadar(args)))

        return false
    })
    s.PrivateMethod("cdadar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdar(Cdaar(args)))

        return false
    })
    s.PrivateMethod("cdaddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cdar(Cddar(args)))

        return false
    })
    s.PrivateMethod("cddaar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cddr(Caaar(args)))

        return false
    })
    s.PrivateMethod("cddadr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cddr(Cadar(args)))

        return false
    })
    s.PrivateMethod("cdddar", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cddr(Cdaar(args)))

        return false
    })
    s.PrivateMethod("cddddr", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cddr(Cddar(args)))

        return false
    })
    s.PrivateMethod("cons", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Cons(Car(args), Cadr(args)))

        return false
    })
    s.PrivateMethod("eval", func(p *Process, args Cell) bool {
        p.ReplaceState(psEvalCommand)

        p.Code = Car(args)
        p.Scratch = Cdr(p.Scratch)

        return true
    })
    s.PrivateMethod("length", func(p *Process, args Cell) bool {
        var l int64 = 0

        switch c := Car(args); c.(type) {
        case *String, *Symbol:
            l = int64(len(Raw(c)))
        default:
            l = Length(c)
        }

        SetCar(p.Scratch, NewInteger(l))

        return false
    })
    s.PrivateMethod("list", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, args);

        return false
    })
    s.PrivateMethod("list-to-string", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

        SetCar(p.Scratch, NewString(s))

        return false
    })
    s.PrivateMethod("list-to-symbol", func(p *Process, args Cell) bool {
		s := ""
		for l := Car(args); l != Null; l = Cdr(l) {
			s = fmt.Sprintf("%s%c", s, int(Car(l).(Atom).Int()))
		}

        SetCar(p.Scratch, NewSymbol(s))

        return false
    })
    s.PrivateMethod("open", func(p *Process, args Cell) bool {
        name := Raw(Car(args))
        mode := Raw(Cadr(args))

        flags := os.O_CREATE

        if strings.IndexAny(mode, "r") != -1 {
            flags |= os.O_WRONLY
        } else if strings.IndexAny(mode, "aw") != -1 {
            flags |= os.O_RDONLY
        } else {
            flags |= os.O_RDWR
        }

        if strings.IndexAny(mode, "a") != -1 {
            flags |= os.O_APPEND
        }

        f, err := os.OpenFile(name, flags, 0666)
        if err != nil {
            panic(err)
        }

        SetCar(p.Scratch, channel(p, f, f))

        return false
    })
    s.PrivateMethod("reverse", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, Reverse(Car(args)))

        return false
    })
    s.PrivateMethod("set-car", func(p *Process, args Cell) bool {
        SetCar(Car(args), Cadr(args))
        SetCar(p.Scratch, Cadr(args))

        return false
    })
    s.PrivateMethod("set-cdr", func(p *Process, args Cell) bool {
        SetCdr(Car(args), Cadr(args))
        SetCar(p.Scratch, Cadr(args))

        return false
    })
    s.PrivateMethod("sprintf", func(p *Process, args Cell) bool {
        f := Raw(Car(args))
        
        argv := []interface{}{}
        for l := Cdr(args); l != Null; l = Cdr(l) {
            switch t := Car(l).(type) {
            case *Boolean:
                argv = append(argv, *t)
            case *Integer:
                argv = append(argv, *t)
            case *Status:
                argv = append(argv, *t)
            case *Float:
                argv = append(argv, *t)
            default:
                argv = append(argv, Raw(t))
            }
        }
        
        s := fmt.Sprintf(f, argv...)
        SetCar(p.Scratch, NewString(s))

        return false
    })
    s.PrivateMethod("text-to-list", func(p *Process, args Cell) bool {
		l := Null
		for _, char := range Raw(Car(args)) {
			l = Cons(NewInteger(int64(char)), l)
		}

        SetCar(p.Scratch, Reverse(l))

        return false
    })

    /* Predicates. */
    s.PrivateMethod("is-atom", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, NewBoolean(IsAtom(Car(args))))

        return false
    })
    s.PrivateMethod("is-boolean",
        func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Boolean)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-channel",
        func(p *Process, args Cell) bool {
        o, ok := Car(args).(Interface)
        if ok {
            ok = false
            c := Resolve(o.Expose(), nil, NewSymbol("guts"))
            if c != nil {
                _, ok = c.GetValue().(*Channel)
            }
        } 

        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-cons", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, NewBoolean(IsCons(Car(args))))

        return false
    })
    s.PrivateMethod("is-float", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Float)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-integer",
        func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Integer)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-list", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, NewBoolean(IsList(Car(args))))

        return false
    })
    s.PrivateMethod("is-method", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Method)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-null", func(p *Process, args Cell) bool {
        ok := Car(args) == Null
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-number", func(p *Process, args Cell) bool {
        _, ok := Car(args).(Number)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-object", func(p *Process, args Cell) bool {
        _, ok := Car(args).(Interface)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-status", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Status)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-string", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*String)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-symbol", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Symbol)
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-text", func(p *Process, args Cell) bool {
        _, ok := Car(args).(*Symbol)
        if !ok {
            _, ok = Car(args).(*String)
        }
        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })

    /* Generators. */
    s.PrivateMethod("boolean", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, NewBoolean(Car(args).Bool()))

        return false
    })
    s.PrivateMethod("channel", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, channel(p, nil, nil))

        return false
    })
    s.PrivateMethod("float", func(p *Process, args Cell) bool {
        SetCar(p.Scratch,
            NewFloat(Car(args).(Atom).Float()))

        return false
    })
    s.PrivateMethod("integer", func(p *Process, args Cell) bool {
        SetCar(p.Scratch,
            NewInteger(Car(args).(Atom).Int()))

        return false
    })
    s.PrivateMethod("status", func(p *Process, args Cell) bool {
        SetCar(p.Scratch,
            NewStatus(Car(args).(Atom).Status()))

        return false
    })
    s.PrivateMethod("string", func(p *Process, args Cell) bool {
        SetCar(p.Scratch,
            NewString(Car(args).String()))

        return false
    })
    s.PrivateMethod("symbol", func(p *Process, args Cell) bool {
        SetCar(p.Scratch,
            NewSymbol(Raw(Car(args))))

        return false
    })

    /* Relational. */
    s.PrivateMethod("eq", func(p *Process, args Cell) bool {
        prev := Car(args)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args)

            if !prev.Equal(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("ge", func(p *Process, args Cell) bool {
        prev := Car(args).(Atom)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args).(Atom)

            if prev.Less(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("gt", func(p *Process, args Cell) bool {
        prev := Car(args).(Atom)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args).(Atom)

            if !prev.Greater(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("is", func(p *Process, args Cell) bool {
        prev := Car(args)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args)

            if prev != curr {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("le", func(p *Process, args Cell) bool {
        prev := Car(args).(Atom)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args).(Atom)

            if prev.Greater(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("lt", func(p *Process, args Cell) bool {
        prev := Car(args).(Atom)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args).(Atom)

            if !prev.Less(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("match", func(p *Process, args Cell) bool {
        pattern := Raw(Car(args))
        text := Raw(Cadr(args))

        ok, err := path.Match(pattern, text)
        if err != nil {
            panic(err)
        }

        SetCar(p.Scratch, NewBoolean(ok))

        return false
    })
    s.PrivateMethod("ne", func(p *Process, args Cell) bool {
        /*
         * This should really check to make sure no arguments are equal.
         * Currently it only checks whether adjacent pairs are not equal.
         */

        prev := Car(args)

        SetCar(p.Scratch, False)

        for Cdr(args) != Null {
            args = Cdr(args)
            curr := Car(args)

            if prev.Equal(curr) {
                return false
            }

            prev = curr
        }

        SetCar(p.Scratch, True)
        return false
    })
    s.PrivateMethod("not", func(p *Process, args Cell) bool {
        SetCar(p.Scratch, NewBoolean(!Car(args).Bool()))

        return false
    })

    /* Arithmetic. */
    s.PrivateMethod("add", func(p *Process, args Cell) bool {
        acc := Car(args).(Atom)

        for Cdr(args) != Null {
            args = Cdr(args)
            acc = acc.Add(Car(args))

        }

        SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("sub", func(p *Process, args Cell) bool {
        acc := Car(args).(Number)

        for Cdr(args) != Null {
            args = Cdr(args)
            acc = acc.Subtract(Car(args))
        }

        SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("div", func(p *Process, args Cell) bool {
        acc := Car(args).(Number)

        for Cdr(args) != Null {
            args = Cdr(args)
            acc = acc.Divide(Car(args))
        }

        SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("mod", func(p *Process, args Cell) bool {
        acc := Car(args).(Number)

        for Cdr(args) != Null {
            args = Cdr(args)
            acc = acc.Modulo(Car(args))
        }

        SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("mul", func(p *Process, args Cell) bool {
        acc := Car(args).(Atom)

        for Cdr(args) != Null {
            args = Cdr(args)
            acc = acc.Multiply(Car(args))
        }

        SetCar(p.Scratch, acc)
        return false
    })

    e.Add(NewSymbol("$$"), NewInteger(int64(os.Getpid())))

    /* Command-line arguments */
    args := Null
    if len(os.Args) > 1 {
        e.Add(NewSymbol("$0"), NewSymbol(os.Args[1]))

        for i, v := range os.Args[2:] {
            e.Add(NewSymbol("$" + strconv.Itoa(i + 1)), NewSymbol(v))
        }

        for i := len(os.Args) - 1; i > 1; i-- {
            args = Cons(NewSymbol(os.Args[i]), args)
        }
    } else {
        e.Add(NewSymbol("$0"), NewSymbol(os.Args[0]))
    }
    e.Add(NewSymbol("$args"), args)

    /* Environment variables. */
    for _, s := range os.Environ() {
        kv := strings.SplitN(s, "=", 2)
        e.Add(NewSymbol("$" + kv[0]), NewSymbol(kv[1]))
    }

    Parse(bufio.NewReader(strings.NewReader(`
define echo: builtin: $stdout::write @$args
define expand: builtin: return $args
define printf: method: echo: sprintf (car $args) @(cdr $args)
define read: builtin: $stdin::read
define readline: builtin: $stdin::readline
define write: method: $stdout::write @$args
define list-tail: method k x {
    if k {
        list-tail (sub k 1): cdr x
    } else {
        return x
    }
}
define list-ref: method k x: car: list-tail k x
`)), Evaluate)

    /* Read and execute rc script if it exists. */
    rc := filepath.Join(os.Getenv("HOME"), ".ohrc")
    if _, err := os.Stat(rc); err == nil {
        proc0.NewState(psEvalCommand)
        proc0.Code = List(NewSymbol("source"), NewSymbol(rc))
        
        run(proc0)
        
        proc0.Scratch = Cdr(proc0.Scratch)
    }
}
