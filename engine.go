/* Released under an MIT-style license. See LICENSE. */

package engine

import (
    "bufio"
    "exec"
    "fmt"
    "os"
    "path"
    "path/filepath"
    "strings"
    "strconv"
    "./cell"
)

const (
    psNone = 0

    psChangeScope = cell.SaveMax + iota
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

var main *cell.Process

func channel(p *cell.Process, r, w *os.File) cell.Interface {
    c, ch := cell.NewScope(p.Lexical), cell.NewChannel(r, w)

    var read cell.Function = func (p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, ch.Read())                                   
        return false
    }

    var readline cell.Function = func (p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, ch.ReadLine())
        return false
    }

    var write cell.Function = func (p *cell.Process, args cell.Cell) bool {
        ch.Write(args)
        cell.SetCar(p.Scratch, cell.True)
        return false
    }

    c.Public(cell.NewSymbol("guts"), ch)
    c.Public(cell.NewSymbol("read"), method(read, cell.Null, c))
    c.Public(cell.NewSymbol("readline"), method(readline, cell.Null, c))
    c.Public(cell.NewSymbol("write"), method(write, cell.Null, c))

    return cell.NewObject(c)
}

func debug(p *cell.Process, s string) {
    fmt.Printf("%s: p.Code = %v, p.Scratch = %v\n", s, p.Code, p.Scratch)
}

func expand(args cell.Cell) cell.Cell {
    list := cell.Null

    for args != cell.Null {
        c := cell.Car(args)

        s := cell.Raw(c)
        if _, ok := c.(*cell.Symbol); ok {
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
                        list = cell.AppendTo(list, cell.NewSymbol(e))
                    }
                }
            } else {
                list = cell.AppendTo(list, cell.NewSymbol(s))
            }
        } else {
            list = cell.AppendTo(list, cell.NewSymbol(s))
        }
        args = cell.Cdr(args)
    }   

    return list
}

func external(p *cell.Process, args cell.Cell) bool {
    name, err := exec.LookPath(cell.Raw(cell.Car(p.Scratch)))

    cell.SetCar(p.Scratch, cell.False)

    if err != nil {
        panic(err)
    }

    argv := []string{name}

    for args = expand(args); args != cell.Null; args = cell.Cdr(args) {
        argv = append(argv, cell.Car(args).String())
    }

    c := cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol("$cwd"))
    dir := c.GetValue().String()

    fd := []*os.File{os.Stdin, os.Stdout, os.Stderr}

    c = cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol("$stdin"))
    c = cell.Resolve(
        c.GetValue().(cell.Interface).Expose(), nil, cell.NewSymbol("guts"))
    fd[0] = c.GetValue().(*cell.Channel).ReadEnd()

    c = cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol("$stdout"))
    c = cell.Resolve(
        c.GetValue().(cell.Interface).Expose(), nil, cell.NewSymbol("guts"))
    fd[1] = c.GetValue().(*cell.Channel).WriteEnd()

    c = cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol("$stderr"))
    c = cell.Resolve(
        c.GetValue().(cell.Interface).Expose(), nil, cell.NewSymbol("guts"))
    fd[2] = c.GetValue().(*cell.Channel).WriteEnd()

    proc, err := os.StartProcess(name, argv, &os.ProcAttr{dir, nil, fd})
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

    cell.SetCar(p.Scratch, cell.NewStatus(status))

    return false
}

func function(body, param cell.Cell, scope *cell.Scope) *cell.Method {
    return cell.NewMethod(cell.NewClosure(body, param, scope), nil)
}

func init() {
    main = cell.NewProcess(psNone, nil, nil)

    main.Scratch = cell.Cons(cell.NewStatus(0), main.Scratch)

    e, s := main.Dynamic, main.Lexical.Expose()

    e.Add(cell.NewSymbol("$stdin"), channel(main, os.Stdin, nil))
    e.Add(cell.NewSymbol("$stdout"), channel(main, nil, os.Stdout))
    e.Add(cell.NewSymbol("$stderr"), channel(main, nil, os.Stderr))

    if wd, err := os.Getwd(); err == nil {
        e.Add(cell.NewSymbol("$cwd"), cell.NewSymbol(wd))
    }

    s.PrivateState("block", psBlock)
    s.PrivateState("define", psDefine)
    s.PrivateState("dynamic", psDynamic)
    s.PrivateState("for", psFor)
    s.PrivateState("builtin", psBuiltin)
    s.PrivateState("if", psIf)
    s.PrivateState("import", psImport)
    s.PrivateState("source", psSource)
    s.PrivateState("method", psMethod)
    s.PrivateState("object", psObject)
    s.PrivateState("setenv", psSetenv)
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

    s.PrivateState("backtick", psBacktick)
    s.PrivateState("and", psAnd)
    s.PrivateState("or", psOr)
    s.PrivateState("quote", psQuote)
    s.PrivateState("set", psSet)
    s.PrivateState("spawn", psSpawn)
    s.PrivateState("splice", psSplice)

    s.PrivateFunction("cd", func(p *cell.Process, args cell.Cell) bool {
        err, status := os.Chdir(cell.Raw(cell.Car(args))), 0
        if err != nil {
            status = int(err.(*os.PathError).Error.(os.Errno))
        }
        cell.SetCar(p.Scratch, cell.NewStatus(int64(status)))

        if wd, err := os.Getwd(); err == nil {
            p.Dynamic.Add(cell.NewSymbol("$cwd"), cell.NewSymbol(wd))
        }

        return false
    })
    s.PrivateFunction("debug", func(p *cell.Process, args cell.Cell) bool {
        debug(p, "debug")

        return false
    })
    s.PrivateFunction("exit", func(p *cell.Process, args cell.Cell) bool {
        var status int64 = 0

        a, ok := cell.Car(args).(cell.Atom)
        if ok {
            status = a.Status()
        }

        p.Scratch = cell.List(cell.NewStatus(status))
        p.Stack = cell.Null

        return true
    })

    s.PublicMethod("child", func(p *cell.Process, args cell.Cell) bool {
        o := cell.Car(p.Scratch).(*cell.Method).Self.Expose()

        cell.SetCar(p.Scratch, cell.NewObject(cell.NewScope(o)))

        return false
    })
    s.PublicMethod("clone", func(p *cell.Process, args cell.Cell) bool {
        o := cell.Car(p.Scratch).(*cell.Method).Self.Expose()

        cell.SetCar(p.Scratch, cell.NewObject(o.Copy()))

        return false
    })

    s.PrivateMethod("open", func(p *cell.Process, args cell.Cell) bool {
        name := cell.Raw(cell.Car(args))
        mode := cell.Raw(cell.Cadr(args))

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

        cell.SetCar(p.Scratch, channel(p, f, f))

        return false
    })

    s.PrivateMethod("sprintf", func(p *cell.Process, args cell.Cell) bool {
        f := cell.Raw(cell.Car(args))
        
        argv := []interface{}{}
        for l := cell.Cdr(args); l != cell.Null; l = cell.Cdr(l) {
            switch t := cell.Car(l).(type) {
            case *cell.Boolean:
                argv = append(argv, *t)
            case *cell.Integer:
                argv = append(argv, *t)
            case *cell.Status:
                argv = append(argv, *t)
            case *cell.Float:
                argv = append(argv, *t)
            default:
                argv = append(argv, cell.Raw(t))
            }
        }
        
        s := fmt.Sprintf(f, argv...)
        cell.SetCar(p.Scratch, cell.NewString(s))

        return false
    })

    s.PrivateMethod("apply", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Car(args))
        next(p)
        
        p.Scratch = cell.Cons(nil, p.Scratch)
        for args = cell.Cdr(args); args != cell.Null; args = cell.Cdr(args) {
            p.Scratch = cell.Cons(cell.Car(args), p.Scratch)
        }
        
        return true
    })
    s.PrivateMethod("append", func(p *cell.Process, args cell.Cell) bool {
        /*
         * NOTE: Our append works differently than Scheme's append.
         *       To mimic Scheme's behavior used append l1 @l2 ... @ln
         */

        /* TODO: We should just copy this list: ... */
        l := cell.Car(args)

        /* TODO: ... and then set it's cdr to cdr(args). */
        argv := make([]cell.Cell, 0)
        for args = cell.Cdr(args); args != cell.Null; args = cell.Cdr(args) {
            argv = append(argv, cell.Car(args))
        }
        
        cell.SetCar(p.Scratch, cell.Append(l, argv...))
        
        return false
    })
    s.PrivateMethod("car", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caar(args))

        return false
    })
    s.PrivateMethod("cdr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdar(args))

        return false
    })
    s.PrivateMethod("caar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caaar(args))

        return false
    })
    s.PrivateMethod("cadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cadar(args))

        return false
    })
    s.PrivateMethod("cdar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdaar(args))

        return false
    })
    s.PrivateMethod("cddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cddar(args))

        return false
    })
    s.PrivateMethod("caaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Car(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("caadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Car(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("cadar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Car(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("caddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Car(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("cdaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdr(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("cdadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdr(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("cddar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdr(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("cdddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdr(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("caaaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caar(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("caaadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caar(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("caadar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caar(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("caaddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Caar(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("cadaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cadr(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("cadadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cadr(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("caddar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cadr(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("cadddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cadr(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("cdaaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdar(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("cdaadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdar(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("cdadar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdar(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("cdaddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cdar(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("cddaar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cddr(cell.Caaar(args)))

        return false
    })
    s.PrivateMethod("cddadr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cddr(cell.Cadar(args)))

        return false
    })
    s.PrivateMethod("cdddar", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cddr(cell.Cdaar(args)))

        return false
    })
    s.PrivateMethod("cddddr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cddr(cell.Cddar(args)))

        return false
    })
    s.PrivateMethod("cons", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Cons(cell.Car(args), cell.Cadr(args)))

        return false
    })
    s.PrivateMethod("eval", func(p *cell.Process, args cell.Cell) bool {
        p.ReplaceState(psEvalCommand)

        p.Code = cell.Car(args)
        p.Scratch = cell.Cdr(p.Scratch)

        return true
    })
    s.PrivateMethod("length", func(p *cell.Process, args cell.Cell) bool {
        var l int64 = 0

        switch c := cell.Car(args); c.(type) {
        case *cell.String, *cell.Symbol:
            l = int64(len(cell.Raw(c)))
        default:
            l = cell.Length(c)
        }

        cell.SetCar(p.Scratch, cell.NewInteger(l))

        return false
    })
    s.PrivateMethod("list", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, args);

        return false
    })
    s.PrivateMethod("reverse", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.Reverse(cell.Car(args)))

        return false
    })
    s.PrivateMethod("set-car", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(cell.Car(args), cell.Cadr(args))
        cell.SetCar(p.Scratch, cell.Cadr(args))

        return false
    })
    s.PrivateMethod("set-cdr", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCdr(cell.Car(args), cell.Cadr(args))
        cell.SetCar(p.Scratch, cell.Cadr(args))

        return false
    })

    /* Predicates. */
    s.PrivateMethod("is-atom", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.NewBoolean(cell.IsAtom(cell.Car(args))))

        return false
    })
    s.PrivateMethod("is-boolean",
        func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Boolean)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-channel",
        func(p *cell.Process, args cell.Cell) bool {
        o, ok := cell.Car(args).(cell.Interface)
        if ok {
            ok = false
            c := cell.Resolve(o.Expose(), nil, cell.NewSymbol("guts"))
            if c != nil {
                _, ok = c.GetValue().(*cell.Channel)
            }
        } 

        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-cons", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.NewBoolean(cell.IsCons(cell.Car(args))))

        return false
    })
    s.PrivateMethod("is-float", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Float)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-integer",
        func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Integer)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-list", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.NewBoolean(cell.IsList(cell.Car(args))))

        return false
    })
    s.PrivateMethod("is-method", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Method)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-null", func(p *cell.Process, args cell.Cell) bool {
        ok := cell.Car(args) == cell.Null
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-number", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(cell.Number)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-object", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(cell.Interface)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-status", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Status)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-string", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.String)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-symbol", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Symbol)
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("is-text", func(p *cell.Process, args cell.Cell) bool {
        _, ok := cell.Car(args).(*cell.Symbol)
        if !ok {
            _, ok = cell.Car(args).(*cell.String)
        }
        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })

    /* Generators. */
    s.PrivateMethod("boolean", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.NewBoolean(cell.Car(args).Bool()))

        return false
    })
    s.PrivateMethod("channel", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, channel(p, nil, nil))

        return false
    })
    s.PrivateMethod("float", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch,
            cell.NewFloat(cell.Car(args).(cell.Atom).Float()))

        return false
    })
    s.PrivateMethod("integer", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch,
            cell.NewInteger(cell.Car(args).(cell.Atom).Int()))

        return false
    })
    s.PrivateMethod("status", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch,
            cell.NewStatus(cell.Car(args).(cell.Atom).Status()))

        return false
    })
    s.PrivateMethod("string", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch,
            cell.NewString(cell.Car(args).String()))

        return false
    })
    s.PrivateMethod("symbol", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch,
            cell.NewSymbol(cell.Raw(cell.Car(args))))

        return false
    })

    /* Relational. */
    s.PrivateMethod("match", func(p *cell.Process, args cell.Cell) bool {
        pattern := cell.Raw(cell.Car(args))
        text := cell.Raw(cell.Cadr(args))

        ok, err := path.Match(pattern, text)
        if err != nil {
            panic(err)
        }

        cell.SetCar(p.Scratch, cell.NewBoolean(ok))

        return false
    })
    s.PrivateMethod("not", func(p *cell.Process, args cell.Cell) bool {
        cell.SetCar(p.Scratch, cell.NewBoolean(!cell.Car(args).Bool()))

        return false
    })
    s.PrivateMethod("eq", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args)

            if !prev.Equal(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("ge", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args).(cell.Atom)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args).(cell.Atom)

            if prev.Less(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("gt", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args).(cell.Atom)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args).(cell.Atom)

            if !prev.Greater(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("is", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args)

            if prev != curr {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("le", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args).(cell.Atom)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args).(cell.Atom)

            if prev.Greater(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("lt", func(p *cell.Process, args cell.Cell) bool {
        prev := cell.Car(args).(cell.Atom)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args).(cell.Atom)

            if !prev.Less(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })
    s.PrivateMethod("ne", func(p *cell.Process, args cell.Cell) bool {
        /*
         * This should really check to make sure no arguments are equal.
         * Currently it only checks whether adjacent pairs are not equal.
         */

        prev := cell.Car(args)

        cell.SetCar(p.Scratch, cell.False)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            curr := cell.Car(args)

            if prev.Equal(curr) {
                return false
            }

            prev = curr
        }

        cell.SetCar(p.Scratch, cell.True)
        return false
    })

    /* Arithmetic. */
    s.PrivateMethod("add", func(p *cell.Process, args cell.Cell) bool {
        acc := cell.Car(args).(cell.Atom)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            acc = acc.Add(cell.Car(args))

        }

        cell.SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("sub", func(p *cell.Process, args cell.Cell) bool {
        acc := cell.Car(args).(cell.Number)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            acc = acc.Subtract(cell.Car(args))
        }

        cell.SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("div", func(p *cell.Process, args cell.Cell) bool {
        acc := cell.Car(args).(cell.Number)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            acc = acc.Divide(cell.Car(args))
        }

        cell.SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("mod", func(p *cell.Process, args cell.Cell) bool {
        acc := cell.Car(args).(cell.Number)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            acc = acc.Modulo(cell.Car(args))
        }

        cell.SetCar(p.Scratch, acc)
        return false
    })
    s.PrivateMethod("mul", func(p *cell.Process, args cell.Cell) bool {
        acc := cell.Car(args).(cell.Atom)

        for cell.Cdr(args) != cell.Null {
            args = cell.Cdr(args)
            acc = acc.Multiply(cell.Car(args))
        }

        cell.SetCar(p.Scratch, acc)
        return false
    })

    e.Add(cell.NewSymbol("$$"), cell.NewInteger(int64(os.Getpid())))

    /* Command-line arguments */
    args := cell.Null
    if len(os.Args) > 1 {
        e.Add(cell.NewSymbol("$0"), cell.NewSymbol(os.Args[1]))

        for i, v := range os.Args[2:] {
            e.Add(cell.NewSymbol("$" + strconv.Itoa(i + 1)), cell.NewSymbol(v))
        }

        for i := len(os.Args) - 1; i > 1; i-- {
            args = cell.Cons(cell.NewSymbol(os.Args[i]), args)
        }
    } else {
        e.Add(cell.NewSymbol("$0"), cell.NewSymbol(os.Args[0]))
    }
    e.Add(cell.NewSymbol("$args"), args)

    /* Environment variables. */
    for _, s := range os.Environ() {
        kv := strings.Split(s, "=", 2)
        e.Add(cell.NewSymbol("$" + kv[0]), cell.NewSymbol(kv[1]))
    }
}

func method(body, param cell.Cell, scope *cell.Scope) *cell.Method {
    return cell.NewMethod(cell.NewClosure(body, param, scope), scope)
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

func next(p *cell.Process) bool {
    body := cell.Car(p.Scratch).(*cell.Method).Func.Body
    
    switch t := body.(type) {
    case cell.Function:
        p.ReplaceState(psExecBuiltin)
        
    case *cell.Integer:
        p.ReplaceState(t.Int())
        return true
        
    default:
        p.ReplaceState(psExecApplication)
    }

    return false
}

func run(p *cell.Process) {
    defer func(saved cell.Process) {
        r := recover()
        if r == nil {
            return
        }

        fmt.Printf("oh: %v\n", r)

        *p = saved

        p.Code = cell.Null
        p.Scratch = cell.Cons(cell.False, p.Scratch)
        p.Stack = cell.Cdr(p.Stack)
    }(*p)

    for p.Stack != cell.Null {
        switch state := p.GetState(); state {
        case psNone:
            return

        case psDoEvalCommand:
            switch cell.Car(p.Scratch).(type) {
            case *cell.String, *cell.Symbol:
                p.ReplaceState(psExecExternal)

            default:
                if next(p) {
                    continue
                }
            }

            p.NewState(psEvalArguments)

            fallthrough
        case psEvalArguments:
            p.Scratch = cell.Cons(nil, p.Scratch)

            p.ReplaceState(psDoEvalArguments)

            fallthrough
        case psDoEvalArguments:
            if p.Code == cell.Null {
                break
            }

            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

            p.Code = cell.Car(p.Code)

            p.NewState(psEvalElement)

            fallthrough
        case psEvalElement:
            if p.Code != cell.Null && cell.IsCons(p.Code) {
                if cell.IsAtom(cell.Cdr(p.Code)) {
                    p.ReplaceState(psEvalAccess)
                } else {
                    p.ReplaceState(psEvalCommand)
                    continue
                }
            } else if sym, ok := p.Code.(*cell.Symbol); ok {
                if c := cell.Resolve(p.Lexical, p.Dynamic, sym); c != nil {
                    p.Scratch = cell.Cons(c.GetValue(), p.Scratch)
                } else {
                    p.Scratch = cell.Cons(sym, p.Scratch)
                }
                break
            } else {
                p.Scratch = cell.Cons(p.Code, p.Scratch)
                break
            }

            fallthrough
        case psEvalAccess:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.NewState(psExecAccess)
            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

            p.Code = cell.Car(p.Code)

            p.NewState(psEvalElement)
            continue

        case psBlock:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Lexical = cell.NewScope(p.Lexical)

            p.NewState(psEvalBlock)

            fallthrough
        case psEvalBlock:
            if !cell.IsCons(p.Code) || !cell.IsCons(cell.Car(p.Code)) {
                break
            }

            if cell.Cdr(p.Code) == cell.Null ||
                !cell.IsCons(cell.Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            fallthrough
        case psEvalCommand:
            p.ReplaceState(psDoEvalCommand)
            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

            p.Code = cell.Car(p.Code)

            p.NewState(psEvalElement)
            continue

        case psEvalFor:
            p.ReplaceState(psExecFor)
            args := p.Arguments()

            /* Second argument to for is a method. First argument is a list. */
            p.Code = cell.Car(args)
            cell.SetCar(p.Scratch, cell.Cadr(args))
            p.Scratch = cell.Cons(cell.Null, p.Scratch)

            fallthrough
        case psExecFor:
            r := cell.Car(p.Scratch)
            p.Scratch = cell.Cdr(p.Scratch)

            if p.Code == cell.Null {
                cell.SetCar(p.Scratch, r)
                break
            }

            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

            p.Scratch = cell.Cons(cell.Car(p.Scratch), p.Scratch)
            p.Scratch = cell.Cons(nil, p.Scratch)
            p.Scratch = cell.Cons(cell.Car(p.Code), p.Scratch)

            p.NewState(psExecApplication)

            fallthrough
        case psExecApplication:
            args := p.Arguments()

            m := cell.Car(p.Scratch).(*cell.Method)
            if m.Self == nil {
                args = expand(args)
            }

            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.Code = m.Func.Body
            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Lexical = cell.NewScope(m.Func.Lexical)

            param := m.Func.Param
            for args != cell.Null && param != cell.Null {
                p.Lexical.Public(cell.Car(param), cell.Car(args))
                args, param = cell.Cdr(args), cell.Cdr(param)
            }
            p.Lexical.Public(cell.NewSymbol("$args"), args)
            p.Lexical.Public(cell.NewSymbol("$self"), m.Self)
            p.Lexical.Public(cell.NewSymbol("return"),
                p.Continuation(psReturn))

            p.NewState(psEvalBlock)
            continue

        case psSet:
            p.ReplaceState(psExecSet)
            p.NewState(psEvalArguments)
            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

            p.Code = cell.Car(p.Code)

            p.NewState(psEvalReference)

            fallthrough
        case psEvalReference:
            p.RemoveState()

            p.Scratch = cell.Cdr(p.Scratch)

            if p.Code != cell.Null && cell.IsCons(p.Code) {
                p.SaveState(cell.SaveLexical)
                p.NewState(psExecReference)
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))

                p.Code = cell.Car(p.Code)
                
                p.NewState(psChangeScope)
                p.NewState(psEvalElement)
                continue
            }

            p.NewState(psExecReference)

            fallthrough
        case psExecReference:
            k := p.Code.(*cell.Symbol)
            v := cell.Resolve(p.Lexical, p.Dynamic, k)
            if v == nil {
                panic("'" + k.String() + "' is not defined")
            }

            p.Scratch = cell.Cons(v, p.Scratch)

        case psDefine, psPublic:
            p.RemoveState()

            l := cell.Car(p.Scratch).(*cell.Method).Self
            if p.Lexical != l {
                p.SaveState(cell.SaveLexical)
                p.Lexical = l
            }

            if state == psDefine {
                p.NewState(psExecDefine)
            } else {
                p.NewState(psExecPublic)
            }

            k := cell.Car(p.Code)

            p.Code = cell.Cadr(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.SaveState(cell.SaveCode | cell.SaveLexical, k)
            p.NewState(psEvalElement)
            continue

        case psExecDefine, psExecPublic:
            if state == psDefine {
                p.Lexical.Private(p.Code, cell.Car(p.Scratch))
            } else {
                p.Lexical.Public(p.Code, cell.Car(p.Scratch))
            }

        case psDynamic, psSetenv:
            k := cell.Car(p.Code)

            if state == psSetenv {
                if !strings.HasPrefix(k.String(), "$") {
                    break
                }

                p.ReplaceState(psExecSetenv)
            } else {
                p.ReplaceState(psExecDynamic)
            }

            p.Code = cell.Cadr(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.SaveState(cell.SaveCode | cell.SaveDynamic, k)
            p.NewState(psEvalElement)
            continue

        case psExecDynamic, psExecSetenv:
            k := p.Code
            v := cell.Car(p.Scratch)

            if state == psExecSetenv {
                s := cell.Raw(v)
                os.Setenv(strings.TrimLeft(k.String(), "$"), s)
            }

            p.Dynamic.Add(k, v)

        case psWhile:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.NewState(psEvalWhileTest)

            fallthrough
        case psEvalWhileTest:
            p.ReplaceState(psEvalWhileBody)
            p.SaveState(cell.SaveCode, p.Code)

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psEvalWhileBody:
            if !cell.Car(p.Scratch).Bool() {
                break
            }

            p.ReplaceState(psEvalWhileTest)
            p.SaveState(cell.SaveCode, p.Code)

            p.Code = cell.Cdr(p.Code)

            p.NewState(psEvalBlock)
            continue

        case psAnd:
            cell.SetCar(p.Scratch, cell.True)
            p.ReplaceState(psEvalAnd)

            fallthrough
        case psEvalAnd:
            prev := cell.Car(p.Scratch).Bool()
            cell.SetCar(p.Scratch, cell.NewBoolean(prev))

            if p.Code == cell.Null || !prev {
                break
            }

            if cell.Cdr(p.Code) == cell.Null {
                p.ReplaceState(psEvalElement)
            } else {
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
                p.NewState(psEvalElement)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)
            continue

        case psAndf:
            p.Scratch = cell.Cons(cell.True, p.Scratch)
            p.ReplaceState(psEvalAndf)

            fallthrough
        case psEvalAndf:
            if !cell.IsCons(p.Code) || !cell.IsCons(cell.Car(p.Code)) {
                break
            }

            if !cell.Car(p.Scratch).Bool() {
                break
            }

            if cell.Cdr(p.Code) == cell.Null ||
                !cell.IsCons(cell.Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)
            continue

        case psOr:
            cell.SetCar(p.Scratch, cell.False)
            p.ReplaceState(psEvalOr)

            fallthrough
        case psEvalOr:
            prev := cell.Car(p.Scratch).Bool()
            cell.SetCar(p.Scratch, cell.NewBoolean(prev))

            if p.Code == cell.Null || prev {
                break
            }

            if cell.Cdr(p.Code) == cell.Null {
                p.ReplaceState(psEvalElement)
            } else {
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
                p.NewState(psEvalElement)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)
            continue

        case psOrf:
            p.Scratch = cell.Cons(cell.False, p.Scratch)
            p.ReplaceState(psEvalOrf)

            fallthrough
        case psEvalOrf:
            if !cell.IsCons(p.Code) || !cell.IsCons(cell.Car(p.Code)) {
                break
            }

            if cell.Car(p.Scratch).Bool() {
                break
            }

            if cell.Cdr(p.Code) == cell.Null ||
                !cell.IsCons(cell.Cadr(p.Code)) {
                p.ReplaceState(psEvalCommand)
            } else {
                p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
                p.NewState(psEvalCommand)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)
            continue

        case psChangeScope:
            p.Lexical = cell.Car(p.Scratch).(cell.Interface)
            p.Scratch = cell.Cdr(p.Scratch)

        case psExecAccess:
            p.Dynamic = nil
            p.Lexical = cell.Car(p.Scratch).(cell.Interface)
            p.Scratch = cell.Cdr(p.Scratch)
            p.ReplaceState(psEvalElement)
            continue

        case psExecBuiltin:
            args := p.Arguments()

            m := cell.Car(p.Scratch).(*cell.Method)
            if m.Self == nil {
                args = expand(args)
            }

            if m.Func.Body.(cell.Function)(p, args) {
                continue
            }

        case psExecExternal:
            args := p.Arguments()

            if external(p, args) {
                continue
            }

        case psExecIf:
            if !cell.Car(p.Scratch).Bool() {
                p.Code = cell.Cdr(p.Code)

                for cell.Car(p.Code) != cell.Null &&
                    !cell.IsAtom(cell.Car(p.Code)) {
                    p.Code = cell.Cdr(p.Code)
                }

                p.Code = cell.Cdr(p.Code)
            }

            if p.Code == cell.Null {
                break
            }

            p.ReplaceState(psEvalBlock)
            continue

        case psExecImport:
            n := cell.Raw(cell.Car(p.Scratch))

            k, err := module(n)
            if err != nil {
                cell.SetCar(p.Scratch, cell.False)
                break
            }

            v := cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol(k))
            if v != nil {
                cell.SetCar(p.Scratch, v.GetValue())
                break
            }

            p.ReplaceState(psCreateModule)
            p.SaveState(cell.SaveCode, cell.NewSymbol(n))
            p.NewState(psExecSource)

            fallthrough
        case psExecSource:
            f, err := os.OpenFile(
                cell.Raw(cell.Car(p.Scratch)),
                os.O_RDONLY, 0666)
            if err != nil {
                panic(err)
            }

            p.Code = cell.Null
            cell.ParseFile(f, func (c cell.Cell) {
                p.Code = cell.AppendTo(p.Code, c)
            })

            if state == psExecImport {
                p.RemoveState()
                p.SaveState(cell.SaveDynamic | cell.SaveLexical)

                p.Dynamic = cell.NewEnv(p.Dynamic)
                p.Lexical = cell.NewScope(p.Lexical)

                p.NewState(psExecObject)
                p.NewState(psEvalBlock)
            } else {
                if p.Code == cell.Null {
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
            p.Lexical.Private(cell.NewSymbol(k), cell.Car(p.Scratch))

        case psExecObject:
            cell.SetCar(p.Scratch, cell.NewObject(p.Lexical))

        case psExecSet:
            args := p.Arguments()

            r := cell.Car(p.Scratch).(*cell.Reference)

            r.SetValue(cell.Car(args))
            cell.SetCar(p.Scratch, r.GetValue())

        case psExecSplice:
            l := cell.Car(p.Scratch)
            p.Scratch = cell.Cdr(p.Scratch)

            if !cell.IsCons(l) {
                break
            }

            for l != cell.Null {
                p.Scratch = cell.Cons(cell.Car(l), p.Scratch)
                l = cell.Cdr(l)
            }

            /* Command states */
        case psBackground:
            child := cell.NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psEvalCommand)

            child.Code = cell.Car(p.Code)
            cell.SetCar(p.Scratch, cell.True)

            go run(child)

        case psBacktick:
            c := channel(p, nil, nil)

            child := cell.NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psPipeChild)

            s := cell.NewSymbol("$stdout")
            child.SaveState(cell.SaveCode, s)

            child.Code = cell.Car(p.Code)
            child.Dynamic.Add(s, c)

            child.NewState(psEvalCommand)

            go run(child)

            g := cell.Resolve(c, nil, cell.NewSymbol("guts"))
            b := bufio.NewReader(g.GetValue().(*cell.Channel).ReadEnd())

            l := cell.Null

            done := false
            line, err := b.ReadString('\n')
            for !done {
                if err != nil {
                    done = true
                }

                line = strings.Trim(line, " \t\n")

                if len(line) > 0 {
                    l = cell.AppendTo(l, cell.NewString(line))
                }

                line, err = b.ReadString('\n')
            }

            cell.SetCar(p.Scratch, l)

        case psBuiltin, psMethod:
            param := cell.Null
            for !cell.IsCons(cell.Car(p.Code)) {
                param = cell.Cons(cell.Car(p.Code), param)
                p.Code = cell.Cdr(p.Code)
            }

            if state == psBuiltin {
                cell.SetCar(
                    p.Scratch,
                    function(p.Code, cell.Reverse(param), p.Lexical.Expose()))
            } else {
                cell.SetCar(
                    p.Scratch,
                    method(p.Code, cell.Reverse(param), p.Lexical.Expose()))
            }

        case psFor:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.NewState(psEvalFor)
            p.NewState(psEvalArguments)
            continue

        case psIf:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Lexical = cell.NewScope(p.Lexical)

            p.NewState(psExecIf)
            p.SaveState(cell.SaveCode, cell.Cdr(p.Code))
            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psImport, psSource:
            if state == psImport {
                p.ReplaceState(psExecImport)
            } else {
                p.ReplaceState(psExecSource)
            }

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psObject:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic | cell.SaveLexical)

            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Lexical = cell.NewScope(p.Lexical)

            p.NewState(psExecObject)
            p.NewState(psEvalBlock)
            continue

        case psQuote:
            cell.SetCar(p.Scratch, cell.Car(p.Code))

        case psReturn:
            p.Code = cell.Car(p.Code)

            m := cell.Car(p.Scratch).(*cell.Method)
            p.Scratch = cell.Car(m.Func.Param)
            p.Stack = cell.Cadr(m.Func.Param)

            p.NewState(psEvalElement)
            continue

        case psSpawn:
            child := cell.NewProcess(psNone, p.Dynamic, p.Lexical)

            child.Scratch = cell.Cons(cell.Null, child.Scratch)
            child.NewState(psEvalBlock)

            child.Code = p.Code

            go run(child)

        case psSplice:
            p.ReplaceState(psExecSplice)

            p.Code = cell.Car(p.Code)
            p.Scratch = cell.Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psPipeStderr, psPipeStdout:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic)

            c := channel(p, nil, nil)

            child := cell.NewProcess(psNone, p.Dynamic, p.Lexical)

            child.NewState(psPipeChild)

            var s *cell.Symbol
            if state == psPipeStderr {
                s = cell.NewSymbol("$stderr")
            } else {
                s = cell.NewSymbol("$stdout")
            }
            child.SaveState(cell.SaveCode, s)

            child.Code = cell.Car(p.Code)
            child.Dynamic.Add(s, c)

            child.NewState(psEvalCommand)

            go run(child)

            p.Code = cell.Cadr(p.Code)
            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Scratch = cell.Cdr(p.Scratch)

            p.Dynamic.Add(cell.NewSymbol("$stdin"), c)

            p.NewState(psPipeParent)
            p.NewState(psEvalCommand)
            continue
            
        case psPipeChild:
            c := cell.Resolve(p.Lexical, p.Dynamic, p.Code.(*cell.Symbol))
            c = cell.Resolve(
                c.GetValue().(cell.Interface).Expose(), nil,
                cell.NewSymbol("guts"))
            c.GetValue().(*cell.Channel).WriteEnd().Close()
            
        case psPipeParent:
            c := cell.Resolve(p.Lexical, p.Dynamic, cell.NewSymbol("$stdin"))
            c = cell.Resolve(
                c.GetValue().(cell.Interface).Expose(), nil,
                cell.NewSymbol("guts"))
            c.GetValue().(*cell.Channel).Close()

        case psAppendStderr, psAppendStdout, psRedirectStderr,
            psRedirectStdin, psRedirectStdout:
            p.RemoveState()
            p.SaveState(cell.SaveDynamic)

            initial := cell.NewInteger(state)

            p.NewState(psRedirectCleanup)
            p.NewState(psEvalCommand)
            p.SaveState(cell.SaveCode, cell.Cadr(p.Code))
            p.NewState(psRedirectSetup)
            p.SaveState(cell.SaveCode, initial)

            p.Code = cell.Car(p.Code)
            p.Dynamic = cell.NewEnv(p.Dynamic)
            p.Scratch = cell.Cdr(p.Scratch)

            p.NewState(psEvalElement)
            continue

        case psRedirectSetup:
            flags, name := 0, ""
            initial := p.Code.(cell.Atom).Int()
            
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

            c, ok := cell.Car(p.Scratch).(cell.Interface)
            if !ok {
                n := cell.Raw(cell.Car(p.Scratch))
                
                f, err := os.OpenFile(n, flags, 0666)
                if err != nil {
                    panic(err)
                }

                if name == "$stdin" {
                    c = channel(p, f, nil)
                } else {
                    c = channel(p, nil, f)
                }
                cell.SetCar(p.Scratch, c)

                r := cell.Resolve(c, nil, cell.NewSymbol("guts"))
                ch := r.GetValue().(*cell.Channel)

                ch.Implicit = true
            }

            p.Dynamic.Add(cell.NewSymbol(name), c)

        case psRedirectCleanup:
            c := cell.Cadr(p.Scratch).(cell.Interface)
            r := cell.Resolve(c, nil, cell.NewSymbol("guts"))
            ch := r.GetValue().(*cell.Channel)

            if ch.Implicit {
                ch.Close()
            }

            cell.SetCdr(p.Scratch, cell.Cddr(p.Scratch))

        default:
            if state >= cell.SaveMax {
                panic(fmt.Sprintf("command not found: %s", p.Code))
            } else {
                p.RestoreState()
            }
        }

        p.RemoveState()
    }
}

func Evaluate(c cell.Cell) {
    main.NewState(psEvalCommand)
    main.Code = c
    
    run(main)

    if main.Stack == cell.Null {
        os.Exit(Status())
    }

    main.Scratch = cell.Cdr(main.Scratch)
}

func Start() {

    cell.Parse(bufio.NewReader(strings.NewReader(`
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
        main.NewState(psEvalCommand)
        main.Code = cell.List(cell.NewSymbol("source"), cell.NewSymbol(rc))
        
        run(main)
        
        main.Scratch = cell.Cdr(main.Scratch)
    }
}

func Status() int {
    s, ok := cell.Car(main.Scratch).(*cell.Status)
    if !ok {
        return 0
    }
    return int(s.Int())
}
