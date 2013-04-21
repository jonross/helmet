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
    "sync"
)

type Graph struct {
    // out edges per node
    outs *edgeSet
    // in edges per node
    ins *edgeSet
}

type edgeSet struct {
    // Merged edge list of all nodes
    edges []ObjectId
    // Offset of each node's edges in merged list, or 0 if none
    offsets []int
    // Indicates which offsets are the start of an edge set
    isStart []bool
}

// Use this form to create a Graph if only the source and destination nodes are
// known for each edge.
//
func NewGraph(src []ObjectId, dst[]ObjectId) *Graph {

    srcMax := ObjectId(0)
    dstMax := ObjectId(0)

    var wg sync.WaitGroup
    wg.Add(2)

    maxer := func(a []ObjectId, result *ObjectId) {
        max := ObjectId(0)
        for _, node := range a {
            if node > max {
                max = node
            }
        }
        *result = max
        wg.Done()
    }

    go maxer(src, &srcMax)
    go maxer(dst, &dstMax)
    wg.Wait()

    if dstMax > srcMax {
        srcMax = dstMax
    }

    return NewGraphWithMax(src, dst, srcMax)
}

// Use this form to create a Graph if the source and destination nodes + the
// maximum node ID are known.
//
func NewGraphWithMax(src []ObjectId, dst []ObjectId, maxNode ObjectId) * Graph {

    var srcCounts []int
    var dstCounts []int

    var wg sync.WaitGroup
    wg.Add(2)

    counter := func(a []ObjectId, result *[]int) {
        counts := make([]int, maxNode + 1)
        for _, node := range a {
            counts[node]++
        }
        *result = counts
        wg.Done()
    }

    go counter(src, &srcCounts)
    go counter(dst, &dstCounts)
    wg.Wait()

    return NewGraphWithCounts(src, dst, srcCounts, dstCounts)
}

// Use this form to create a Graph if the source and destination nodes + the
// edge counts from each source & destination are known.  Modifies count arrays.
//
func NewGraphWithCounts(src []ObjectId, dst []ObjectId, srcCounts []int, dstCounts []int) *Graph {

    g := &Graph{}
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        g.outs = newEdgeSet(src, dst, srcCounts)
        wg.Done()
    }()

    go func() {
        g.ins = newEdgeSet(dst, src, dstCounts)
        wg.Done()
    }()

    wg.Wait()
    return g
}

// Walk the out edges of a node, e.g.
//
//     for pos, node := g.OutEdges(n); pos != 0; pos, node := g.NextOutEdge(pos) {
//         ...
//     }
//
func (g *Graph) OutEdges(node ObjectId) (ObjectId, int) {
    return g.outs.walk(node)
}

func (g *Graph) NextOutEdge(pos int) (ObjectId, int) {
    return g.outs.next(pos)
}

// Walk the in edges of a node, e.g.
//
//     for pos, node := g.InEdges(n); pos != 0; pos, node := g.NextInEdge(pos) {
//         ...
//     }
//
func (g *Graph) InEdges(node ObjectId) (ObjectId, int) {
    return g.ins.walk(node)
}

func (g *Graph) NextInEdge(pos int) (ObjectId, int) {
    return g.ins.next(pos)
}

// Create an edge set.  Overwrites the count array as a side effect.
//
func newEdgeSet(src []ObjectId, dst []ObjectId, counts []int) *edgeSet {

    e := &edgeSet {
        edges: make([]ObjectId, len(src) + 1), // 1 entry per edge, index 0 not used
        isStart: make([]bool, len(src) + 2), // matches edges but also need a terminator entry
        offsets: make([]int, len(counts)), // 1 entry per node
    }

    // Determine the offset of the start of the edge list for each ndoe

    offset := 1
    for node, count := range counts {
        if count > 0 {
            e.offsets[node] = offset
            e.isStart[offset] = true
            offset += count
        }
    }
    e.isStart[offset] = true // terminate last list

    // Populate the edge lists

    for i, node := range src {
        counts[node]--
        e.edges[e.offsets[node] + counts[node]] = dst[i]
    }

    return e
}

// Start walking an edge set from the specified node.
//
func (e *edgeSet) walk(node ObjectId) (ObjectId, int) {
    offset := e.offsets[node]
    if offset == 0 {
        return 0, 0
    }
    return e.edges[offset], offset
}

func (e *edgeSet) next(offset int) (ObjectId, int) {
    offset++
    if e.isStart[offset] {
        return 0, 0
    }
    return e.edges[offset], offset
}


