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
    "bytes"
    "fmt"
    "io"
    "sort"
)

// Report on the # of instances of each class and the total byte count per class,
// a la 'jmap -histo'
//
type Histo struct {
    // counts indexed by class ID
    counts []*ClassCount
    // indicates what objects we've seen
    known BitSet
}

type ClassCount struct {
    name []byte
    count uint32
    nbytes uint64
}

// Support sort weirdness. :-(

type classCounts []*ClassCount
func (cc classCounts) Len() int { return len(cc) }
func (cc classCounts) Swap(i, j int) { cc[i], cc[j] = cc[j], cc[i] }

func (cc classCounts) Less(i, j int) bool { 
    if cc[i].nbytes > cc[j].nbytes {
        return true
    }
    if cc[i].nbytes < cc[j].nbytes {
        return false
    }
    return bytes.Compare(cc[i].name, cc[j].name) < 0
}

// Create a Histo with enough room for indicated # of classes & objects.
//
func NewHisto(numClasses, numObjects uint32) *Histo {
    return &Histo{
        counts: make([]*ClassCount, numClasses + 1), // 1-based
        known: NewBitSet(numObjects + 1), // 1-based
    }
}

// Add an object if not already known.
//
func (h *Histo) Add(oid ObjectId, class *ClassDef, size uint32) {
    id := uint32(oid)
    if h.known.has(id) {
        return
    }
    h.known.set(id)
    slot := h.counts[class.Cid]
    if slot == nil {
        slot = &ClassCount{name: []byte(class.Name)}
        h.counts[class.Cid] = slot
    }
    slot.count++
    slot.nbytes += uint64(size)
}

// Print the histogram.
//

func (h *Histo) Print(out io.Writer) {

    // Some classes have no instances
    counts := []*ClassCount{}
    for _, slot := range h.counts {
        if slot != nil {
            counts = append(counts, slot)
        }
    }
    sort.Sort(classCounts(counts))

    totalCount := uint32(0)
    totalBytes := uint64(0)

    for _, slot := range counts {
        fmt.Fprintf(out, "%10d %10d %s\n", slot.count, slot.nbytes, string(slot.name))
        totalCount += slot.count
        totalBytes += slot.nbytes
    }

    fmt.Fprintf(out, "%10d %10d total\n", totalCount, totalBytes)
}
