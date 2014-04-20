/*
    Copyright (c) 2013, 2014 by Jonathan Ross (jonross@alum.mit.edu)

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
    "sync"
)

// For accumulating a list of references from an instance or array dump.  We know 
// the object ID of the source but not that of the target, because we don't 
// build the HID->OID mapping until after all the object IDs are known.
//
type References struct {
    from []Oid
    to []Hid
}

func (refs *References) Add(from Oid, to Hid) {
    refs.from = append(refs.from, from)
    refs.to = append(refs.to, to)
}

// Combine and resolve a list of Referencess into separate referrer / referee arrays,
// using a resolution function to turn referee heap IDs into object IDs.  The bags
// should be discarded afterward to save memory.
//
func MergeReferences(arefs []*References, resolver func(Hid) Oid) ([]Oid, []Oid){

    count := 0
    for _, refs := range arefs {
        count += len(refs.from)
    }

    var wg sync.WaitGroup
    newFrom := make([]Oid, count)
    newTo := make([]Oid, count)
    offset := 0

    for _, refs := range arefs {
        wg.Add(1)
        go func(from []Oid, to []Hid, base int) {
            for i, oid := range from {
                newFrom[base+i] = oid
                newTo[base+i] = resolver(to[i])
            }
            wg.Done()
        }(refs.from, refs.to, offset)
        offset += len(refs.from)
    }

    wg.Wait()
    return newFrom, newTo // TODO: include # of unmappable references
}


