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
    . "launchpad.net/gocheck"
)

type GraphSuite struct{}
var _ = Suite(&GraphSuite{})

func (s *GraphSuite) TestEdges(c *C) {
    verifyGraph(c, makeGraph(edges_2), edges_2)
}

// Verify a graph against its raw edge data.  //
func verifyGraph(c *C, g *Graph, edges[][]int) {
    for _, list := range edges {
        node := ObjectId(list[0])
        var actual []int
        for n, pos := g.OutEdges(node); pos != 0; n, pos = g.NextOutEdge(pos) {
            actual = append(actual, int(n))
        }
        // EdgeSet filling approach in graph.go reverses the initial edge order.
        IntAryReverse(actual)
        // Remove zeroes from expected results, graph walk won't return them.
        expected := make([]int, 0)
        for i, edge := range list {
            if i > 0 && edge != 0 {
                expected = append(expected, edge)
            }
        }
        if ! IntAryEq(expected, actual) {
            c.Errorf("Wrong edge list for %d, wanted %v, got %v\n", node, expected, actual)
        }
    }
}

// Generate a sample graph.
//
func makeGraph(edges [][]int) *Graph {
    var src, dst []ObjectId
    for _, list := range edges {
        for _, node := range list[1:] {
            src = append(src, ObjectId(list[0]))
            dst = append(dst, ObjectId(node))
        }
    }
    return NewGraph(src, dst)
}

// Sample edge data for use with makeGraph
//
var edges_2 = [][]int {
    []int{1, 2, 19, 23},
    []int{2, 3, 6},
    []int{3, 5},
    []int{4},
    []int{5, 0, 4},
    []int{6, 5, 7},
    []int{7, 8, 9, 10},
    []int{8, 6, 16},
    []int{9, 18},
    []int{10, 11, 0, 14, 15},
    []int{11, 12, 13},
    []int{12},
    []int{13},
    []int{14, 0},
    []int{15},
    []int{16, 17},
    []int{17, 18},
    []int{18},
    []int{19, 20, 21, 22},
    []int{20},
    []int{21, 0},
    []int{22},
    []int{23, 24},
    []int{0, 24, 25, 26},
    []int{25, 26},
    []int{26, 23},
    []int{27, 28, 29},
    []int{28},
    []int{29},
    []int{30, 10},
}

