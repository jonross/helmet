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
    "log"
)

type GCRoots struct {
    // Heap IDs of gc roots
    hids []HeapId
    // Which ones are live, as indexed by object ID
    // TODO make smaller
    live []bool
    // How many are live
    numLive int
    // Function to check liveness based on "set garbage" value
    CanSee func(ObjectId) bool
}

func (gcr *GCRoots) AddRoot(hid HeapId) {
    gcr.hids = append(gcr.hids, hid)
}

func (gcr *GCRoots) LinkMasterRoot(bag *RefBag) {
    for _, root := range gcr.hids {
        bag.AddReference(1, root)
    }
}

func (gcr *GCRoots) FindLiveObjects(g *ObjectIdGraph, resolver func(HeapId) ObjectId, maxOid ObjectId) {
    gcr.live = make([]bool, maxOid + 1)
    gcr.live[1] = true
    for _, root := range gcr.hids {
        // log.Printf("Find roots from oid=%d hid=%x\n", resolver(root), root)
        gcr.findFrom(g, resolver(root))
    }
    // Correct counts for master root
    log.Printf("%d of %d objects are live", gcr.numLive - 1, maxOid - 1)
    gcr.SetVisible("live")
}

func (gcr *GCRoots) findFrom(g *ObjectIdGraph, oid ObjectId) {
    if oid != 0 && ! gcr.live[oid] {
        gcr.live[oid] = true
        gcr.numLive++
        for dst, pos := g.OutEdges(oid); pos != 0; dst, pos = g.NextOutEdge(pos) {
            gcr.findFrom(g, dst)
        }
    }
}

func (gcr *GCRoots) SetVisible(tag string) {
    switch tag {
        case "all":
            gcr.CanSee = func(oid ObjectId) bool { return true }
        case "live":
            gcr.CanSee = func(oid ObjectId) bool { return gcr.live[oid] }
        case "nonlive":
            gcr.CanSee = func(oid ObjectId) bool { return ! gcr.live[oid] }
        default:
            panic("Bad visibility tag: " + tag)
    }
}

