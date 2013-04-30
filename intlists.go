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

const NIL = uint32(0)

// Lists of integers accessed by list index, with free conses managed as a single
// large slice, for GC-friendliness.
//
type IntLists struct {
    // first cons cell index of each list
    firsts []uint32
    // last cons cell index of each list
    lasts []uint32
    // every two slots in this slice is one cons cell
    chains []uint32
    // head of free cons chain
    freelist uint32
}

func NewIntLists(maxIndex uint32) *IntLists {
    return &IntLists {
        firsts: make([]uint32, maxIndex + 1),
        lasts: make([]uint32, maxIndex + 1),
        chains: make([]uint32, 2, 1000000), // 0 means nil so the first cons isn't used
    }
}

// Add a value to the indicated list.
//
func (ls *IntLists) Add(id uint32, value uint32) {
    // put value in a new cons cell
    cons := ls.alloc()
    ls.chains[cons] = value
    ls.chains[cons+1] = NIL
    if ls.firsts[id] == NIL {
        // this is the first cell in the list
        ls.firsts[id] = cons
        ls.lasts[id] = cons
    } else {
        // make this the new last + the old last point to the new one
        last := ls.lasts[id]
        ls.lasts[id] = cons
        ls.chains[last+1] = cons
    }
}

// Clear the indicated list.
//
func (ls *IntLists) Clear(id uint32) {
    // Put released conses on the free list
    for cons := ls.firsts[id]; cons != NIL; {
        next := ls.chains[cons+1]
        ls.free(cons)
        cons = next
    }
    ls.firsts[id] = NIL
    ls.lasts[id] = NIL
}

// Return the head of a list, or 0 if none.
//
func (ls *IntLists) Head(id uint32) uint32 {
    cons := ls.firsts[id]
    if cons != NIL {
        return ls.chains[cons]
    }
    return 0
}

// Iterate the indicated list.  Usage:
//
//    for val, pos := ls.Walk(id); pos != 0; val, pos = ls.Next(pos) {
//        ...
//
func (ls *IntLists) Walk(id uint32) (uint32, uint32) {
    cons := ls.firsts[id]
    if cons != NIL {
        return ls.chains[cons], cons
    }
    return 0, NIL
}

func (ls *IntLists) Next(cons uint32) (uint32, uint32) {
    next := ls.chains[cons+1]
    if next != NIL {
        return ls.chains[next], next
    }
    return 0, NIL
}

// Allocate a cons cell.
//
func (ls *IntLists) alloc() uint32 {
    if ls.freelist != NIL {
        cons := ls.freelist
        ls.freelist = ls.chains[cons+1]
        return cons
    }
    cons := len(ls.chains)
    ls.chains = append(ls.chains, 0)
    ls.chains = append(ls.chains, NIL)
    return uint32(cons)
}

// "Free" a cons cell (put on free list.)
//
func (ls *IntLists) free(cons uint32) {
    ls.chains[cons+1] = ls.freelist
    ls.freelist = cons
}
