/*
    Copyright (c) 2013 by Jonathan Ross (jonross@alum.mit.edu)

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in
    all copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
*/

package main

import (
    "fmt"
    // "log"
    . "github.com/jonross/peggy"
    "reflect"
)

// Represents a function call in a query.  This is a temporary artifact of the
// parsing code; see validateSearch for creating full Query object.
//
type QFun struct {
    fnName string
    fnArgs []string
}

// Several nodes from the PEG grammar are returned by NewParsers so it's easy
// to test each individually.
//
type Parsers struct {
    ClassName *Parser
    Step *Parser
    Path *Parser
    Search *Parser
    Setting *Parser
    Command *Parser
}

// Generate interactive command parser.  The "Command" field is used for processing
// an interactive session; the others are intermediate parsers for testing only.
//
func NewParsers() *Parsers {

    letter := AnyOf("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_$")
    digit := AnyOf("0123456789")
    identifier := Sequence(letter, ZeroOrMoreOf(OneOf(letter, digit))).Adjacent().As(String)

    // Match e.g. Integer, com.myco.*, long[][]
    className := Sequence(identifier, ZeroOrMoreOf(Sequence(".", identifier)), 
                          Optional(OneOf(".*", OneOrMoreOf("[]")))).Adjacent().As(String)

    // Match classname followed by optional step var name, and generate a Step
    step := Sequence(className, Optional(identifier)).
        Handle(func (s *State) interface{} {
            cname := s.Get(1).String()
            vname := ""
            if s.Get(2).Kind() == reflect.String {
                vname = s.Get(2).String()
            }
            return &Step{cname, vname, true, false}
        })

    // Modify outbound / skip settings of a chain of Steps
    arrow := OneOf("<<-", "<-", "->>", "->")
    path := Sequence(step, ZeroOrMoreOf(Sequence(arrow, step))).Flatten(2).
        Handle(func (s *State) interface{} {
            steps := []*Step{s.Get(1).Interface().(*Step)}
            for i := 2; i <= s.Len(); i += 2 {
                arrow := s.Get(i).String()
                step := s.Get(i+1).Interface().(*Step)
                switch arrow {
                    case "<<-": 
                        step.to = false
                        step.skip = true
                    case "<-":
                        step.to = false
                        step.skip = false
                    case "->":
                        step.to = true
                        step.skip = false
                    case "->>":
                        step.to = true
                        step.skip = true
                }
                steps = append(steps, step)
            }
            return steps
        })

    // Match e.g. "histo x, y"
    funargs := Sequence(identifier, ZeroOrMoreOf(Sequence(",", identifier).Pick(2))).Flatten(1).As(Strings)
    funcall := Sequence(identifier, funargs).
        Handle(func (s *State) interface{} {
            fnName := s.Get(1).String()
            fnArgs := s.Get(2).Interface().([]string)
            return &QFun{fnName, fnArgs}
        })

    search := Sequence(funcall, "of", path).
        Handle(func (s *State) interface{} {
            function := s.Get(1).Interface().(*QFun)
            path := s.Get(3).Interface().([]*Step)
            query, err := validateSearch(function, path)
            if err != nil {
                return ErrorAction{err}
            } else {
                return SearchAction{query}
            }
        })

    setting := newSettingsParser()

    command := OneOf(search, setting)

    return &Parsers{
        ClassName: className,
        Step: step,
        Path: path,
        Search: search,
        Setting: setting,
        Command: command,
    }
}

// Create sub-parser for "set" commands.
//
func newSettingsParser() *Parser {

    number := OneOrMoreOf(AnyOf("0123456789")).Adjacent().As(Int)
    size := Sequence(number, OneOf("k", "m", "g")).
        Handle(func (s *State) interface{} {
            value := s.Get(1).Int()
            switch s.Get(2).String() {
                case "k": value *= 1 << 10
                case "m": value *= 1 << 20
                case "g": value *= 1 << 30
            }
            return value
        })

    setGarbage := Sequence("set", OneOf("garbage", "nogarbage"), Optional("only")).
        Handle(func(s *State) interface{} {
            var value int64
            if s.Get(2).String() == "nogarbage" {
                value = 0
            } else if ! s.Get(3).IsValid() {
                value = 1
            } else {
                value = 2
            }
            return Setting{Name: "garbage", Number: value}
        })

    setThreshold := Sequence("set", "threshold", size, OneOf("objects", "bytes", "retained")).
        Handle(func(s *State) interface{} {
            sname := s.Get(2).String()
            sval := int64(s.Get(3).Int())
            // log.Printf("Got %s = %d\n", sname, sval)
            return Setting{Name: sname, Number: sval}
        })

    setNoThreshold := Sequence("set", "nothreshold").
        Handle(func(s *State) interface{} {
            return Setting{Name: "threshold", Number: 0}
        })

    return OneOf(setThreshold, setNoThreshold, setGarbage)
}

// Validate search parameters; ensure all function params are defined
// in the path, and return a fully composed Query.
//
func validateSearch(fn *QFun, steps []*Step) (*Query, error) {
    query := &Query {
        steps,
        make([]int, len(fn.fnArgs)),
    }
    for i, arg := range fn.fnArgs {
        found := false
        for j, step := range steps {
            if arg == step.varName {
                query.argIndices[i] = j
                found = true
            }
        }
        if ! found {
            return nil, fmt.Errorf("Function variable %s is not defined in path", arg)
        }
    }
    return query, nil
}

