// GENERATED FILE
// DO NOT EDIT

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
    "sync"
)

// An immutable adjacency-list graph representation with all lists merged to a
// single slice for minimal mark/sweep overhead.
//
type ObjectIdGraph struct {
    // highest node id
    MaxNode ObjectId
    // out edges per node
    outs *ObjectIdEdgeSet
    // in edges per node
    ins *ObjectIdEdgeSet
}

// In/out edges use an identical structure, just with the edge direction reversed.
// TODO: data compression on offsets; use bitset for isStart
//
type ObjectIdEdgeSet struct {
    // Merged edge list of all adjacent nodes
    edges []ObjectId
    // Offset of each node's edges in merged list, or 0 if none
    offsets []int
    // Indicates which offset values are the start of a node's edge list
    isStart []bool
}

// Use this form to create a ObjectIdGraph if only the source and destination nodes are
// known for each edge.  Finds the maximum node ID then hands off to NewObjectIdGraph1.
//
func NewObjectIdGraph(from []ObjectId, to[]ObjectId) *ObjectIdGraph {

    maxFrom := ObjectId(0)
    maxTo := ObjectId(0)

    var waiter sync.WaitGroup
    waiter.Add(2)

    maxFinder := func(nodes []ObjectId, result *ObjectId) {
        max := ObjectId(0)
        for _, node := range nodes {
            if node > max {
                max = node
            }
        }
        *result = max
        waiter.Done()
    }

    go maxFinder(from, &maxFrom)
    go maxFinder(to, &maxTo)
    waiter.Wait()

    if maxTo > maxFrom {
        maxFrom = maxTo
    }

    return NewObjectIdGraph1(maxFrom, from, to)
}

// Use this form to create a ObjectIdGraph if the source and destination nodes + the
// maximum node ID are known.  Calculates the edge counts per node then hands
// off to NewObjectIdGraph2
//
func NewObjectIdGraph1(maxNode ObjectId, from []ObjectId, to []ObjectId) * ObjectIdGraph {

    // log.Printf("maxNode is %d\n", maxNode)

    var fromCounts []int
    var toCounts []int

    var waiter sync.WaitGroup
    waiter.Add(2)

    edgeCounter := func(nodes []ObjectId, result *[]int) {
        counts := make([]int, maxNode + 1)
        for _, node := range nodes {
            counts[node]++
        }
        *result = counts
        waiter.Done()
    }

    go edgeCounter(from, &fromCounts)
    go edgeCounter(to, &toCounts)
    waiter.Wait()

    return NewObjectIdGraph2(maxNode, from, to, fromCounts, toCounts)
}

// Use this form to create a ObjectIdGraph if the source and destination nodes + the
// edge counts from each source & destination are known.  Modifies count arrays.
//
func NewObjectIdGraph2(maxNode ObjectId, from, to []ObjectId, fromCounts, toCounts []int) *ObjectIdGraph {

    g := &ObjectIdGraph{MaxNode: maxNode}
    var waiter sync.WaitGroup
    waiter.Add(2)

    go func() {
        g.outs = newObjectIdEdgeSet(from, to, fromCounts)
        waiter.Done()
    }()

    go func() {
        g.ins = newObjectIdEdgeSet(to, from, toCounts)
        waiter.Done()
    }()

    waiter.Wait()
    return g
}

// Walk the out edges of a node, e.g.
//
//     for node, pos := g.OutEdges(n); pos != 0; node, pos := g.NextOutEdge(pos) {
//         ...
//     }
//
func (g *ObjectIdGraph) OutEdges(node ObjectId) (ObjectId, int) {
    return g.outs.walk(node)
}

func (g *ObjectIdGraph) NextOutEdge(pos int) (ObjectId, int) {
    return g.outs.next(pos)
}

// Walk the in edges of a node, e.g.
//
//     for node, pos := g.InEdges(n); pos != 0; node, pos := g.NextInEdge(pos) {
//         ...
//     }
//
func (g *ObjectIdGraph) InEdges(node ObjectId) (ObjectId, int) {
    return g.ins.walk(node)
}

func (g *ObjectIdGraph) NextInEdge(pos int) (ObjectId, int) {
    return g.ins.next(pos)
}

// Create an edge set.  Overwrites the count array as a side effect.
//
func newObjectIdEdgeSet(from []ObjectId, to []ObjectId, counts []int) *ObjectIdEdgeSet {

    e := &ObjectIdEdgeSet {
        edges: make([]ObjectId, len(from) + 1), // 1 entry per edge, index 0 not used
        isStart: make([]bool, len(from) + 2), // matches edges but also need a terminator entry
        offsets: make([]int, len(counts)), // 1 entry per node
    }

    // Determine the offset of the start of the edge list for each node

    offset := 1
    for node, count := range counts {
        if count == 0 {
            e.offsets[node] = 0
        } else {
            e.offsets[node] = offset
            e.isStart[offset] = true
            offset += count
        }
    }
    e.isStart[offset] = true // terminate last list

    // Populate the edge lists

    for i, node := range from {
        counts[node]--
        e.edges[e.offsets[node] + counts[node]] = to[i]
    }

    return e
}

// Start walking an edge set from the specified node.  If position == 0
// there are no edges from this node.
//
func (e *ObjectIdEdgeSet) walk(node ObjectId) (ObjectId, int) {
    offset := e.offsets[node]
    if offset == 0 {
        return 0, 0
    }
    edge := e.edges[offset]
    if edge > 0 {
        return edge, offset
    }
    return e.next(offset)
}

// Continue walking an edge set from the previous position.  Returns
// position == 0 if there are no more edges.
//
func (e *ObjectIdEdgeSet) next(offset int) (ObjectId, int) {
    offset++
    if e.isStart[offset] {
        return 0, 0
    }
    edge := e.edges[offset]
    if edge > 0 {
        return edge, offset
    }
    return e.next(offset)
}

// Debug output.
//
func logObjectIdDegrees(inOut string, maxNode ObjectId, counts []int) {
    log.Printf("Frequency of %s-degree across %d nodes\n", inOut, maxNode)
    stats := [11]int{}
    for i, _ := range counts {
        if i > 0 {
            degree := counts[i]
            if degree < 10 {
                stats[degree]++
            } else {
                stats[10]++
            }
        }
    }
    for i, _ := range stats {
        var prefix string
        if i == len(stats) - 1 {
            prefix = "+"
        } else {
            prefix = " "
        }
        log.Printf("%2d%s  %10s", i, prefix, counts[i])
    }
}
