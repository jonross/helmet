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

package helmet

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
    // is this def for java.lang.Object
    IsRoot bool
    // # of instances in heap dump
    NumInstances uint32
    // # of bytes in all instances
    NumBytes uint64
}

// One of these for each non-static member in a class def
//
type Field struct {
    // field name from java source
    Name string
    // type information
    jtype *JType
    // offset from start of the this class's fields
    Offset uint32
}

// Information about java value types, "basic type" as defined in HPROF spec
//
type JType struct {
    // JVM short class name for an array of this type e.g. "[I"
    arrayClass string
    // true if this is an object type, not a primitive type
    isObj bool
    // size in bytes
    size uint32
    // assigned when found in heap
    hid HeapId
}

// Create a ClassDef given the minimal required information.
//
func makeClassDef(heap *Heap, name string, cid ClassId, 
                    hid HeapId, superHid HeapId, fields []*Field) *ClassDef {
    isRoot := name == "java.lang.Object" || name == "java/lang/Object"
    return &ClassDef{cooked: false, heap: heap, Name: name, Cid: ClassId(cid), 
                        Hid: hid, SuperHid: superHid, super: nil, fields: fields, 
                        IsRoot: isRoot, NumInstances: 0, NumBytes: 0}
}

// Return this class's superclass ClassDef.  Not usable until the class def
// has been cooked.
//
func (def *ClassDef) SuperDef() *ClassDef {
    def.ensureCooked("SuperDef")
    return def.super
}

// Is this class a subclass of another class.  Same caveats as SuperDef.
//
func (def *ClassDef) IsSubclassOf(super *ClassDef) bool {
    return !def.IsRoot && (def.SuperHid == super.Hid || def.SuperDef().IsSubclassOf(super))
}

// Update instance count & size
//
func (def *ClassDef) addObject(size uint32) {
    def.NumInstances += 1
    def.NumBytes += uint64(size)
}

// "Cook" the class def after the heap dump is read.  Resolves the superclass pointer
// (which can't be done until all classes are known) and computes reference offsets.
// Cooks all superclasses as a side effect.
//
func (def *ClassDef) cook() {
    if def.cooked {
        return
    }
    if ! def.IsRoot {
        def.super = def.heap.HidClass(def.SuperHid)
        if def.super == nil {
            log.Fatalf("No super def for %v\n", def)
        }
        def.super.cook()
    }
    def.cooked = true
}

// Verify class is cooked else panic.
//
func (def *ClassDef) ensureCooked(funcName string) {
    if ! def.cooked {
        log.Fatalf("ClassDef.%s invoked on raw def: %v\n", funcName, def)
    }
}

