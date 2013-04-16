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
)

const MaxHeapBits = 36
const MaxHeapId = HeapId(1 << MaxHeapBits - 1)

// Maps native heap ids to ObjectIds.  To save space we pay attention to only the
// lower 36 bits of the HID (which handles heaps up to 68G.)  We then have 1<<20
// maps, each of which maps the low 16 bits of the HID for the same high 20 bits.
//
type ObjectMap struct {
    mappings []map[uint16]ObjectId
    heapIds [][]uint64
}

func MakeObjectMap() *ObjectMap {
    numMaps := 1 << (MaxHeapBits - 16)
    heapIds := [][]uint64{make([]uint64, 0, 100000)}
    return &ObjectMap{make([]map[uint16]ObjectId, numMaps, numMaps), heapIds}
}

func (m *ObjectMap) add(hid HeapId, oid ObjectId) {
    if hid > MaxHeapId {
        log.Fatalf("Heap ID %d too big\n", hid)
    }
    slot := hid >> 16
    mapping := m.mappings[slot]
    if mapping == nil {
        m.mappings[slot] = make(map[uint16]ObjectId)
        mapping = m.mappings[slot]
    }
    m.heapIds = xappend64(m.heapIds, uint64(hid))
    return // TODO put back
    mapping[uint16(hid & 0xFFFF)] = oid
}

func (m * ObjectMap) get(hid HeapId) ObjectId {
    slot := hid >> 16
    mapping := m.mappings[slot]
    if mapping != nil {
        return mapping[uint16(hid & 0xFFFF)]
    }
    return 0
}
