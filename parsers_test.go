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
    "fmt"
    "log"
)

type ParserSuite struct{} 
var _ = Suite(&ParserSuite{})

// Some fine-grained query tests of lower-level parser nodes.  These let us detect
// breakage there with less debugging.
//
func (s *ParserSuite) TestQueries(c *C) {

    parsers := NewParsers()

    _, _, result := parsers.ClassName.Parse("Object")
    c.Check(result, Equals, "Object")

    _, _, result = parsers.ClassName.Parse("java.lang.Object")
    c.Check(result, Equals, "java.lang.Object")

    _, _, result = parsers.ClassName.Parse("int[][]")
    c.Check(result, Equals, "int[][]")

    _, _, result = parsers.Step.Parse("Object")
    c.Check(result, DeepEquals, &Step{"Object", "", true, false})

    _, _, result = parsers.Step.Parse("Object x")
    c.Check(result, DeepEquals, &Step{"Object", "x", true, false})

    _, _, result = parsers.Path.Parse("Map y ->> Integer x")
    c.Check(result, DeepEquals, []*Step {
        &Step{"Map", "y", true, false},
        &Step{"Integer", "x", true, true},
    })

    _, _, result = parsers.Path.Parse("Integer x <<- Map y")
    c.Check(result, DeepEquals, []*Step {
        &Step{"Integer", "x", true, false},
        &Step{"Map", "y", false, true},
    })

    _, _, result = parsers.Command.Parse("histo x, y of Map x -> Integer y")
    c.Check(result, DeepEquals, SearchAction{
        &Query {
            []*Step {
                &Step{"Map", "x", true, false},
                &Step{"Integer", "y", true, false},
            },
            []int{0, 1},
        },
    })

    _, _, result = parsers.Command.Parse("histo x, y of Map x -> Integer z")
    c.Check(result, DeepEquals, ErrorAction{
        fmt.Errorf("Function variable y is not defined in path"),
    })

    log.Print("")
}

// Verify all the "set" actions.
//
func (s *ParserSuite) TestSettings(c *C) {

    session := NewSession(nil)

    c.Check(session.Threshold, DeepEquals, Setting{"threshold", "", 1024, "bytes"})

    session.run("set threshold 100k bytes")
    c.Check(session.Threshold, DeepEquals, Setting{"threshold", "", int64(100 * (1 << 10)), "bytes"})

    session.run("set threshold 5m objects")
    c.Check(session.Threshold, DeepEquals, Setting{"threshold", "", int64(5 * (1 << 20)), "objects"})

    session.run("set threshold 1g retained")
    c.Check(session.Threshold, DeepEquals, Setting{"threshold", "", int64(1 << 30), "retained"})

    session.run("set nothreshold")
    c.Check(session.Threshold, DeepEquals, Setting{"threshold", "", int64(0), ""})

    c.Check(session.Garbage, DeepEquals, Setting{"garbage", "", 0, "live"})

    session.run("set garbage")
    c.Check(session.Garbage, DeepEquals, Setting{"garbage", "", 0, "all"})

    session.run("set garbage only")
    c.Check(session.Garbage, DeepEquals, Setting{"garbage", "", 0, "nonlive"})

    session.run("set nogarbage")
    c.Check(session.Garbage, DeepEquals, Setting{"garbage", "", 0, "live"})
}

