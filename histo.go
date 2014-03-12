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
    heap *Heap
    // counts indexed by class ID
    counts []*ClassCount
    // indicates what objects we've seen
    known BitSet
}

type ClassCount struct {
    name []byte
    count uint32
    nbytes uint64
    retained uint64 // TODO calculate
}

// Support sort weirdness. :-(
//
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

// Add an object if not already known.
//
func (h *Histo) Add(oid ObjectId, class *ClassDef, size uint32) {
    id := uint32(oid)
    if h.known.Has(id) {
        return
    }
    h.known.Set(id)
    slot := h.counts[class.Cid]
    if slot == nil {
        slot = &ClassCount{name: []byte(class.Name)}
        h.counts[class.Cid] = slot
    }
    slot.count++
    slot.nbytes += uint64(size)
}

// Return count, nbytes for a class.
//
func (h *Histo) Counts(class *ClassDef) (uint32, uint64) {
    slot := h.counts[class.Cid]
    if slot != nil {
        return slot.count, slot.nbytes
    } else {
        return 0, 0
    }
}

// Implement Collector.Collect
//
func (h *Histo) Collect(oids []ObjectId) {
    group := oids[0]
    member := oids[1]
    // TODO check known first
    h.Add(member, h.heap.ClassOf(group), h.heap.SizeOf(member))
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

/*
    TODO add

    abstract sealed trait Threshold
    case object NoLimit extends Threshold
    case class MaxCount(count: Int) extends Threshold
    case class MaxBytes(nbytes: Long) extends Threshold
    case class MaxRetained(nbytes: Long) extends Threshold
    
    abstract sealed trait GarbageVisibility
    case object AllObjects extends GarbageVisibility
    case object LiveOnly extends GarbageVisibility
    case object GarbageOnly extends GarbageVisibility

        val diff = heap.threshold match {
            case NoLimit =>             (a: Counts, b: Counts) => a.nbytes - b.nbytes
            case MaxCount(count) =>     (a: Counts, b: Counts) => a.count.toLong - b.count.toLong
            case MaxBytes(nbytes) =>    (a: Counts, b: Counts) => a.nbytes - b.nbytes
            case MaxRetained(nbytes) => (a: Counts, b: Counts) => a.retained - b.retained
        }
        
        val slots = counts.toList filter {_ != null} sortWith { (a,b) =>
            val delta = diff(a, b)
            if (delta > 0) true
            else if (delta < 0) false
            else a.classDef.name.compareTo(b.classDef.name) < 0
        }
        
        val total = new Counts(null)
        val hidden = new Counts(null)
        
        val hide = heap.threshold match {
            case NoLimit =>             c: Counts => false
            case MaxCount(count) =>     c: Counts => c.count < count
            case MaxBytes(nbytes) =>    c: Counts => c.nbytes < nbytes
            case MaxRetained(nbytes) => c: Counts => c.retained < nbytes
        }
        
        for (slot <- slots) {
            if (hide(slot)) {
                hidden.count += slot.count
                hidden.nbytes += slot.nbytes
            }
            else {
                out.write("%10d %10d %10d %s\n".format(slot.count, slot.nbytes, slot.retained, slot.classDef.name))
                total.count += slot.count
                total.nbytes += slot.nbytes
            }
        }
*/
