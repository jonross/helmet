/*
    Copyright (c) 2012, 2013 by Jonathan Ross (jonross@alum.mit.edu)

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
    "os"
)

type SearchSuite struct{} 
var _ = Suite(&SearchSuite{})

var testHeap *Heap

func (s *SearchSuite) TestSearch(c *C) {

    fmt.Print("") // gratuitous use

    heap := getHeap(c)
    parsers := NewParsers()

    // verify Thing counts
    histo := NewHisto(heap, nil)
    _, _, result := parsers.Command.Parse("histo x, x of com.myco.Thing1 x")
    SearchHeap(heap, result.(SearchAction).Query, histo)
    count, _ := histo.Counts(heap.ClassNamed("com.myco.Thing1"))
    c.Check(count, Equals, uint32(10000))

    // manually construct "x group y of Object x -> Integer y"
    query := &Query {
        []*Step {
            &Step{"Object", "x", true, false},
            &Step{"Integer", "y", true, false},
        },
        []int{0, 1},
    }
    histo = NewHisto(heap, nil)
    SearchHeap(heap, query, histo)
    histo.Print(os.Stdout)
}

func getHeap(c *C) *Heap {
    if testHeap != nil {
        return testHeap
    }
    heapFile := "./genheap.hprof"
    if _, err := os.Stat(heapFile); os.IsNotExist(err) {
        c.Fatal("Can't find %s, make sure it's been generated\n", heapFile)
    }
    options := &Options{NeedRefs: true}
    testHeap := ReadHeapDump(heapFile, options)
    return testHeap
}
