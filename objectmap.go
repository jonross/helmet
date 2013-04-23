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
    "sync"
)

const MaxHeapBits = 36
const MaxHeapId = HeapId(1 << MaxHeapBits - 1)
const NumSlots = 1 << (MaxHeapBits - 16)

// Maps native heap ids to ObjectIds.  To save space we pay attention to only the
// lower 36 bits of the HID (which handles heaps up to 68G.)  We then have 1<<20
// maps, each of which maps the low 16 bits of the HID for the same high 20 bits.
//
type ObjectMap [NumSlots]*omSlot

type omSlot struct {
    // Start by just saving the heap ids
    heapIds []uint16
    // and object ids
    objectIds []ObjectId
    // and later we'll put them in a map
    mapping map[uint16]ObjectId
}

func (m *ObjectMap) Add(hid HeapId, oid ObjectId) {
    if hid > MaxHeapId {
        log.Fatalf("Heap ID %x too big\n", hid)
    }
    index := hid >> 16
    slot := m[index]
    if slot == nil {
        slot = &omSlot{}
        m[index] = slot
    }
    slot.heapIds = append(slot.heapIds, uint16(hid & 0xFFFF))
    slot.objectIds = append(slot.objectIds, oid)
}

func (m *ObjectMap) PostProcess() {
    var wg sync.WaitGroup
    for _, slot := range m {
        if slot != nil {
            wg.Add(1)
            go func(slot *omSlot) {
                slot.mapping = make(map[uint16]ObjectId, len(slot.heapIds))
                for i, hid := range slot.heapIds {
                    slot.mapping[hid] = slot.objectIds[i]
                }
                slot.heapIds = nil
                slot.objectIds = nil
                wg.Done()
            }(slot)
        }
    }
    wg.Wait()
}

func (m *ObjectMap) Get(hid HeapId) ObjectId {
    slot := m[hid >> 16]
    if slot != nil {
        return slot.mapping[uint16(hid & 0xFFFF)]
    }
    return 0
}
