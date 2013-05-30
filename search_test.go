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
    "os"
    "testing"
)

var testHeap *Heap

func TestSearch(t *testing.T) {
    LogTestOutput()
    heap := getHeap(t)
    // manually construct "x group y from Object x -> Integer y"
    query := Query([]*Step {
        &Step{"Object", "x", true, false, StepGroup},
        &Step{"Integer", "y", true, false, StepMember},
    })
    histo := NewHisto(heap)
    SearchHeap(heap, query, histo)
    histo.Print(os.Stdout)
    t.FailNow()
}

func getHeap(t *testing.T) *Heap {
    if testHeap != nil {
        return testHeap
    }
    heapFile := "./genheap.hprof"
    if _, err := os.Stat(heapFile); os.IsNotExist(err) {
        t.Fatalf("Can't find %s, make sure it's been generated\n", heapFile)
    }
    options := &Options{NeedRefs: true}
    testHeap := ReadHeapDump(heapFile, options)
    return testHeap
}
