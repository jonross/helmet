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
    "runtime"
)

// Responsible for reading an HPROF binary heap dump and handing information off
// to a Heap instance.  See also SegReader.
//
type HProfReader struct {
    // From command-line options
    *Options
    // Yes that
    Filename string
    // And that
    *MappedFile
    // Size of a native ID on the heap, 4 or 8
    IdSize uint32
    // true if IdSize is 8
    longIds bool
    // target heap tracker
    *Heap
    // segment reader, if needRefs is true
    *SegReader
}

func ReadHeapDump(filename string, options *Options) *Heap {

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
        Options: options,
        Filename: filename,
        MappedFile: mappedFile,
    }

    if options.NeedRefs {
        hprof.SegReader = NewSegReader(hprof)
    }

    hprof.IdSize = in.GetUInt32()
    if hprof.IdSize != 4 && hprof.IdSize != 8 {
        log.Fatalf("Unknown reference size %d\n", hprof.IdSize)
    }
    hprof.longIds = hprof.IdSize == 8
    in.Skip(8) // skip timestamp

    return hprof.read(in, options)
}

// Complete heap reader in one call.  Most of the error conditions (like
// unresolvable classes) cause panics BTW.
//
func (hprof *HProfReader) read(in *MappedSection, options *Options) *Heap {

    hprof.Heap = NewHeap(hprof.IdSize)
    heap := hprof.Heap

    headerSize := uint32(9)
    numRecords := 0
    numStrings := 0

    // TODO: keep input struct constant, don't return different one

    for in.Demand(headerSize) {

        numRecords++
        tag := in.GetByte()
        in.Skip(4) // Skip timestamp
        length := in.GetUInt32()
        // log.Printf("Record type %d len %d at %d\n", tag, length, in.Offset() - uint64(headerSize))
        // demand length here?

        // A function table would be more efficient but there aren't
        // that many top-level records compared to instance records.

        switch tag {
            case 0x01: // UTF8
                numStrings++
                hid := hprof.readId(in)
                str := in.GetString(length - hprof.IdSize)
                heap.AddString(hid, str)

            case 0x02: // LOAD_CLASS
                in.Skip(4) // skip classSerial
                classHid := hprof.readId(in)
                in.Skip(4) // skip stackSerial
                nameHid := hprof.readId(in)
                heap.AddClassName(classHid, nameHid)

            case 0x0c, 0x1c: // HEAP_DUMP, HEAP_DUMP_SEGMENT
                log.Printf("Heap dump or segment of %d MB at %x", 
                           length / 1048576, in.Offset() - uint64(headerSize))
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

    heap.PostProcess(hprof.SegReader)
    hprof.SegReader = nil // allow GC
    runtime.GC()

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
        in.Demand(1)
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
                hprof.readGCRoot(in, "JNI global", hprof.IdSize)
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

    // header is
    //
    // class heap id    HeapId
    // stack serial     uint32      (ignored)
    // superclass id    HeapId
    // classloader id   HeapId      (ignored)
    // signer id        HeapId      (ignored)
    // prot domain id   HeapId      (ignored)
    // reserved 1       HeapId      (ignored)
    // reserved 2       HeapId      (ignored)
    // instance size    uint32      TODO: use this?

    in.Demand(7 * hprof.IdSize + 8)
    hid := hprof.readId(in) // hid
    in.Skip(4)
    superHid := hprof.readId(in) // superHid
    in.Skip(5 * hprof.IdSize)
    in.Skip(4)

    // Class name was read earlier as a UTF8 record

    heap := hprof.Heap
    nameId := heap.ClassNameId(hid)
    if nameId == 0 {
        log.Fatalf("Class with hid %d has no name mapping\n", hid)
    }

    name := heap.StringWithId(nameId)
    if name == "" {
        log.Fatalf("Class name id %d for class hid %d has no mapping\n", hid)
    }

    // Skip over constant pool

    in.Demand(2)
    numConstants := in.GetUInt16()
    in.Demand(11 * uint32(numConstants)) // worst case, all are longs

    for i := 0; i < int(numConstants); i++ {
        in.Skip(2)
        jtype := hprof.readJType(in)
        in.Skip(jtype.Size)
    }

    // Static fields

    in.Demand(2)
    numStatics := in.GetUInt16()
    in.Demand(11 * uint32(numStatics)) // worst case, all are longs
    staticRefs := []HeapId{}

    for i := 0; i < int(numStatics); i++ {
        in.Skip(hprof.IdSize) // field name ID
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
    numFields := uint32(in.GetUInt16())
    fieldNames := make([]string, numFields, numFields)
    fieldTypes := make([]*JType, numFields, numFields)

    in.Demand(numFields * (1 + hprof.IdSize))
    for i := 0; i < int(numFields); i++ {
        fieldName := heap.StringWithId(hprof.readId(in))
        if fieldName == "" {
            log.Fatalf("No name for field %d in class with hid %d\n", i, hid)
            fieldName = "UNKNOWN"
        }
        fieldNames[i] = fieldName
        fieldTypes[i] = hprof.readJType(in)
    }

    heap.AddClass(name, hid, superHid, fieldNames, fieldTypes, staticRefs)
}

// Read a GC root.  This has the HID at the start followed by some amount
// of per-root data that we don't use.
//
func (hprof *HProfReader) readGCRoot(in *MappedSection, kind string, skip uint32) {
    in.Demand(hprof.IdSize + skip)
    hid := hprof.readId(in)
    hprof.AddRoot(hid)
    in.Skip(skip)
}

// Read header for an object instance, then pass off required info
// for SegReader to handle it in the background.
//
func (hprof *HProfReader) readInstance(in *MappedSection) {

    offset := in.Offset() -1 // SegReader must read record tag again
    heap := hprof.Heap

    // header is
    //
    // instance id      HeapId
    // stack serial     uint32      (ignored)
    // class id         HeapId
    // length           uint32

    in.Demand(8 + 2 * hprof.IdSize)
    hid := hprof.readId(in)
    in.Skip(4) // stack serial
    class := heap.HidClass(hprof.readId(in))
    length := in.GetUInt32()
    oid := heap.AddInstance(hid, class, length + hprof.IdSize, offset) // include object monitor

    if hprof.SegReader != nil {
        hprof.doInstance(offset, oid, class)
    }

    in.Skip(length)
}

// Read header for an array, then pass off required info
// for SegReader to handle it in the background.
//
func (hprof *HProfReader) readArray(in *MappedSection, isObjects bool) {

    offset := in.Offset() - 1 // SegReader must read record tag again
    heap := hprof.Heap

    // header is
    //
    // instance id      HeapId
    // stack serial     uint32      (ignored)
    // # elements       uint32

    in.Demand(hprof.IdSize + 8)
    hid := hprof.readId(in)
    in.Skip(4) // stack serial
    count := in.GetUInt32()

    // TODO heap.addPrimitiveArray(id, jtype, offset, count * jtype.size + 2 * heap.IdSize)

    if isObjects {
        in.Demand(hprof.IdSize)
        class := heap.HidClass(hprof.readId(in))
        oid := heap.AddInstance(hid, class, (count + 2) * hprof.IdSize, offset) // include header size
        if hprof.SegReader != nil {
            hprof.doInstance(offset, oid, class)
        }
        in.Skip(count * hprof.IdSize)
    } else {
        in.Demand(1)
        jtype :=  hprof.readJType(in)
        heap.AddInstance(hid, jtype.Class, count * jtype.Size + 2 * hprof.IdSize, offset) // include header size
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
    if tag < 0 || tag >= len(hprof.Jtypes) {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    jtype := hprof.Jtypes[tag]
    if jtype == nil {
        log.Fatalf("Unknown basic type %d at %d\n", tag, in.Offset() - 1)
    }
    return jtype
}

