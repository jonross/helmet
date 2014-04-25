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
    "strings"
)

// Demangle heap class names, e.g.
//
//     [[I                -> int[][]
//     [Lcom/foo/Bar;     -> com.foo.Bar[]
//     com/foo/Bar        -> com.foo.Bar
//
func Demangle(name string) string {
    dimen := 0
    for name[0] == '[' {
        name = name[1:]
        dimen++
    }
    if name[0] == 'L' {
        return Demangle(name[1:len(name)-1]) + strings.Repeat("[]", dimen)
    }
    if dimen > 0 {
        prim, ok := prims[name[0]]
        if ! ok {
            log.Fatalf("Unknown primitive in type spec %s\n", name)
        }
        return prim + strings.Repeat("[]", dimen)
    }
    return strings.Replace(name, "/", ".", -1)
}

var prims = map[byte]string{
    'Z': "boolean", 'C': "char", 'F': "float", 'D': "double",
    'B': "byte", 'S': "short", 'I': "int", 'J': "long",
}

func IntAryReverse(a []int) []int {
    j := len(a) - 1
    if j > 0 {
        for i := 0; i < j; {
            a[i], a[j] = a[j], a[i]
            i++
            j--
        }
    }
    return a
}

func IntAryEq(a, b []int) bool {
    if len(a) != len(b) {
        return false
    }
    for i, x := range a {
        if x != b[i] {
            return false
        }
    }
    return true
}

// A simple bit set, no frills, no bounds checking, no dynamic sizing.
//
type BitSet []uint64

func MakeBitSet(size uint32) BitSet {
    size = 1 + (size - 1) / 64
    return make([]uint64, size, size)
}

func (b BitSet) Set(i uint32) {
    b[i/64] |= 1 << (i % 64)
}

func (b BitSet) Clear(i uint32) {
    b[i/64] &^= 1 << (i % 64)
}

func (b BitSet) Has(i uint32) bool {
    return b[i/64] & (1 << (i % 64)) != 0
}

// A BitSet that maintains a list of bits that have been set, and can reset them.
//
type UndoableBitSet struct {
    bits BitSet
    haveSet []uint32
}

func NewUndoableBitSet(size uint32) *UndoableBitSet {
    return &UndoableBitSet{bits: MakeBitSet(size)}
}

func (ub *UndoableBitSet) Set(i uint32) {
    ub.bits.Set(i)
    ub.haveSet = append(ub.haveSet, i)
}

func (ub *UndoableBitSet) Has(i uint32) bool {
    return ub.bits.Has(i)
}

func (ub *UndoableBitSet) Undo() {
    if ub.haveSet != nil {
        for _, bit := range ub.haveSet {
            ub.bits.Clear(bit)
        }
        ub.haveSet = ub.haveSet[:0]
    }
}
