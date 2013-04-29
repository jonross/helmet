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

type StepRole int

const (
    StepNone StepRole = iota// this step has no special role
    StepGroup  // this step is the grouping node
    StepMember // this step is the member node
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
    // What role when objects match this step
    role StepRole
}

type Query []*Step

// Implemented by types that can collect group / member object ids
// during a search
//
type Collector interface {
    Collect(ObjectId, ObjectId)
}

// Holds search state around one Step
//
type Finder struct {
    // heap being searched
    *Heap
    // underlying query step
    *Step
    // who's collecting results
    Collector
    // what class IDs match Step.types
    classes BitSet
    // does this step skip skipped classes
    skip bool
    // current object id at this Finder
    focus ObjectId
    // object IDs to be considered
    stack []ObjectId
    // next finder in query, or nil at end
    next *Finder
    // shortcut to the Finder for the group step
    group *Finder
    // shortcut to the Finder for the member step
    member *Finder
    // what pass through this step is this
    pass int
    // what objects have been skipped, as of what pass
    skipped []int
}

func SearchHeap(heap *Heap, query Query, coll Collector) {

    // Build finders & chain them

    var finder, group, member *Finder

    for i := len(query) - 1; i >= 0; i-- {
        prev := &Finder{
            Heap: heap,
            Step: query[i],
            Collector: coll,
            classes: heap.CidsMatching(query[i].types),
            skip: query[i].skip && i > 0,
            focus: 0,
            stack: make([]ObjectId, 0, 10000),
            next: finder,
            pass: 0,
            skipped: make([]int, heap.MaxObjectId + 1),
        }
        finder = prev
        if query[i].role == StepGroup {
            group = finder
        } else if query[i].role == StepMember {
            member = finder
        }
    }

    // Establish shortcuts to group & member nodes

    for f := finder; f != nil; f = f.next {
        f.group = group
        f.member = member
    }

    // Run the finder chain for each object that matches the first node

    for oid := ObjectId(1); oid <= heap.MaxObjectId; oid++ {
        if finder.classes.Has(uint32(heap.OidClass(oid).Cid)) {
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
    class := heap.OidClass(oid)
    if finder.classes.Has(uint32(class.Cid)) {
        // Object is a match at this query step
        if finder.next != nil {
            // Not at last query step?  Let next step handle adjacent nodes.
            if (finder.next.Step.to) {
                for dst, pos := heap.OutEdges(oid); pos != 0; dst, pos = heap.NextOutEdge(pos) {
                    finder.next.check(dst)
                }
            } else {
                for dst, pos := heap.InEdges(oid); pos != 0; dst, pos = heap.NextInEdge(pos) {
                    finder.next.check(dst)
                }
            }
        } else {
            // Complete match of finder chain
            finder.Collect(finder.group.focus, finder.member.focus)
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

