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

// Represents one step in a query.
//
type Step struct {
    // The type name / wildcard, e.g. "ArrayList" 
    types string
    // Optional variable name, e.g. "list", else ""
    varName string
    // Follow refs to this node; true if ->, false if <-
    to bool
    // Skip instances of skipped classes
    skip bool
}

// Implemented by types that can collect group / member object ids
// during a search
//
type Collector interface {
    Collect([]ObjectId)
}

// Maps finder foci to collector arg list
//
type CollectorArgs struct {
    Collector
    indices []int
    foci []*Finder
    funargs []ObjectId
}

// Holds search state around one Step
//
type Finder struct {
    // index in finder chain
    index int
    // heap being searched
    *Heap
    // underlying query step
    *Step
    // what class IDs match Step.types
    classes BitSet
    // does this step skip skipped classes
    skip bool
    // current object id at this Finder
    focus ObjectId
    // common arg-passing info
    *CollectorArgs
    // object IDs to be considered
    stack []ObjectId
    // next finder in query, or nil at end
    next *Finder
    // what pass through this step is this
    pass int
    // what objects have been skipped, as of what pass
    skipped []int
}

func SearchHeap(heap *Heap, query []*Step, coll Collector, argIndices []int) {

    // Build finders & chain them

    finders := make([]*Finder, len(query))

    for i, step := range query {
        finders[i] = &Finder{
            index: i,
            Heap: heap,
            Step: step,
            classes: heap.CidsMatching(query[i].types),
            skip: step.skip && i > 0,
            focus: 0,
            stack: make([]ObjectId, 0, 10000),
            next: nil,
            pass: 0,
            // TODO make this more compact
            skipped: make([]int, heap.MaxObjectId + 1),
        }
    }

    for i := 0; i < len(query)-1; i++ {
        finders[i].next = finders[i+1]
    }

    // Save state about collector args

    cargs := &CollectorArgs{
        Collector: coll,
        indices: argIndices,
        foci: make([]*Finder, len(argIndices)),
        funargs: make([]ObjectId, len(argIndices)),
    }

    for i, index := range argIndices {
        cargs.foci[i] = finders[index]
    }

    for _, finder := range finders {
        finder.CollectorArgs = cargs
    }

    // Run the finder chain for each object that matches the first node.  Use max oid
    // from graph, not heap, because objects near the end may not have references.

    finder := finders[0]
    for oid := ObjectId(1); oid <= heap.Graph.MaxNode; oid++ {
        class := heap.ClassOf(oid)
        if finder.classes.Has(uint32(class.Cid)) {
            finder.check(oid)
        }
    }
}

// Check an object ID for a match against the matching classes, plus any IDs that we 
// check as a result of skipping the object.  This uses an inline stack to DFS because
// I'd written it that way in Scala to keep from blowing the JVM stack.
//
func (finder *Finder) check(oid ObjectId) {
    finder.pass += 1
    finder.doCheck(oid)
    for {
        top := len(finder.stack) - 1
        if top < 0 {
            return
        }
        oid := finder.stack[top]
        finder.stack = finder.stack[:top]
        finder.doCheck(oid)
    }
}

// Check one object against one finder in the chain.
//
func (finder *Finder) doCheck(oid ObjectId) {
    heap := finder.Heap
    finder.focus = oid
    class := heap.ClassOf(oid)
    log.Printf("doCheck %d %d a %s\n", finder.index, oid, class.Name)
    if finder.classes.Has(uint32(class.Cid)) {
        // Object is a match at this query step
        if finder.next != nil {
            // Not at last query step?  Let next step handle adjacent nodes.
            if (finder.next.Step.to) {
                for dst, pos := heap.OutEdges(oid); pos != 0; dst, pos = heap.NextOutEdge(pos) {
                    log.Printf("follow %d\n", dst)
                    finder.next.check(dst)
                }
            } else {
                for dst, pos := heap.InEdges(oid); pos != 0; dst, pos = heap.NextInEdge(pos) {
                    log.Printf("follow %d\n", dst)
                    finder.next.check(dst)
                }
            }
        } else {
            // Complete match of finder chain; call function
            for i, _ := range finder.funargs {
                finder.funargs[i] = finder.foci[i].focus
            }
            finder.Collect(finder.funargs)
        }
    } else if finder.skip && class.Skip {
        // Skipped object; search adjacent nodes using the same finder.  I've
        // found we must reset the state of what objects have been skipped at
        // this step for each pass, otherwise if (for example) we get 'String x
        // <<- MyObject y' and the strings are held in a data structure whose
        // internals are elided, we will ignore all paths from all x to y after
        // the first one.  Unfortunately for heaps with large such structures,
        // the skipped set can get pretty big.  TODO: make less expensive
        explore := finder.skipped[oid] < finder.pass
        finder.skipped[oid] = finder.pass
        if explore {
            if (finder.Step.to) {
                for dst, pos := heap.OutEdges(oid); pos != 0; dst, pos = heap.NextOutEdge(pos) {
                    finder.stack = append(finder.stack, dst)
                }
            } else {
                for dst, pos := heap.InEdges(oid); pos != 0; dst, pos = heap.NextInEdge(pos) {
                    finder.stack = append(finder.stack, dst)
                }
            }
        }
    }
}

/*
// TODO: do wildcard matching differently
val isWild = target.types endsWith ".*"
val typePrefix = target.types.substring(0, target.types.length - 1)

if (baseClass != null)
    matchingClasses.put(baseClass.classId, 1)

for (classDef <- heap.classes getAll)
    if (baseClass != null && (classDef hasSuper baseClass))
        matchingClasses.put(classDef.classId, 1)
    else if (isWild && classDef.name.startsWith(typePrefix))
        matchingClasses.put(classDef.classId, 1)
println(target.types + " matches " + matchingClasses.size + " classes")
*/

