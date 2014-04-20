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
    "regexp"
    "strings"
)

const (
    RecordsPerGB = 10000000 // rough estimate based on observations; TODO wider use
)

// A native ID read from the heap dump
//
type Hid uint64

// 1-based, assigned as we read instances from the dump
//
type Oid uint32

// Information read from a binary heap dump.
// TODO: consider a unit test that doesn't involve a real heap.
//
type Heap struct {
    // native identifier size
    IdSize uint32
    // gc root data
    GCRoots
    // static strings from UTF8 records
    strings map[Hid]string
    // highest class id assigned, 1-based
    MaxClassId uint32
    // highest heap ID encountered
    maxHid Hid
    // highest heap offset encountered
    maxOffset uint64
    // class defs indexed by cid
    classes []*Class
    // maps Hid of a class to Hid of its name; we have to do this because
    // LOAD_CLASS and CLASS_DUMP are different records.
    classNames map[Hid]Hid
    // same, indexed by demangled class name
    classesByName map[string]*Class
    // same, by native heap id
    classesByHid map[Hid]*Class
    // highect object ID assigned, 1-based
    MaxOid Oid
    // object cids, indexed by synthetic object id
    objectCids []ClassId
    // object sizes, indexed by same
    objectSizes []uint32
    // temporary mapping from Hids to Oids
    objectMap *ObjectMap
    // maps java value type tags to JType objects
    Jtypes []*JType
    // packages to search for unqualified class names
    autoPrefixes []string
    // classes to skip during graph searches
    skipNames []string
    // object graph
    graph *OidGraph
}

func NewHeap(idSize uint32) *Heap {

    heap := &Heap{

        IdSize: idSize,
        strings: make(map[Hid]string, 100000),           // good enough

        MaxClassId: 0,
        maxHid: 0,
        maxOffset: 0,

        classes: []*Class{nil},                          // leave room for entry [0]
        classNames: make(map[Hid]Hid, 50000),         // handles most heaps
        classesByName: make(map[string]*Class, 50000),   // good enough
        classesByHid: make(map[Hid]*Class, 50000),

        MaxOid: 0,
        // TODO size accurately
        objectCids: make([]ClassId, 1, 10000000),           // entry[0] not used
        objectSizes: make([]uint32, 1, 10000000),           // entry[0] not used
        objectMap: &ObjectMap{},

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

        autoPrefixes: []string {
            "java.lang.",
            "java.util.",
            "java.util.concurrent.",
        },

        skipNames: nil,
        graph: nil,
    }

    root := heap.AddClass("root", 1, 1, []string{}, []*JType{}, []Hid{})
    heap.AddInstance(1, root, 0, 0)

    return heap
}

// Add a UTF8 string record (class names, fields etc; not user-defined strings.)
//
func (heap *Heap) AddString(hid Hid, str string) {
    heap.strings[hid] = str
    // log.Printf("%x -> %s\n", hid, heap.strings[hid])
}

// Return a string record added with AddString, or "" if none.  (None of the
// strings that matter are empty.)
//
func (heap *Heap) StringWithId(hid Hid) string {
    name, ok := heap.strings[hid]
    if ok {
        return name
    }
    return ""
}

// Add a class ID / name string ID binding.
//
func (heap *Heap) AddClassName (classHid Hid, nameHid Hid) {
    heap.classNames[classHid] = nameHid
    // log.Printf("%x -> %x -> %s\n", classHid, nameHid, heap.strings[nameHid])
}

// Return the name ID for a class ID added with AddClassName, or zero if none.
//
func (heap *Heap) ClassNameId(hid Hid) Hid {
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
func (heap *Heap) AddClass(name string, hid Hid, superHid Hid, fieldNames []string, 
                            fieldTypes []*JType, staticRefs []Hid) *Class {

    if hid > heap.maxHid {
        heap.maxHid = hid
    }

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

    class = NewClass(heap, dname, ClassId(cid), hid, superHid, fields, staticRefs)
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

// Note a class instance, incrementing MaxOid and binding it to its class
// definition.  Does not record anything about the instance data.
//
func (heap *Heap) AddInstance(hid Hid, class *Class, size uint32, offset uint64) Oid {

    if hid > heap.maxHid {
        heap.maxHid = hid
    }
    if offset > heap.maxOffset {
        heap.maxOffset = offset
    }

    heap.MaxOid++
    heap.objectCids = append(heap.objectCids, class.Cid)
    heap.objectSizes = append(heap.objectSizes, size)
    heap.objectMap.Add(hid, heap.MaxOid)

    return heap.MaxOid
}

// Return the Class with the given name, or nil if none.
// Uses auto-prefix list to resolve unqualified names.
//
func (heap *Heap) ClassNamed(name string) *Class {
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
func (heap *Heap) PostProcess(sr *SegReader) { // TODO > 1 SegReader ??

    /*
        val fakes = new mutable.HashMap[Long,(Int,Long)]
        val superHid = classes.getByName("java.lang.Class").heapId
        val noFields = new Array[Java.Type](0)
        val noNames = new Array[String](0)
        
        for (i <- 0 until staticRefs.size by 2) {
            val fromClass = staticRefs.get(i)
            val toObject = staticRefs.get(i+1)
            val (fakeId, fakeHid) = fakes.getOrElseUpdate(fromClass, {
                val classDef = classes.getByHid(fromClass)
                val fakeName = classDef.name + ".class"
                val fakeDef = addClass(fakeName, fabricateHid(), superHid, noFields, noNames)
                val fakeHid = fabricateHid()
                val fakeId = addInstance(fakeHid, fakeDef.heapId, fabricateOffset(), 0)
                (fakeId, fakeHid)
            })
            addReference(fakeId, toObject)
            addGCRoot(fakeHid, "loaded class")
        }
    */

    // Fabricate a class object to hold static references for each class.  This
    // is done after the heap read is complete so we can guarantee unique HIDs.

    jlo := heap.ClassNamed("java.lang.Object")
    bag := &References{}

    for _, class := range heap.classes[1:] {

        fakeName := class.Name + ".class"
        fakeClassHid := heap.fabricateHid()
        fakeInstanceHid := heap.fabricateHid()
        fakeOffset := heap.fabricateOffset()
        fakeClass := heap.AddClass(fakeName, fakeClassHid, jlo.Hid, []string{}, []*JType{}, []Hid{})
        fakeOid := heap.AddInstance(fakeInstanceHid, fakeClass, 0, fakeOffset)
        // log.Printf("Assign fake oid %d hid %x for %s\n", fakeOid, fakeInstanceHid, fakeName)

        // Make each class a GC root and link its static references

        heap.AddRoot(fakeInstanceHid)
        for _, hid := range class.StaticRefs {
            if hid != 0 {
                bag.Add(fakeOid, hid)
            }
        }
    }

    heap.objectMap.PostProcess()
    heap.LinkMasterRoot(bag)

    if sr != nil {
        bags := sr.close()
        bags = append(bags, bag)
        from, to := MergeReferences(bags, func(hid Hid) Oid {return heap.objectMap.Get(hid)})
        log.Printf("%d references\n", len(from))
        // TODO: add static references to graph
        heap.graph = NewOidGraph(from, to)
    }

    resolver := func(hid Hid) Oid { return heap.objectMap.Get(hid) }
    heap.FindLiveObjects(heap.graph, resolver, heap.MaxOid)
    heap.objectMap = nil // allow GC

    for _, def := range heap.classes[1:] {
        def.Cook()
    }

}

// Return the Class with the given cid, or nil if none.
//
func (heap *Heap) HidClass(hid Hid) *Class {
    return heap.classesByHid[hid]
}

// Return the Class for a given object id
//
func (heap *Heap) ClassOf(oid Oid) *Class {
    cid := heap.objectCids[oid]
    class := heap.classes[cid]
    if class == nil {
        panic(fmt.Sprintf("oid %d cid %d has no class def", oid, cid))
    }
    return class
}

// Return the size for a given object id
//
func (heap *Heap) SizeOf(oid Oid) uint32 {
    return heap.objectSizes[oid]
}

// Add a (possibly wildcard) class name to the list of classes to be skipped
// (or not) during graph searches.
//
func (heap *Heap) DoSkip(name string, skip bool) {

    if name == "none" {
        if skip {
            // "skip none" clears skip state
            heap.skipNames = nil
            for _, class := range heap.classes[1:] {
                class.Skip = false
            }
            return
        } else {
            // turn "noskip none" into "skip all"
            name = "all"
            skip = true
        }
    }

    numMatching := 0

    if name == "all" {
        for _, class := range heap.classes[1:] {
            class.Skip = skip
            numMatching += 1
        }
    } else {
        for _, name := range heap.skipNames {
            heap.WithClassesMatching(name, func(class *Class) {
                class.Skip = skip
                numMatching += 1
            })
        }
    }

    if numMatching == 0 {
        fmt.Errorf("No classes match %s\n", name)
    } else if skip {
        heap.skipNames = append(heap.skipNames, "+ " + name)
    } else {
        heap.skipNames = append(heap.skipNames, "- " + name)
    }
}

// Return a bitset with class IDs of classes matching a type wildcard turned on.
//
func (heap *Heap) CidsMatching(name string) BitSet {
    bits := MakeBitSet(heap.MaxClassId + 1)
    heap.WithClassesMatching(name, func(class *Class) {
        addSubclassCids(class, bits)
    })
    return bits
}

func addSubclassCids(class *Class, bits BitSet) {
    bits.Set(uint32(class.Cid))
    for _, subclass := range class.subclasses {
        addSubclassCids(subclass, bits)
    }
}

// Execute a function for each class matching a type wildcard.
//
func (heap *Heap) WithClassesMatching(name string, f func(*Class)) {
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

// New wildcard version
//
func (heap *Heap) MatchClasses(bits BitSet, name string, include bool) bool {

    matched := false

    // replace [] with \[\] so we can use it as a pattern

    name = strings.Replace(name, "[]", "\\[\\]", -1)
    re, err := regexp.Compile(name)
    if err != nil {
        // TODO fix
        log.Fatal("Can't make RE from %s: %s\n", name, err)
    }

    for _, class := range heap.classesByName {
        if re.MatchString(class.Name) {
            markClass(class, bits, include)
            matched = true
        }
    }

    return matched
}

func markClass(class *Class, bits BitSet, include bool) {
    if (include) {
        bits.Set(uint32(class.Cid))
    } else {
        bits.Clear(uint32(class.Cid))
    }
    for _, c := range class.subclasses {
        markClass(c, bits, include)
    }
}

// Return a fabricated heap ID higher than the highest one already seen.  This
// is used for building placeholder objects and classes.
//
func (heap *Heap) fabricateHid() Hid {
    heap.maxHid += Hid(heap.IdSize)
    return heap.maxHid
}

// Same idea as fabricateHid
//
func (heap *Heap) fabricateOffset() uint64 {
    heap.maxOffset += uint64(heap.IdSize)
    return heap.maxOffset
}

