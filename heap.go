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

// Add a new class definition.  Takes the demangled name.
//
func (heap *Heap) addClass(name string, hid HeapId, superHid HeapId, 
                            fieldNames []string, fieldTypes []*JType) *ClassDef {

    class := heap.classesByName[name]
    if class != nil {
        log.Fatalf("Class named %s already defined\n", name)
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

    def := makeClassDef(heap, name, ClassId(cid), hid, superHid, fields)
    heap.classes = append(heap.classes, def)
    heap.classesByName[name] = def
    heap.classesByHid[hid] = def

    // log.Printf("Created %v\n", def)
    return def
}

// Return the next available object ID and associate it with the given class def.
//
func (heap *Heap) nextOid(hid HeapId, def *ClassDef) ObjectId {
    heap.NumObjects += 1
    oid := ObjectId(heap.NumObjects)
    // heap.objectMap.add(hid, oid)
    heap.objectCids = append(heap.objectCids, def.Cid)
    return oid
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

