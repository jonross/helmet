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

func TestObjectMap(t *testing.T) {

    var hids [1000000]HeapId
    var om ObjectMap

    lastHid := HeapId(0)
    for i, _ := range hids {
        lastHid += 1 + HeapId(rand.Int63n(1000))
        om.Add(lastHid, ObjectId(i + 1))
        hids[i] = lastHid
    }

    om.PostProcess()
    for i, hid := range hids {
        oid := om.Get(hid)
        if oid != ObjectId(i + 1) {
            t.Fatalf("Expected %d -> %d but was %d\n", hid, i + 1, oid)
        }
    }
}
