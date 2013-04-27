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

// 1-based, assigned as we read class definitions from the dump
//
type ClassId uint32

// One of these per class we find in the heap dump
//
type ClassDef struct {
    // have we completed post-processing
    cooked bool
    // ptr back to parent heap
    heap *Heap
    // demangled name
    Name string
    // assigned unique id
    Cid ClassId
    // native id from heap
    Hid HeapId
    // native id of superclass
    SuperHid HeapId
    // superclass def, after classdef is cooked
    super *ClassDef
    // instance member information
    fields []*Field
    // heap ids of static referees
    staticRefs []HeapId
    // is this def for java.lang.Object
    IsRoot bool
    // # of instances in heap dump
    NumInstances uint32
    // # of bytes in all instances
    NumBytes uint64
    // size of instance layout, including superclasses; returned by layoutSize()
    span uint32
    // offsets of reference fields, including superclases; returned by refOffsets()
    refs []uint32
    // can instances of this class be skipped in graph searches
    Skip bool
}

// One of these for each non-static member in a class def
//
type Field struct {
    // field name from java source
    Name string
    // type information
    JType *JType
    // offset from start of the this class's fields
    Offset uint32
}

// Information about java value types, "basic type" as defined in HPROF spec
//
type JType struct {
    // JVM short class name for an array of this type e.g. "[I"
    ArrayClass string
    // true if this is an object type, not a primitive type
    IsObj bool
    // size in bytes
    Size uint32
    // assigned when found in heap
    Class *ClassDef
}

// Create a ClassDef given the minimal required information.
//
func makeClassDef(heap *Heap, name string, cid ClassId, hid HeapId, superHid HeapId,
                    fields []*Field, staticRefs []HeapId) *ClassDef {
    isRoot := name == "java.lang.Object" || name == "java/lang/Object"
    return &ClassDef{cooked: false, heap: heap, Name: name, Cid: ClassId(cid),
                        Hid: hid, SuperHid: superHid, super: nil, fields: fields,
                        IsRoot: isRoot, NumInstances: 0, NumBytes: 0, span: 0,
                        refs: nil, staticRefs: staticRefs}
}

// Return this class's superclass ClassDef.
// Cooks the class as a side effect.
//
func (def *ClassDef) SuperDef() *ClassDef {
    if ! def.cooked {
        def.Cook()
    }
    return def.super
}

// Is this class a subclass of another class.
// Cooks the class as a side effect.
//
func (def *ClassDef) IsSubclassOf(super *ClassDef) bool {
    if ! def.cooked {
        def.Cook()
    }
    return !def.IsRoot && (def.SuperHid == super.Hid || def.SuperDef().IsSubclassOf(super))
}

// Return offsets of reference fields, including superclasses.
// Cooks the class as a side effect.
//
func (def *ClassDef) RefOffsets() []uint32 {
    if ! def.cooked {
        def.Cook()
    }
    return def.refs
}

// Return size of the instance layout in bytes, including superclasses.
// Cooks the class as a side effect.
//
func (def *ClassDef) LayoutSize() uint32 {
    if ! def.cooked {
        def.Cook()
    }
    return def.span
}

// Update instance count & size
//
func (def *ClassDef) AddObject(size uint32) {
    def.NumInstances += 1
    def.NumBytes += uint64(size)
}

// "Cook" the class def, generally after the heap dump is read but in any case only works
// after all superclass defs have been identified.  Resolves the superclass pointer
// and computes reference offsets.  Cooks all superclasses as a side effect.
//
func (def *ClassDef) Cook() {

    if def.cooked {
        return
    }
    if ! def.IsRoot {
        def.super = def.heap.HidClass(def.SuperHid)
        if def.super == nil {
            log.Fatalf("No super def for %v\n", def)
        }
        def.super.Cook()
    }

    // Determine size of instance layout

    span := uint32(0)
    if ! def.IsRoot {
        span = def.super.LayoutSize()
    }
    for _, field := range def.fields {
        span += field.JType.Size
    }
    def.span = span

    // Determine offsets of reference fields

    base := uint32(0)
    offsets := []uint32{}
    if ! def.IsRoot {
        base = def.super.LayoutSize()
        for _, offset := range def.super.RefOffsets() {
            offsets = append(offsets, offset)
        }
    }
    for _, field := range def.fields {
        if field.JType.IsObj {
            offsets = append(offsets, base)
        }
        base += field.JType.Size
    }
    def.refs = offsets
    // log.Printf("%s has refs at %v\n", def.Name, def.refs)

    def.cooked = true
}
