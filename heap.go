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
    "fmt"
    "log"
    "strings"
)

const (
    RecordsPerGB = 10000000 // rough estimate based on observations
)

// A native ID read from the heap dump
//
type HeapId uint64

// 1-based, assigned as we read instances from the dump
//
type ObjectId uint32

// Information read from a binary heap dump.
// TODO: consider a unit test that doesn't involve a real heap.
//
type Heap struct {
    // native identifier size
    IdSize uint32
    // static strings from UTF8 records
    strings map[HeapId]string
    // heap IDs of GC roots
    gcRoots []HeapId
    // highest class id assigned, 1-based
    MaxClassId uint32
    // class defs indexed by cid
    classes []*ClassDef
    // maps HeapId of a class to HeapId of its name; we have to do this because
    // LOAD_CLASS and CLASS_DUMP are different records.
    classNames map[HeapId]HeapId
    // same, indexed by demangled class name
    classesByName map[string]*ClassDef
    // same, by native heap id
    classesByHid map[HeapId]*ClassDef
    // highect object ID assigned, 1-based
    MaxObjectId ObjectId
    // object cids, indexed by synthetic object id
    objectCids []ClassId
    // object sizes, indexed by same
    objectSizes []uint32
    // temporary mapping from HeapIds to ObjectIds
    objectMap *ObjectMap
    // maps java value type tags to JType objects
    Jtypes []*JType
    // packages to search for unqualified class names
    autoPrefixes []string
    // classes to skip during graph searches
    skipNames []string
    // skip ID of objects with classes matched by skipNames
    skipIds []int
    // object graph
    *Graph
}

func NewHeap(idSize uint32) *Heap {
    return &Heap{

        IdSize: idSize,
        strings: make(map[HeapId]string, 100000),           // good enough
        gcRoots: make([]HeapId, 0, 10000),                  // good enough

        MaxClassId: 0,
        classes: []*ClassDef{nil},                          // leave room for entry [0]
        classNames: make(map[HeapId]HeapId, 50000),         // handles most heaps
        classesByName: make(map[string]*ClassDef, 50000),   // good enough
        classesByHid: make(map[HeapId]*ClassDef, 50000),

        MaxObjectId: 0,
        // TODO size accurately
        objectCids: make([]ClassId, 1, 10000000),           // entry[0] not used
        objectSizes: make([]uint32, 1, 10000000),           // entry[0] not used
        objectMap: &ObjectMap{},

        autoPrefixes: []string {
            "java.lang.",
            "java.util.",
            "java.util.concurrent.",
        },

        // Indexed by the "basic type" tag found in a CLASS_DUMP or PRIMITIVE_ARRAY_DUMP
        Jtypes: []*JType{
            nil,  // 0 unused
            nil,  // 1 unused
            &JType{"", true, idSize, nil}, // object descriptor unnamed because it varies by actual type
            nil,  // 3 unused
            &JType{"[Z", false, 1, nil},
            &JType{"[C", false, 2, nil},
            &JType{"[F", false, 4, nil},
            &JType{"[D", false, 8, nil},
            &JType{"[B", false, 1, nil},
            &JType{"[S", false, 2, nil},
            &JType{"[I", false, 4, nil},
            &JType{"[J", false, 8, nil},
        },

        skipNames: nil,
        skipIds: nil,
        Graph: nil,
    }
}

// Add a UTF8 string record (class names, fields etc; not user-defined strings.)
//
func (heap *Heap) AddString(hid HeapId, str string) {
    heap.strings[hid] = str
    // log.Printf("%x -> %s\n", hid, heap.strings[hid])
}

// Return a string record added with AddString, or "" if none.  (None of the
// strings that matter are empty.)
//
func (heap *Heap) StringWithId(hid HeapId) string {
    name, ok := heap.strings[hid]
    if ok {
        return name
    }
    return ""
}

// Add a class ID / name string ID binding.
//
func (heap *Heap) AddClassName (classHid HeapId, nameHid HeapId) {
    heap.classNames[classHid] = nameHid
    // log.Printf("%x -> %x -> %s\n", classHid, nameHid, heap.strings[nameHid])
}

// Return the name ID for a class ID added with AddClassName, or zero if none.
//
func (heap *Heap) ClassNameId(hid HeapId) HeapId {
    nameId, ok := heap.classNames[hid]
    if ok {
        return nameId
    }
    return 0
}

// Add a new class definition and increment MaxClassId.  The name is as read from 
// the heap but we Demangle it for indexing.  Also updates the Jtypes class
// definitions if we've discovered one of the predefined primitive array types.
//
func (heap *Heap) AddClass(name string, hid HeapId, superHid HeapId, fieldNames []string, 
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

    heap.MaxClassId += 1
    cid := heap.MaxClassId

    // Create descriptors for each field.

    fields := make([]*Field, len(fieldNames))
    offset := uint32(0)
    for i, name := range fieldNames {
        fields[i] = &Field{name, fieldTypes[i], offset}
        offset += fields[i].JType.Size
    }

    class = NewClassDef(heap, dname, ClassId(cid), hid, superHid, fields, staticRefs)
    heap.classes = append(heap.classes, class)
    heap.classesByName[dname] = class
    heap.classesByHid[hid] = class

    // Update the JTypes if we've found a primitive array type.

    if len(name) == 2 && name[0] == '[' {
        for _, jtype := range heap.Jtypes {
            if jtype != nil && name == jtype.ArrayClass {
                // log.Printf("Found %s hid %d\n", name, hid)
                jtype.Class = class
            }
        }
    }

    // log.Printf("Created %v\n", class)
    return class
}

// Note a class instance, incrementing MaxObjectId and binding it to its class
// definition.  Does not record anything about the instance data.
//
func (heap *Heap) AddInstance(hid HeapId, class *ClassDef, size uint32) ObjectId {
    heap.MaxObjectId++
    heap.objectCids = append(heap.objectCids, class.Cid)
    heap.objectSizes = append(heap.objectSizes, size)
    heap.objectMap.Add(hid, heap.MaxObjectId)
    return heap.MaxObjectId
}

// Return the ClassDef with the given name, or nil if none.
// Uses auto-prefix list to resolve unqualified names.
//
func (heap *Heap) ClassNamed(name string) *ClassDef {
    if strings.IndexRune(name, '.') >= 0 {
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

// Post-process the heap by incorporating references scanned by the concurrent
// segment readers, and resolve heap IDs to synthetic object IDs.
//
func (heap *Heap) PostProcess(sr *SegReader) {

    heap.objectMap.PostProcess()

    if sr != nil {
        bags := sr.close()
        from, to := MergeBags(bags, func(hid HeapId) ObjectId {return heap.objectMap.Get(hid)})
        log.Printf("%d references\n", len(from))
        // TODO: add static references to graph
        heap.Graph = NewGraph(from, to)
        bags = nil // allow gc
    }

    heap.objectMap = nil // allow GC

    for _, def := range heap.classes[1:] {
        def.Cook()
    }

}

// Return the ClassDef with the given cid, or nil if none.
//
func (heap *Heap) HidClass(hid HeapId) *ClassDef {
    return heap.classesByHid[hid]
}

// Return the ClassDef for a given object id
//
func (heap *Heap) ClassOf(oid ObjectId) *ClassDef {
    cid := heap.objectCids[oid]
    class := heap.classes[cid]
    if class == nil {
        panic(fmt.Sprintf("oid %d cid %d has no class def", oid, cid))
    }
    return class
}

// Return the size for a given object id
//
func (heap *Heap) SizeOf(oid ObjectId) uint32 {
    return heap.objectSizes[oid]
}

// Add a (possibly wildcard) class name to the list of classes to be skipped
// during graph searches.  Must then call ProcessSkips before the next search.
//
func (heap *Heap) AddSkip(name string) {
    heap.skipNames = append(heap.skipNames, name)
    heap.skipIds = nil
}

// Return a bitset with class IDs of classes matching a type wildcard turned on.
//
func (heap *Heap) CidsMatching(name string) BitSet {
    bits := NewBitSet(heap.MaxClassId + 1)
    heap.WithClassesMatching(name, func(class *ClassDef) {
        addSubclassCids(class, bits)
    })
    return bits
}

func addSubclassCids(class *ClassDef, bits BitSet) {
    bits.Set(uint32(class.Cid))
    for _, subclass := range class.subclasses {
        addSubclassCids(subclass, bits)
    }
}

// Execute a function for each class matching a type wildcard.
//
func (heap *Heap) WithClassesMatching(name string, f func(*ClassDef)) {
    isWild := strings.HasSuffix(name, "*") // TODO: do better, maybe regexp
    if isWild {
        name = name[:len(name)-1]
        for _, class := range heap.classes[1:] {
            if strings.HasPrefix(class.Name, name) {
                f(class)
            }
        }
    } else {
        class := heap.ClassNamed(name)
        if class != nil {
            f(class)
            return
        }
        for _, prefix := range heap.autoPrefixes {
            class := heap.ClassNamed(prefix + name)
            if class != nil {
                f(class)
                return
            }
        }
    }
}

// Assign a unique ID to every object whose class is skipped.  This allows the
// graph search to maintain a piece of information for skipped objects only
// rather than also having an empty slot for non-skipped objects.
//
func (heap *Heap) ProcessSkips() {

    for _, class := range heap.classes[1:] {
        class.Skip = false
    }

    numSkips := uint32(0)
    for _, name := range heap.skipNames {
        heap.WithClassesMatching(name, func(class *ClassDef) {
            class.Skip = true
            numSkips += class.NumInstances
        })
    }

    if heap.skipIds == nil {
        heap.skipIds = make([]int, heap.MaxObjectId + 1)
    }

    skipId := 1
    for oid := ObjectId(1); oid <= heap.MaxObjectId; oid++ {
        if heap.ClassOf(oid).Skip {
            heap.skipIds[oid] = skipId
            skipId++
        } else {
            heap.skipIds[oid] = 0
        }
    }
}

func (heap *Heap) SkipIdOf(oid ObjectId) int {
    return heap.skipIds[oid]
}

