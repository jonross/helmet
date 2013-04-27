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

type HProfReader struct {
    // Yes that
    Filename string
    // And that
    *MappedFile
    // Size of a native ID on the heap, 4 or 8
    idSize uint32
    // true if idSize is 8
    longIds bool
    // target heap tracker
    *Heap
    // track references?
    needRefs bool
}

func ReadHeapDump(filename string, options *HeapOptions) *Heap {

    mappedFile, err := MapFile(filename)
    if err != nil {
        log.Fatal("Can't map %s: %s\n", filename, err)
    }
    defer mappedFile.Close()

    // Verify this is a real HPROF file & determine native ID size.

    in := mappedFile.MapAt(0)
    version := string(in.GetRaw(18))
    in.Skip(1) // trailing NUL

    if version != "JAVA PROFILE 1.0.1" && version != "JAVA PROFILE 1.0.2" {
        log.Fatalf("Unknown heap version %s\n", version)
    }

    hprof := &HProfReader{
        Filename: filename,
        MappedFile: mappedFile,
        needRefs: options.NeedRefs,
    }

    hprof.idSize = in.GetUInt32()
    if hprof.idSize != 4 && hprof.idSize != 8 {
        log.Fatalf("Unknown reference size %d\n", hprof.idSize)
    }
    hprof.longIds = hprof.idSize == 8
    in.Skip(8) // skip timestamp

    return hprof.read(in, options)
}

// Complete heap reader in one call.  Most of the error conditions (like
// unresolvable classes) cause panics BTW.
//
func (hprof *HProfReader) read(in *MappedSection, options *HeapOptions) *Heap {

    hprof.Heap = NewHeap(hprof, options)
    heap := hprof.Heap

    // TODO: keep input struct constant, don't return different one

    headerSize := uint32(9)
    numRecords := 0
    numStrings := 0

    for in.Demand(headerSize) != nil {

        numRecords++
        tag := in.GetByte()
        in.Skip(4) // Skip timestamp
        length := in.GetUInt32()
        // log.Printf("Record type %d len %d at %d\n", tag, length, in.Offset() - uint64(headerSize))

        // A function table would be more efficient but there aren't
        // that many top-level records compared to instance records.

        switch tag {
            case 0x01: // UTF8
                numStrings++
                hid := heap.readId(in)
                str := in.GetString(length - heap.idSize)
                heap.strings[hid] = str
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
                numRecords += hprof.readSegment(in, length)

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

    heap.PostProcess()

    log.Printf("%d records, %d UTF8\n", numRecords, numStrings)
    log.Printf("%d objects\n", heap.MaxObjectId)

    return heap
}

// Handle HEAP_DUMP or HEAP_DUMP_SEGMENT record.
//
func (hprof *HProfReader) readSegment(in *MappedSection, length uint32) int {
    end := in.Offset() + uint64(length)
    numRecords := 0
    for in.Offset() < end {
        numRecords++
        tag := in.GetByte()
        // log.Printf("tag %d\n", tag)
        switch tag {
            case 0x21: // INSTANCE_DUMP
                hprof.readInstance(in)
            case 0x22: // OBJECT_ARRAY
                hprof.readArray(in, true)
            case 0x23: // PRIMITIVE_ARRAY
                hprof.readArray(in, false)
            case 0x20: // CLASS_DUMP
                hprof.readClassDump(in)
            case 0x01: // ROOT_JNI_GLOBAL
                hprof.readGCRoot(in, "JNI global", hprof.idSize)
            case 0x02: // ROOT_JNI_LOCAL
                hprof.readGCRoot(in, "JNI local", 8)
            case 0x03: // ROOT_JAVA_FRAME
                hprof.readGCRoot(in, "java frame", 8)
            case 0x04: // ROOT_NATIVE_STACK
                hprof.readGCRoot(in, "native stack", 4)
            case 0x05: // ROOT_STICKY_CLASS
                hprof.readGCRoot(in, "sticky class", 0)
            case 0x06: // ROOT_THREAD_BLOCK
                hprof.readGCRoot(in, "thread block", 4)
            case 0x07: // ROOT_MONITOR_USED
                hprof.readGCRoot(in, "monitor used", 0)
            case 0x08: // ROOT_THREAD_OBJECT
                hprof.readGCRoot(in, "thread object", 8)
            case 0xff: // ROOT_UNKNOWN
                hprof.readGCRoot(in, "unknown root", 0)
            default:
                log.Fatalf("Unknown HPROF record type %d at %d\n", tag, in.Offset() - 1)
        }
    }
    return numRecords
}

// Read a CLASS_DUMP record, which defines the layout of a class in the heap
//
func (hprof *HProfReader) readClassDump(in *MappedSection) {

    // Header

    in.Demand(7 * hprof.idSize + 8)
    hid := hprof.readId(in) // hid
    in.Skip(4) // stack serial
    superHid := hprof.readId(in) // superHid
    in.Skip(5 * hprof.idSize) // skip class loader ID, signer ID, protection domain ID, 2 reserved
    in.Skip(4) // instance size

    // Class name was read early as a UTF8 record

    heap := hprof.Heap
    nameId, ok := heap.classNames[hid]
    if ! ok {
        log.Fatalf("Class with hid %d has no name mapping\n", hid)
    }

    name, ok := heap.strings[nameId]
    if ! ok {
        log.Fatalf("Class name id %d for class hid %d has no mapping\n", hid)
    }

    // Skip over constant pool

    in.Demand(2)
    numConstants := in.GetUInt16()
    in.Demand(11 * uint32(numConstants))

    for i := 0; i < int(numConstants); i++ {
        in.Skip(2)
        jtype := hprof.readJType(in)
        in.Skip(jtype.Size)
    }

    // Static fields

    in.Demand(2)
    numStatics := in.GetUInt16()
    in.Demand(11 * uint32(numStatics))
    staticRefs := []HeapId{}

    for i := 0; i < int(numStatics); i++ {
        in.Skip(hprof.idSize) // field name ID
        jtype := hprof.readJType(in)
        if jtype.IsObj {
            toHid := hprof.readId(in)
            if toHid != 0 {
                staticRefs = append(staticRefs, toHid)
            }
        } else {
            in.Skip(jtype.Size)
        }
    }

    // Instance fields

    in.Demand(2)
    numFields := in.GetUInt16()
    fieldNames := make([]string, numFields, numFields)
    fieldTypes := make([]*JType, numFields, numFields)

    for i := 0; i < int(numFields); i++ {
        fieldName, ok := heap.strings[hprof.readId(in)]
        if ! ok {
            log.Fatalf("No name for field %d in class with hid %d\n", i, hid)
            fieldName = "UNKNOWN"
        }
        fieldNames[i] = fieldName
        fieldTypes[i] = hprof.readJType(in)
    }

    heap.addClass(name, hid, superHid, fieldNames, fieldTypes, staticRefs)
}

// Read a GC root.  This has the HID at the start followed by some amount
// of per-root data that we don't use.
//
func (hprof *HProfReader) readGCRoot(in *MappedSection, kind string, skip uint32) {
    in.Demand(hprof.idSize + skip)
    hid := hprof.readId(in)
    // TODO fix
    hprof.Heap.gcRoots = append(hprof.Heap.gcRoots, hid)
    in.Skip(skip)
}

// Read header for an object instance, then pass off required info
// for segReader to handle it in the background.
//
func (hprof *HProfReader) readInstance(in *MappedSection) {

    offset := in.Offset() -1 // segReader must read record tag again
    heap := hprof.Heap

    // header is
    //
    // instance id      HeapId
    // stack serial     uint32      (ignored)
    // class id         HeapId
    // length           uint32

    in.Demand(8 + 2 * hprof.idSize)
    hid := hprof.readId(in)
    in.Skip(4) // stack serial
    class := heap.HidClass(hprof.readId(in))
    length := in.GetUInt32()
    heap.AddInstance(hid, class, length + heap.idSize) // include object monitor

    if heap.segReader != nil {
        heap.doInstance(offset, heap.MaxObjectId, class)
    }

    in.Skip(length)
}

// Read header for an array, then pass off required info
// for segReader to handle it in the background.
//
func (hprof *HProfReader) readArray(in *MappedSection, isObjects bool) {

    offset := in.Offset() - 1 // segReader must read record tag again
    heap := hprof.Heap

    // header is
    //
    // instance id      HeapId
    // stack serial     uint32      (ignored)
    // # elements       uint32

    in.Demand(hprof.idSize + 8)
    hid := hprof.readId(in)
    in.Skip(4) // stack serial
    count := in.GetUInt32()

    if isObjects {
        in.Demand(hprof.idSize)
        class := heap.HidClass(hprof.readId(in))
        heap.AddInstance(hid, class, (count + 2) * hprof.idSize) // include header size
        if heap.segReader != nil {
            heap.doInstance(offset, heap.MaxObjectId, class)
        }
        in.Skip(count * hprof.idSize)
    } else {
        in.Demand(1)
        jtype :=  hprof.readJType(in)
        heap.AddInstance(hid, jtype.Class, count * jtype.Size + 2 * heap.idSize) // include header size
        if heap.segReader != nil {
            heap.doInstance(offset, heap.MaxObjectId, jtype.Class)
        }
        in.Skip(count * jtype.Size)
    }

}

// Read a native ID from heap data.
//
func (hprof *HProfReader) readId(in *MappedSection) HeapId {
    if (hprof.longIds) {
        return HeapId(in.GetUInt64())
    }
    return HeapId(in.GetUInt32())
}

// Read a "Basic Type" ID from heap data and return the JType
//
func (hprof *HProfReader) readJType(in *MappedSection) *JType {
    tag := int(in.GetByte())
    if tag < 0 || tag >= len(hprof.jtypes) {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    jtype := hprof.jtypes[tag]
    if jtype == nil {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    return jtype
}

