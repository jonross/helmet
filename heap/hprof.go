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

// This package understands HPROF binary dump format.
//
package heap

import (
    "log"
    "github.com/jonross/helmet/util"
)

// An ID read from the heap dump
//
type HeapId uint64

// 1-based, assigned as we read class definitions from the dump
//
type ClassId uint32

// 1-based, assigned as we read instances from the dump
//
type ObjectId uint32

// Information about java value types
//
type JType struct {
    // JVM short class name for an array of this type
    arrayClass string
    // true if this is an object type, not a primitive type
    isObj bool
    // size in bytes
    size uint32
    // assigned when found in heap
    hid HeapId
}

type Heap struct {
    // Yes that
    filename string
    // And that
    mappedFile *util.MappedFile
    // Size of a native ID on the heap, 4 or 8
    idSize uint32
    // true if idSize is 8
    longIds bool
    // static strings from UTF8 records
    strings map[HeapId]string
    // maps HeapId of a class to HeapId of its name; we have to do this because
    // LOAD_CLASS and ... are different records.
    classNames map[HeapId]HeapId
    // how many class IDs have we assigned
    numClassIds uint32
    // how many object IDs have we assigned
    numObjectIds uint32
    // heap IDs of GC roots
    gcRoots []HeapId
    // maps java value type tags to JType objects
    jtypes []*JType
}

func ReadHeap(filename string) (heap *Heap, err error) {

    heap = &Heap{filename: filename}
    heap.mappedFile, err = util.MapFile(filename)
    if err != nil {
        return nil, err
    }
    defer heap.mappedFile.Close()

    in := heap.mappedFile.MapAt(0)
    version := string(in.GetRaw(19))
    if version != "JAVA PROFILE 1.0.1\x00" && version != "JAVA PROFILE 1.0.2\x00" {
        log.Fatalf("Unknown heap version %s\n", version)
    }

    heap.idSize = in.GetUInt32()
    if heap.idSize != 4 && heap.idSize != 8 {
        log.Fatalf("Unknown reference size %d\n", heap.idSize)
    }
    heap.longIds = heap.idSize == 8

    // skip timestamp
    in.Skip(8)

    heap.strings = make(map[HeapId]string)
    heap.classNames = make(map[HeapId]HeapId)
    heap.gcRoots = []HeapId{}
    headerSize := uint32(9)

    // JType descriptors are indexed by the "basic type" tag
    // found in a CLASS_DUMP or PRIMITIVE_ARRAY_DUMP

    heap.jtypes = []*JType{
        nil,  // 0 unused
        nil,  // 1 unused
        // Note object type descriptor is unnamed because varies by actual type
        &JType{"", true, heap.idSize, 0},
        nil,  // 3 unused
        &JType{"[Z", false, 1, 0},
        &JType{"[C", false, 2, 0},
        &JType{"[F", false, 4, 0},
        &JType{"[D", false, 8, 0},
        &JType{"[B", false, 1, 0},
        &JType{"[S", false, 2, 0},
        &JType{"[I", false, 4, 0},
        &JType{"[J", false, 8, 0} }

    for in.Demand(headerSize) != nil {

        tag := in.GetByte()
        in.Skip(4) // Skip timestamp
        length := in.GetUInt32()
        // log.Printf("Record type %d len %d at %d\n", tag, length, in.Offset() - uint64(headerSize))

        // A function table would be more efficient but there aren't
        // that many top-level records compared to instance records.

        switch tag {
            case 0x01: // UTF8
                hid := heap.readId(in)
                heap.strings[hid] = in.GetString(length - heap.idSize)
                // log.Printf("%x -> %s\n", hid, heap.strings[hid])

            case 0x02: // LOAD_CLASS
                in.Skip(4) // skip classSerial
                classHid := heap.readId(in)
                in.Skip(4) // skip stackSerial
                nameHid := heap.readId(in)
                heap.classNames[classHid] = nameHid
                // log.Printf("%x -> %x -> %s\n", classHid, nameHid, heap.strings[nameHid])

            case 0x0c, 0x1c: // HEAP_DUMP, HEAP_DUMP_SEGMENT
                log.Printf("Heap dump or segment of %d MB", length / 1048576)
                heap.readSegment(in, length)

            case 0x03: // UNLOAD_CLASS
                fallthrough
            case 0x04: // STACK_FRAME
                fallthrough
            case 0x05: // STACK_TRACE
                fallthrough
            case 0x06: // ALLOC_SITES
                fallthrough
            case 0x07: // HEAP_SUMMARY
                fallthrough
            case 0x0a: // START_THREAD
                fallthrough
            case 0x0b: // END_THREAD
                fallthrough
            case 0x0e: // CONTROL_SETTINGS
                fallthrough
            case 0x2c: // HEAP_DUMP_END
                in.Skip(length)

            default:
                log.Fatalf("Unknown HPROF record type %d at %d\n", tag, in.Offset() - uint64(headerSize))
        }
    }

    return
}

func (heap *Heap) readSegment(in *util.MappedSection, length uint32) {
    end := in.Offset() + uint64(length)
    for in.Offset() < end {
        tag := in.GetByte()
        // log.Printf("tag %d\n", tag)
        switch tag {
            case 0x21: // INSTANCE_DUMP
                heap.readInstance(in)
            case 0x22: // OBJECT_ARRAY
                heap.readArray(in, true)
            case 0x23: // PRIMITIVE_ARRAY
                heap.readArray(in, false)
            case 0x20: // CLASS_DUMP
                heap.readClassDump(in)
            case 0x01: // ROOT_JNI_GLOBAL
                heap.readGCRoot(in, "JNI global", heap.idSize)
            case 0x02: // ROOT_JNI_LOCAL
                heap.readGCRoot(in, "JNI local", 8)
            case 0x03: // ROOT_JAVA_FRAME
                heap.readGCRoot(in, "java frame", 8)
            case 0x04: // ROOT_NATIVE_STACK
                heap.readGCRoot(in, "native stack", 4)
            case 0x05: // ROOT_STICKY_CLASS
                heap.readGCRoot(in, "sticky class", 0)
            case 0x06: // ROOT_THREAD_BLOCK
                heap.readGCRoot(in, "thread block", 4)
            case 0x07: // ROOT_MONITOR_USED
                heap.readGCRoot(in, "monitor used", 0)
            case 0x08: // ROOT_THREAD_OBJECT
                heap.readGCRoot(in, "thread object", 8)
            case 0xff: // ROOT_UNKNOWN
                heap.readGCRoot(in, "unknown root", 0)
            default:
                log.Fatalf("Unknown HPROF record type %d at %d\n", tag, in.Offset() - 1)
        }
    }
}

// Read a GC root.  This has the HID at the start followed by some amount
// of per-root data that we don't use.
//
func (heap *Heap) readGCRoot(in *util.MappedSection, kind string, skip uint32) {
    hid := heap.readId(in)
    heap.gcRoots = append(heap.gcRoots, hid)
    in.Skip(skip)
}

func (heap *Heap) readInstance(in *util.MappedSection) {

    // TODO demand

    heap.readId(in) // hid
    in.Skip(4) // stack serial
    heap.readId(in) // class hid
    length := in.GetUInt32()
    in.Skip(length)
}

func (heap *Heap) readArray(in *util.MappedSection, isObjects bool) {

    // TODO demand

    heap.readId(in) // hid
    in.Skip(4) // stack serial
    count := in.GetUInt32()

    if isObjects {
        heap.readId(in) // array class hid
        in.Skip(count * heap.idSize)
    } else {
        jtype := heap.readJType(in)
        in.Skip(count * jtype.size)
    }
}

func (heap *Heap) readClassDump(in *util.MappedSection) {

    in.Demand(7 * heap.idSize + 8)
    heap.readId(in) // hid
    in.Skip(4) // stack serial
    heap.readId(in) // superHid
    in.Skip(5 * heap.idSize) // skip class loader ID, signer ID, protection domain ID, 2 reserved
    in.Skip(4) // instance size

    // Skip over constant pool

    in.Demand(2)
    numConstants := in.GetUInt16()
    in.Demand(11 * uint32(numConstants))

    for i := 0; i < int(numConstants); i++ {
        in.Skip(2)
        jtype := heap.readJType(in)
        in.Skip(jtype.size)
    }

    // Static fields

    in.Demand(2)
    numStatics := in.GetUInt16()
    in.Demand(11 * uint32(numStatics))

    for i := 0; i < int(numStatics); i++ {
        in.Skip(heap.idSize) // field name ID
        jtype := heap.readJType(in)
        if jtype.isObj {
            heap.readId(in)
            // if (toHid != 0)
            //     heap.addStaticReference(classId, toId)
        } else {
            in.Skip(jtype.size)
        }
    }

    // Instance fields

    in.Demand(2)
    numFields := in.GetUInt16()
    fieldNameIds := make([]HeapId, numFields, numFields)
    fieldTypes := make([]*JType, numFields, numFields)

    for i := 0; i < int(numFields); i++ {
        fieldNameIds[i] = heap.readId(in)
        fieldTypes[i] = heap.readJType(in)
    }

    // heap.addClassDef(classId, superclassId, fieldInfo, fieldNameIds);
}

// Read a native ID from heap data.
//
func (heap *Heap) readId(in *util.MappedSection) HeapId {
    if (heap.longIds) {
        return HeapId(in.GetUInt64())
    }
    return HeapId(in.GetUInt32())
}

// Read a "Basic Type" ID from heap data and return the JType
//
func (heap *Heap) readJType(in *util.MappedSection) *JType {
    tag := int(in.GetByte())
    if tag < 0 || tag >= len(heap.jtypes) {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    jtype := heap.jtypes[tag]
    if jtype == nil {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    return jtype
}


