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
    "math/rand"
    "testing"
)

func TestDemangle(t *testing.T) {
    try := func(input, wanted string) {
        result := Demangle(input)
        if result != wanted {
            t.Errorf("For %s wanted %s but got %s\n", input, wanted, result)
        }
    }
    try("[[I", "int[][]")
    try("[Lcom/foo/Bar;", "com.foo.Bar[]")
    try("com/foo/Bar", "com.foo.Bar")
}

func TestBitSet(t *testing.T) {
    var flags [1000000]bool
    bits := NewBitSet(uint32(len(flags)))
    for i, _ := range flags {
        bits.Set(uint32(i))
        if rand.Int() % 2 == 0 {
            flags[i] = true
        } else {
            bits.Clear(uint32(i))
        }
    }
    for i, flag := range flags {
        if bits.Has(uint32(i)) != flag {
            t.Fatalf("Bit %d should be %v but is %v\n", i, flag, bits.Has(uint32(i)))
        }
    }
}

func TestUndoableBitSet(t *testing.T) {
    var flags [1000000]bool
    bits := NewUndoableBitSet(uint32(len(flags)))
    for i, _ := range flags {
        if rand.Int() % 5 == 0 {
            flags[i] = true
            bits.Set(uint32(i))
        }
    }
    for i, flag := range flags {
        if bits.Has(uint32(i)) != flag {
            t.Fatalf("Bit %d should be %v but is %v\n", i, flag, bits.Has(uint32(i)))
        }
    }
    bits.Undo()
    for i, _ := range flags {
        if bits.Has(uint32(i)) {
            t.Fatalf("Bit %d should be unset\n", i)
        }
    }
}
