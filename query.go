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
    . "github.com/jonross/peggy"
)

type QStep struct {
    className string
    varName string
    outbound bool
    skip bool
}

func makeParser() *Parser {

    letter := AnyOf("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_")
    number := AnyOf("0123456789")
    identifier := Sequence(letter, ZeroOrMoreOf(OneOf(letter, number))).Adjacent().As(String)

    // Match e.g. Integer, com.myco.*, long[][]
    className := Sequence(identifier, ZeroOrMoreOf(Sequence(".", identifier)), 
                          Optional(OneOf(".*", OneOrMoreOf("[]")))).Adjacent().As(String)

    // Match classname followed by optional step var name, and generate a QStep
    step := Sequence(className, Optional(identifier)).
        Handle(func (s *State) interface{} {
            cname := s.Get(1).String()
            vname := ""
            if ! s.Get(2).IsNil() {
                vname = s.Get(2).String()
            }
            return &QStep{cname, vname, true, false}
        })

    // Modify outbound / skip settings of a chain of QSteps
    arrow := OneOf("<<-", "<-", "->>", "->")
    query := Sequence(step, ZeroOrMoreOf(Sequence(arrow, step))).Flatten(1).
        Handle(func (s *State) interface{} {
            steps := []*QStep{s.Get(1).Interface().(*QStep)}
            for i := 2; i <= s.Len(); i++ {
                arrow := s.Get(i).String()
                step := s.Get(i+1).Interface().(*QStep)
                switch arrow {
                    case "<<-": 
                        step.skip = true
                        step.outbound = false
                    case "<-":
                        step.skip = false
                        step.outbound = false
                    case "->":
                        step.skip = false
                        step.outbound = true
                    case "->>":
                        step.skip = true
                        step.outbound = true
                }
            }
            return steps
        })

    funargs := Sequence(identifier, ZeroOrMoreOf(Sequence(",", identifier).Pick(2))).Flatten(1).As(Strings)
    funcall := Sequence(identifier, "(", funargs, ")")
    search := Sequence(funcall, "from", query)

    return search
}

