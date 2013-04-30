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
    "flag"
    "log"
    "os"
    "runtime"
    "runtime/pprof"
)

// Processing options.
//
type Options struct {
    // do we need the reference graph
    NeedRefs bool
}

func main() {

    runtime.GOMAXPROCS(runtime.NumCPU())
    // runtime.GOMAXPROCS(1)

    cpuProfile := flag.String("cpuprofile", "", "write cpu profile to file")
    doHisto := flag.Bool("histo", false, "generate class histogram & exit")
    flag.Parse()
    args := flag.Args()

    if *cpuProfile != "" {
        f, err := os.Create(*cpuProfile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }

    switch {
        case len(args) == 0:
            log.Fatal("Missing heap filename")
        case len(args) > 1:
            log.Fatal("Extra args following heap filename")
    }

    options := &Options{
        NeedRefs: ! *doHisto,
    }

    heap := ReadHeapDump(flag.Arg(0), options)

    if *doHisto {
        histo := NewHisto(heap.MaxClassId, uint32(heap.MaxObjectId))
        for oid := ObjectId(1); oid <= heap.MaxObjectId; oid++ {
            class := heap.ClassOf(oid)
            histo.Add(oid, class, heap.SizeOf(oid))
        }
        histo.Print(os.Stdout)
    }
}

