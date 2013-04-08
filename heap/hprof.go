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
    "os"
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
    filename string
    file *os.File
    bytes []byte
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

    heap.file, err = os.Open(heap.filename)
    if err != nil {
        return nil, err
    }
    defer heap.file.Close()

    heap.bytes, err = util.MMap(heap.file, 0)
    if err != nil {
        return nil, err
    }
    defer util.MUnmap(heap.bytes)

    heap.read(heap.bytes)
    return
}

func (heap *Heap) read(in []byte) {

    version := string(in[:19])
    if version != "JAVA PROFILE 1.0.1\x00" && version != "JAVA PROFILE 1.0.2\x00" {
        log.Fatalf("Unknown heap version %s\n", version)
    }

    heap.idSize = util.GetUInt32(in[19:23])
    if heap.idSize != 4 && heap.idSize != 8 {
        log.Fatalf("Unknown reference size %d\n", heap.idSize)
    }
    heap.longIds = heap.idSize == 8

    // skip timestamp
    in = in[31:]

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

    for len(in) > 0 {

        tag := in[0]
        // Skip timestamp
        length := util.GetUInt32(in[headerSize-4:headerSize])

        // A function table would be more efficient but there aren't
        // that many top-level records compared to instance records.

        switch tag {
            case 0x01: // UTF8
                xin := in[headerSize:headerSize+length]
                hid := heap.readId(xin)
                heap.strings[hid] = string(xin[heap.idSize:])

            case 0x02: // LOAD_CLASS
                // skip classSerial
                xin := in[headerSize+4:headerSize+length]
                classHid := heap.readId(xin)
                // skip stackSerial
                xin = xin[heap.idSize+4:]
                nameHid := heap.readId(xin)
                heap.classNames[classHid] = nameHid
                // log.Printf("%x -> %x -> %s\n", classHid, nameHid, heap.strings[nameHid])

            case 0x0c, 0x1c: // HEAP_DUMP, HEAP_DUMP_SEGMENT
                log.Printf("Heap dump or segment of %d MB", length / 1048576)
                heap.readSegment(in[headerSize:headerSize+length])

            case 0x03: // UNLOAD_CLASS
                break
            case 0x04: // STACK_FRAME
                break
            case 0x05: // STACK_TRACE
                break
            case 0x06: // ALLOC_SITES
                break
            case 0x07: // HEAP_SUMMARY
                break
            case 0x0a: // START_THREAD
                break
            case 0x0b: // END_THREAD
                break
            case 0x0e: // CONTROL_SETTINGS
                break
            case 0x2c: // HEAP_DUMP_END
                break

            default:
                log.Fatalf("Unknown HPROF record type: %d\n", tag)
        }

        in = in[headerSize+length:]
    }
}

// Handle a HEAP_DUMP or HEAP_DUMP_SEGMENT record
//
func (heap *Heap) readSegment(in []byte) {
    for len(in) > 0 {
        tag := in[0]
        switch tag {
            case 0x21: // INSTANCE_DUMP
                in = heap.readInstance(in)
            case 0x22: // OBJECT_ARRAY
                in = heap.readArray(in, true)
            case 0x23: // PRIMITIVE_ARRAY
                in = heap.readArray(in, false)
            case 0x20: // CLASS_DUMP
                in = heap.readClassDump(in)
            case 0x01: // ROOT_JNI_GLOBAL
                in = heap.readGCRoot(in, "JNI global", heap.idSize)
            case 0x02: // ROOT_JNI_LOCAL
                in = heap.readGCRoot(in, "JNI local", 8)
            case 0x03: // ROOT_JAVA_FRAME
                in = heap.readGCRoot(in, "java frame", 8)
            case 0x04: // ROOT_NATIVE_STACK
                in = heap.readGCRoot(in, "native stack", 4)
            case 0x05: // ROOT_STICKY_CLASS
                in = heap.readGCRoot(in, "sticky class", 0)
            case 0x06: // ROOT_THREAD_BLOCK
                in = heap.readGCRoot(in, "thread block", 4)
            case 0x07: // ROOT_MONITOR_USED
                in = heap.readGCRoot(in, "monitor used", 0)
            case 0x08: // ROOT_THREAD_OBJECT
                in = heap.readGCRoot(in, "thread object", 8)
            case 0xff: // ROOT_UNKNOWN
                in = heap.readGCRoot(in, "unknown root", 0)
            default:
                log.Fatalf("Unknown HPROF record type: %d\n", tag)
        }
    }
}

// Read a GC root.  This has the HID at the start followed by some amount
// of per-root data that we don't use.
//
func (heap *Heap) readGCRoot(in []byte, kind string, skip uint32) []byte {
    hid := heap.readId(in)
    heap.gcRoots = append(heap.gcRoots, hid)
    return in[heap.idSize + skip:]
}

func (heap *Heap) readInstance(in []byte) []byte {
    log.Fatal("readInstance not defined\n")
    return in
}

func (heap *Heap) readArray(in []byte, isObjects bool) []byte {
    log.Fatal("readArray not defined")
    return in
}

func (heap *Heap) readClassDump(in []byte) []byte {
    log.Fatal("readClassDump not defined")
    return in
}

// Read a native ID from heap data.
//
func (heap *Heap) readId(in []byte) HeapId {
    if (heap.longIds) {
        return HeapId(util.GetUInt64(in))
    }
    return HeapId(util.GetUInt32(in))
}

