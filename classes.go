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
    // ptr back to parent heap
    *Heap
    // demangled name
    Name string
    // assigned unique id
    Cid ClassId
    // native id from heap
    Hid HeapId
    // native id of superclass
    SuperHid HeapId
    // have we completed post-processing
    cooked bool
    // superclass def, after classdef is cooked
    super *ClassDef
    // instance member information
    fields []*Field
    // is this def for java.lang.Object
    IsRoot bool
    // # of instances in heap dump
    NumInstances uint32
    // # of bytes in all instances
    NumBytes uint64
    // heap ids of static referees
    staticRefs []HeapId
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
    // JVM short class name for an array of this type e.g. "[I" for int[]
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
func NewClassDef(heap *Heap, name string, cid ClassId, hid HeapId, superHid HeapId,
                    fields []*Field, staticRefs []HeapId) *ClassDef {
    isRoot := name == "java.lang.Object"
    return &ClassDef{
        Heap: heap,
        Name: name,
        Cid: cid,
        Hid: hid,
        SuperHid: superHid,
        cooked: false,
        super: nil,
        IsRoot: isRoot,
        fields: fields,
        NumInstances: 0,
        NumBytes: 0,
        staticRefs: staticRefs,
        span: 0,
        refs: nil,
        Skip: false,
    }
}

// Return this class's superclass ClassDef.
// Cooks the class as a side effect.
//
func (class *ClassDef) Super() *ClassDef {
    return class.Cook().super
}

// Is this class a subclass of another class.
// Cooks the class as a side effect.
//
func (class *ClassDef) IsSubclassOf(super *ClassDef) bool {
    return !class.IsRoot && (class.SuperHid == super.Hid || class.Super().IsSubclassOf(super))
}

// Return offsets of reference fields, including superclasses.
// Cooks the class as a side effect.
//
func (class *ClassDef) RefOffsets() []uint32 {
    return class.Cook().refs
}

// Return size of the instance layout in bytes, including superclasses.
// Cooks the class as a side effect.
//
func (class *ClassDef) Span() uint32 {
    return class.Cook().span
}

// Update instance count & size
//
func (class *ClassDef) AddObject(size uint32) {
    class.NumInstances += 1
    class.NumBytes += uint64(size)
}

// "Cook" the class def, generally after the heap dump is read but in any case only works
// after all superclass defs have been identified.  Resolves the superclass pointer
// and computes reference offsets.  Cooks all superclasses as a side effect.
//
func (class *ClassDef) Cook() *ClassDef {

    if class.cooked {
        return class
    }
    if ! class.IsRoot {
        class.super = class.HidClass(class.SuperHid)
        if class.super == nil {
            log.Fatalf("No super def for %v\n", class)
        }
        class.super.Cook()
    }

    // Determine size of instance layout

    span := uint32(0)
    if ! class.IsRoot {
        span = class.super.Span()
    }
    for _, field := range class.fields {
        span += field.JType.Size
    }
    class.span = span

    // Determine offsets of reference fields; includes references in superclasses.
    // Note instance dumps are laid out leaf class first, so the reference offsets
    // of a given class will be different for different subclasses.

    offsets := []uint32{}
    offset := uint32(0)
    c := class

    for ! c.IsRoot {
        for _, field := range c.fields {
            if field.JType.IsObj {
                offsets = append(offsets, offset)
            }
            offset += field.JType.Size
        }
        c = c.super
    }

    class.refs = offsets
    // log.Printf("%s has refs at %v\n", class.Name, class.refs)

    class.cooked = true
    return class
}
