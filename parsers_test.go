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
    . "launchpad.net/gocheck"
    "log"
    "testing"
)

// Hook up gocheck into the "go test" runner. 
func Test(t *testing.T) { TestingT(t) }
type QuerySuite struct{} 
var _ = Suite(&QuerySuite{})

// Some fine-grained query tests of lower-level parser nodes.  These let us detect
// breakage there with less debugging.
//
func (s *QuerySuite) TestQueries(c *C) {

    LogTestOutput()
    parsers := NewParsers()

    _, _, result := parsers.ClassName.Parse("Object")
    c.Check(result, Equals, "Object")

    _, _, result = parsers.ClassName.Parse("java.lang.Object")
    c.Check(result, Equals, "java.lang.Object")

    _, _, result = parsers.ClassName.Parse("int[][]")
    c.Check(result, Equals, "int[][]")

    _, _, result = parsers.Step.Parse("Object")
    c.Check(result, DeepEquals, &QStep{"Object", "", true, false})

    _, _, result = parsers.Step.Parse("Object x")
    c.Check(result, DeepEquals, &QStep{"Object", "x", true, false})

    _, _, result = parsers.Path.Parse("Map y ->> Integer x")
    c.Check(result, DeepEquals, []*QStep {
        &QStep{"Map", "y", true, false},
        &QStep{"Integer", "x", true, true},
    })

    _, _, result = parsers.Path.Parse("Integer x <<- Map y")
    c.Check(result, DeepEquals, []*QStep {
        &QStep{"Integer", "x", true, false},
        &QStep{"Map", "y", false, true},
    })

    log.Print("")
}

// Verify all the "set" actions.
//
func (s *QuerySuite) TestSettings(c *C) {

    session := &Session{Settings: DefaultSettings()}
    sval := func(name string) *Setting {
        return session.Settings[name]
    }

    c.Check(1 << 20, Equals, sval("mingroupsize").IntValue)

    session.run("set mingroupsize 100k")
    c.Check(100 * (1 << 10), Equals, sval("mingroupsize").IntValue)

    session.run("set mingroupsize 5m")
    c.Check(5 * (1 << 20), Equals, sval("mingroupsize").IntValue)

    session.run("set mingroupsize 1g")
    c.Check(1 << 30, Equals, sval("mingroupsize").IntValue)
}

