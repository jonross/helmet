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
    "log"
    "sync"
)

// For accumulating a list of references from an instance or array dump.  We know 
// the object ID of the source but not that of the target, because we don't 
// build the HID->OID mapping until after all the object IDs are known.
//
type RefBag struct {
    from [][]ObjectId
    to [][]HeapId
    count int
}

// Add a reference.
//
func (refs *RefBag) Add(from ObjectId, to HeapId) {
    if refs.count == 0 {
        refs.from = [][]ObjectId{make([]ObjectId, 0, 100000)}
        refs.to = [][]HeapId{make([]HeapId, 0, 100000)}
    }
    refs.from = xappendOid(refs.from, from)
    refs.to = xappendHid(refs.to, to)
    refs.count++
}

// Combine and resolve a list of RefBags into separate referrer / referee arrays,
// using a resolution function to turn referee heap IDs into object IDs.  The bags
// should be discarded afterward to save memory.
//
func MergeBags(bags []*RefBag, resolver func(HeapId) ObjectId) {

    count := 0
    for _, bag := range bags {
        count += bag.count
    }
    log.Printf("Resolving %d references\n", count)

    var wg sync.WaitGroup
    newFrom := make([]ObjectId, count)
    newTo := make([]ObjectId, count)
    offset := 0

    for _, bag := range bags {
        wg.Add(len(bag.from))
        for i, _ := range bag.from {
            go func(from []ObjectId, to []HeapId, offset int) {
                for j, oid := range from {
                    newFrom[offset+j] = oid
                    newTo[offset+j] = resolver(to[j])
                }
                wg.Done()
            }(bag.from[i], bag.to[i], offset)
            offset += len(bag.from[i])
        }
    }

    wg.Wait()
}


