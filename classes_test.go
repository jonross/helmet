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

/*
import (
    . "launchpad.net/gocheck"
    "testing"
)

type CTSuite struct{} 
var _ = Suite(&CTSuite{})

func (s *CTSuite) TestClassMatching(c *C) {

    ci := &ClassInfo{}

    add := func(name string, hid int, superHid int) {
        ci.AddClassDef(name, hid, superHid, []Field{})
    }

    add("java.lang.Object", 1, 0)
    add("java.lang.String", 2, 1)
    add("java.lang.Number", 3, 1)
    add("java.lang.Integer", 4, 3)
    add("java.lang.Long", 5, 3)
    add("java.util.List", 6, 1)
    add("java.util.Map", 7, 1)

    for _, class := ci.AllClasses() {
        ci.Cook()
    }

    bits := NewBitSet(10)

    verify := func(name string, include bool, set []int) {
        ci.MatchClasses(bits, name, include)
        for hid := 1: hid <= 7; hid++ {
            if (HasInt(set, hid)) {
                
            }
            else {
            }
        }
    }

class TestDefs extends FunSuite {
    
    test("matching classes") {
        val info = defs
        val bits = new BitSet(10)
        def verify(hids: Int*) {
            hids.foreach(hid => bits(hid) || fail("hid " + hid + " not set"))
            (1 to 7).toList.diff(hids).foreach(hid => bits(hid) && fail ("hid " + hid + " set"))
        }
        verify()
        info.matchClasses(bits, "java.lang.Object", true)
        verify(1, 2, 3, 4, 5, 6, 7)
        info.matchClasses(bits, "java.lang.Number", false)
        verify(1, 2, 6, 7)
        info.matchClasses(bits, "java.util.*", false)
        verify(1, 2)
    }

*/

