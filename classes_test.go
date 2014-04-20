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
    . "launchpad.net/gocheck"
    . "github.com/jonross/gorgon"
)

type CTSuite struct{} 
var _ = Suite(&CTSuite{})

func (s *CTSuite) TestClassMatching(c *C) {

    h := NewHeap(8)

    add := func(name string, hid int, superHid int) {
        h.AddClass(name, Hid(hid), Hid(superHid), []string{}, []*JType{}, []Hid{})
    }

    add("java.lang.Object", 2, 0)
    add("java.lang.String", 3, 2)
    add("java.lang.Number", 4, 2)
    add("java.lang.Integer", 5, 4)
    add("java.lang.Long", 6, 4)
    add("java.util.List", 7, 2)
    add("java.util.Map", 8, 2)

    for _, class := range h.classesByHid {
        class.Cook()
    }

    bits := MakeBitSet(10)

    verify := func(set []int) {
        for hid := 2; hid <= 8; hid++ {
            if HasInt(set, hid) && !bits.Has(uint32(hid)) {
                c.Errorf("Expected HID %d to be present", hid)
            } else if ! HasInt(set, hid) && bits.Has(uint32(hid)) {
                c.Errorf("Expected HID %d to be absent", hid)
            }
        }
    }

    verify([]int{})

    h.MatchClasses(bits, "java.lang.Object", true)
    verify([]int{2, 3, 4, 5, 6, 7, 8})

    h.MatchClasses(bits, "java.lang.Number", false)
    verify([]int{2, 3, 7, 8})

    h.MatchClasses(bits, "java.util.*", false)
    verify([]int{2, 3})
}

