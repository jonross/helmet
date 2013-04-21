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
    "strings"
)

const (
    RecordsPerGB = 10000000 // rough estimate
)

// A native ID read from the heap dump
//
type HeapId uint64

// 1-based, assigned as we read instances from the dump
//
type ObjectId uint32

type Heap struct {
    // underlying reader
    *HProfReader
    // static strings from UTF8 records
    strings map[HeapId]string
    // maps HeapId of a class to HeapId of its name; we have to do this because
    // LOAD_CLASS and ... are different records.
    classNames map[HeapId]HeapId
    // how many class IDs have we assigned
    NumClasses uint32
    // how many object IDs have we assigned
    NumObjects uint32
    // heap IDs of GC roots
    gcRoots []HeapId
    // maps java value type tags to JType objects
    jtypes []*JType
    // class defs indexed by cid
    classes []*ClassDef
    // same, indexed by demangled class name
    classesByName map[string]*ClassDef
    // same, by native heap id
    classesByHid map[HeapId]*ClassDef
    // object cids, indexed by synthetic object id
    objectCids []ClassId
    // object sizes, indexed by same
    objectSizes []uint32
    // temporary mapping from HeapIds to ObjectIds
    objectMap *ObjectMap
    // packages to search for unqualified class names
    autoPrefixes []string
    // concurrent heap segment reader
    *segReader
}

// Processing options.
//

type HeapOptions struct {
    // do we need the reference graph
    NeedRefs bool
}

func NewHeap(reader *HProfReader, options *HeapOptions) *Heap {

    heap := &Heap{HProfReader: reader}
    heap.strings = make(map[HeapId]string, 100000)
    heap.classNames = make(map[HeapId]HeapId, 50000)
    heap.gcRoots = make([]HeapId, 0, 10000)
    heap.NumClasses = 0
    heap.classes = []*ClassDef{nil} // leave room for entry [0]

    heap.classesByName = make(map[string]*ClassDef, 50000)
    heap.classesByHid = make(map[HeapId]*ClassDef, 50000)

    heap.NumObjects = 0
    heap.objectMap = MakeObjectMap()

    // TODO size accurately
    heap.objectCids = make([]ClassId, 1, 10000000) // entry[0] not used
    heap.objectSizes = make([]uint32, 1, 10000000) // entry[0] not used

    if options.NeedRefs {
        heap.segReader = makeSegReader(heap)
    }

    heap.autoPrefixes = []string {
        "java.lang.",
        "java.util.",
        "java.util.concurrent." }

    // JType descriptors are indexed by the "basic type" tag
    // found in a CLASS_DUMP or PRIMITIVE_ARRAY_DUMP

    heap.jtypes = []*JType{
        nil,  // 0 unused
        nil,  // 1 unused
        &JType{"", true, heap.idSize, nil}, // object descriptor unnamed because it varies by actual type
        nil,  // 3 unused
        &JType{"[Z", false, 1, nil},
        &JType{"[C", false, 2, nil},
        &JType{"[F", false, 4, nil},
        &JType{"[D", false, 8, nil},
        &JType{"[B", false, 1, nil},
        &JType{"[S", false, 2, nil},
        &JType{"[I", false, 4, nil},
        &JType{"[J", false, 8, nil} }

    return heap
}

// Add a new class definition.  Takes the demangled name.
//
func (heap *Heap) addClass(name string, hid HeapId, superHid HeapId, fieldNames []string, 
                            fieldTypes []*JType, staticRefs []HeapId) *ClassDef {

    dname := Demangle(name)
    class := heap.classesByName[dname]
    if class != nil {
        log.Fatalf("Class named %s already defined\n", dname)
    }
    class = heap.classesByHid[hid]
    if class != nil {
        log.Fatalf("Class with HID %d already defined as %s\n", hid, class.Name)
    }

    heap.NumClasses += 1
    cid := heap.NumClasses

    fields := make([]*Field, len(fieldNames))
    offset := uint32(0)
    for i, name := range fieldNames {
        fields[i] = &Field{name, fieldTypes[i], offset}
        offset += fields[i].JType.Size
    }

    def := makeClassDef(heap, dname, ClassId(cid), hid, superHid, fields, staticRefs)
    heap.classes = append(heap.classes, def)
    heap.classesByName[dname] = def
    heap.classesByHid[hid] = def

    // Update the JTypes if we've found a primitive array type.

    if len(name) == 2 && name[0] == '[' {
        for _, jtype := range heap.jtypes {
            if jtype != nil && name == jtype.ArrayClass {
                // log.Printf("Found %s hid %d\n", name, hid)
                jtype.Class = def
            }
        }
    }

    // log.Printf("Created %v\n", def)
    return def
}

func (heap *Heap) AddInstance(class *ClassDef, size uint32) {
    heap.NumObjects++
    heap.objectCids = append(heap.objectCids, class.Cid)
    heap.objectSizes = append(heap.objectSizes, size)
}

// Return the ClassDef with the given name, or nil if none.
// Uses auto-prefix list to resolve unqualified names.

func (heap *Heap) ClassNamed(name string) *ClassDef {
    if strings.IndexRune(name, '.') == -1 {
        return heap.classesByName[name]
    }
    for _, prefix := range heap.autoPrefixes {
        class := heap.classesByName[prefix + name]
        if class != nil {
            return class
        }
    }
    return nil
}

// Return the ClassDef with the given cid, or nil if none.
//
func (heap *Heap) HidClass(hid HeapId) *ClassDef {
    return heap.classesByHid[hid]
}

// Return the ClassDef with the given heap id, or nil if none.
//
func (heap *Heap) CidClass(cid ClassId) *ClassDef {
    return heap.classes[cid]
}

// Return the ClassDef for a given object id
//
func (heap *Heap) OidClass(oid ObjectId) *ClassDef {
    return heap.classes[heap.objectCids[oid]]
}

// Return the size for a given object id
//
func (heap *Heap) OidSize(oid ObjectId) uint32 {
    return heap.objectSizes[oid]
}
