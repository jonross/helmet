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

type BitSet []uint64

func MakeBitSet(size uint32) BitSet {
    size = 1 + (size - 1) / 64
    return make([]uint64, size, size)
}

func (b BitSet) set(i uint32) {
    b[i/64] |= 1 << (i % 64)
}

func (b BitSet) clear(i uint32) {
    b[i/64] &^= 1 << (i % 64)
}

func (b BitSet) has(i uint32) bool {
    return b[i/64] & (1 << (i % 64)) != 0
}

// GC-friendly approach to building enormous arrays.  Start by making one e.g.
//   aa = make([][]uint32, 0, 10000)
// This continually appends to the last array in aa and grows aa as needed but
// never copies the 2nd-level arrays, which is O(n**2) in the worst case.  I've
// found this approach is 20% to 40% faster than a 1-D array.
//
func xappend32(aa [][]uint32, val uint32) [][]uint32 {
    a := aa[len(aa)-1]
    if len(a) == cap(a) {
        a = make([]uint32, 0, len(a))
        aa = append(aa, a)
    }
    a = append(a, val)
    return aa
}

// See xappend32()
//
func xappend64(aa [][]uint64, val uint64) [][]uint64 {
    a := aa[len(aa)-1]
    if len(a) == cap(a) {
        a = make([]uint64, 0, len(a))
        aa = append(aa, a)
    }
    a = append(a, val)
    return aa
}

// See xappend32()
//
func xappendOid(aa [][]ObjectId, val ObjectId) [][]ObjectId {
    a := aa[len(aa)-1]
    if len(a) == cap(a) {
        a = make([]ObjectId, 0, len(a))
        aa = append(aa, a)
    }
    a = append(a, val)
    return aa
}

// See xappend32()
//
func xappendHid(aa [][]HeapId, val HeapId) [][]HeapId {
    a := aa[len(aa)-1]
    if len(a) == cap(a) {
        a = make([]HeapId, 0, len(a))
        aa = append(aa, a)
    }
    a = append(a, val)
    return aa
}