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

type Heap struct {
    // Yes that
    filename string
    // And that
    mappedFile *MappedFile
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
    // temporary mapping from HeapIds to ObjectIds
    objectMap *ObjectMap
    // initial list of referrers
    refsFrom []ObjectId
    // initial list of referees
    refsTo []HeapId
    // packages to search for unqualified class names
    autoPrefixes []string
    // concurrent heap segment reader
    *segReader
}

// Complete heap reader in one call.  Most of the error conditions (like
// unresolvable classes) cause panics BTW.
//
func ReadHeap(filename string) (heap *Heap, err error) {

    heap = &Heap{filename: filename}
    heap.mappedFile, err = MapFile(filename)
    if err != nil {
        return nil, err
    }
    defer heap.mappedFile.Close()

    // Verify this is a real HPROF file & determine native ID size.

    in := heap.mappedFile.MapAt(0)
    version := string(in.GetRaw(18))
    in.Skip(1) // trailing NUL

    if version != "JAVA PROFILE 1.0.1" && version != "JAVA PROFILE 1.0.2" {
        log.Fatalf("Unknown heap version %s\n", version)
    }

    heap.idSize = in.GetUInt32()
    if heap.idSize != 4 && heap.idSize != 8 {
        log.Fatalf("Unknown reference size %d\n", heap.idSize)
    }
    heap.longIds = heap.idSize == 8
    in.Skip(8) // skip timestamp

    // Set up major data structures
    // TODO: presize based on file size

    heap.strings = make(map[HeapId]string, 100000)
    heap.classNames = make(map[HeapId]HeapId, 50000)
    heap.gcRoots = make([]HeapId, 0, 10000)
    headerSize := uint32(9)

    heap.NumClasses = 0
    heap.classes = []*ClassDef{nil} // leave room for entry [0]

    heap.classesByName = make(map[string]*ClassDef, 50000)
    heap.classesByHid = make(map[HeapId]*ClassDef, 50000)

    heap.NumObjects = 0
    heap.objectMap = MakeObjectMap()
    heap.objectCids = make([]ClassId, 1, 10000000) // entry[0] not used

    heap.refsFrom = []ObjectId{}
    heap.refsTo = []HeapId{}

    heap.segReader = makeSegReader(heap)

    heap.autoPrefixes = []string {
        "java.lang.",
        "java.util.",
        "java.util.concurrent." }

    // JType descriptors are indexed by the "basic type" tag
    // found in a CLASS_DUMP or PRIMITIVE_ARRAY_DUMP

    heap.jtypes = []*JType{
        nil,  // 0 unused
        nil,  // 1 unused
        &JType{"", true, heap.idSize, 0}, // object descriptor unnamed because it varies by actual type
        nil,  // 3 unused
        &JType{"[Z", false, 1, 0},
        &JType{"[C", false, 2, 0},
        &JType{"[F", false, 4, 0},
        &JType{"[D", false, 8, 0},
        &JType{"[B", false, 1, 0},
        &JType{"[S", false, 2, 0},
        &JType{"[I", false, 4, 0},
        &JType{"[J", false, 8, 0} }

    // TODO: keep input struct constant, don't return different one

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
                numRecords += heap.readSegment(in, length)

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

    heap.segReader.proceed(false)
    heap.segReader.close()

    // class def post-processing

    for _, def := range heap.classes[1:] {
        def.Cook()
    }

    log.Printf("%d records, %d UTF8\n", numRecords, numStrings)
    log.Printf("%d objects\n", heap.NumObjects)
    log.Printf("%d references\n", len(heap.refsFrom))
    return
}

// Handle HEAP_DUMP or HEAP_DUMP_SEGMENT record.
//
func (heap *Heap) readSegment(in *MappedSection, length uint32) int {
    end := in.Offset() + uint64(length)
    numRecords := 0
    for in.Offset() < end {
        numRecords++
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
    return numRecords
}

// Read a GC root.  This has the HID at the start followed by some amount
// of per-root data that we don't use.
//
func (heap *Heap) readGCRoot(in *MappedSection, kind string, skip uint32) {
    in.Demand(heap.idSize + skip)
    hid := heap.readId(in)
    heap.gcRoots = append(heap.gcRoots, hid)
    in.Skip(skip)
}

// Read an array of objects or numeric primitives.
//
func (heap *Heap) readArray(in *MappedSection, isObjects bool) {

    // TODO demand

    offset := in.Offset()
    hid := heap.readId(in)
    in.Skip(4) // stack serial
    count := in.GetUInt32()

    if isObjects {
        // TODO put back
        in.Skip((1 + count) * heap.idSize)
        return
        classHid := heap.readId(in) // array class hid
        classDef := heap.HidClass(classHid)
        oid := heap.nextOid(hid, classDef)
        heap.addInstance(oid, classDef, offset, heap.idSize * (count + 1))
        for ; count > 0; count -= 1 {
            toHid := heap.readId(in)
            if toHid != 0 {
                heap.addReference(oid, toHid)
            }
        }
    } else {
        jtype := heap.readJType(in)
        if count > 0 {
            in.Skip(count * jtype.Size)
        }
            /*
            heap.addPrimitiveArray(id, jtype, offset, count * jtype.size + 2 * heap.idSize)
            */
    }
}

func (heap *Heap) addInstance(oid ObjectId, def *ClassDef, offset uint64, size uint32) {
}

func (heap *Heap) addReference(from ObjectId, to HeapId) {
    heap.refsFrom = append(heap.refsFrom, from)
    heap.refsTo = append(heap.refsTo, to)
}

func (heap *Heap) readClassDump(in *MappedSection) {

    in.Demand(7 * heap.idSize + 8)
    hid := heap.readId(in) // hid
    in.Skip(4) // stack serial
    superHid := heap.readId(in) // superHid
    in.Skip(5 * heap.idSize) // skip class loader ID, signer ID, protection domain ID, 2 reserved
    in.Skip(4) // instance size

    nameId, ok := heap.classNames[hid]
    if ! ok {
        log.Fatalf("Class with hid %d has no name mapping\n", hid)
    }

    name, ok := heap.strings[nameId]
    if ! ok {
        log.Fatalf("Class name id %d for class hid %d has no mapping\n", hid)
    }

    // Update the JTypes if we've found a primitive array type.

    if len(name) == 2 && name[0] == '[' {
        for _, jtype := range heap.jtypes {
            if jtype != nil && name == jtype.ArrayClass {
                // log.Printf("Found %s hid %d\n", name, hid)
                jtype.Hid = hid
            }
        }
    }

    // Skip over constant pool

    in.Demand(2)
    numConstants := in.GetUInt16()
    in.Demand(11 * uint32(numConstants))

    for i := 0; i < int(numConstants); i++ {
        in.Skip(2)
        jtype := heap.readJType(in)
        in.Skip(jtype.Size)
    }

    // Static fields

    in.Demand(2)
    numStatics := in.GetUInt16()
    in.Demand(11 * uint32(numStatics))

    for i := 0; i < int(numStatics); i++ {
        in.Skip(heap.idSize) // field name ID
        jtype := heap.readJType(in)
        if jtype.IsObj {
            heap.readId(in)
            // if (toHid != 0)
            //     heap.addStaticReference(classId, toId)
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
        fieldName, ok := heap.strings[heap.readId(in)]
        if ! ok {
            log.Fatalf("No name for field %d in class with hid %d\n", i, hid)
            fieldName = "UNKNOWN"
        }
        fieldNames[i] = fieldName
        fieldTypes[i] = heap.readJType(in)
    }

    heap.addClass(name, hid, superHid, fieldNames, fieldTypes)
}

// Read header for an object instance, then pass off required info
// for segReader to handle it in the background.
//
func (heap *Heap) readInstance(in *MappedSection) {
    offset := in.Offset() -1 // must read record tag again
    in.Demand(8 + 2 * heap.idSize)
    hid := heap.readId(in)
    in.Skip(4) // stack serial
    class := heap.HidClass(heap.readId(in))
    length := in.GetUInt32()
    in.Skip(length)
    heap.doInstance(offset, heap.nextOid(hid, class), class)
}

// Read a native ID from heap data.
//
func (heap *Heap) readId(in *MappedSection) HeapId {
    if (heap.longIds) {
        return HeapId(in.GetUInt64())
    }
    return HeapId(in.GetUInt32())
}

// Read a "Basic Type" ID from heap data and return the JType
//
func (heap *Heap) readJType(in *MappedSection) *JType {
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

//////////////////////////////////////////////////////////////////////////////////////////

// Manages a pool of segWorkers in separate goroutines that can independently read
// portions of a heap segment.  Allows us to do as much segment processing as possible
// in parallel
//
type segReader struct {
    // Unused workers are waiting to be taken off this channel
    avail chan *segWorker
    // This worker is having its queue of ids/offsets built
    active *segWorker
    // How large does the queue get before we process it
    batchSize int
}

type segWorker struct {
    // unique id for debugging
    id int
    // parent heap
    heap *Heap
    // parent reader
    *segReader
    // object IDs to process
    objectIds []ObjectId
    // what are their class defs
    classes []*ClassDef
    // where are they found in the heap dump
    offsets []uint64
    // how many to process
    count int
    // referrers found
    refsFrom [][]ObjectId
    // referees found
    refsTo [][]HeapId
}

func makeSegReader(heap *Heap) *segReader {
    // Most segments are 1GB so we'll partition the segment for 100 worker passes
    reader := &segReader{make(chan *segWorker), nil, RecordsPerGB/100}
    go func() {
        for i := 0; i < runtime.NumCPU(); i++ {
            objectIds := make([]ObjectId, reader.batchSize)
            classes := make([]*ClassDef, reader.batchSize)
            offsets := make([]uint64, reader.batchSize)
            refsFrom := [][]ObjectId{make([]ObjectId, 0, 1000000)}
            refsTo := [][]HeapId{make([]HeapId, 0, 1000000)}
            reader.avail <- &segWorker{i + 1, heap, reader, objectIds, 
                                        classes, offsets, 0, refsFrom, refsTo}
        }
    }()
    reader.active = <- reader.avail
    return reader
}

// Add a location to be processed to the active segment worker.  If its queue
// is full, tell it to proceed() and ready the next worker.
//
func (reader *segReader) doInstance(offset uint64, oid ObjectId, class *ClassDef) {
    worker := reader.active
    i := worker.count
    worker.objectIds[i] = oid
    worker.classes[i] = class
    worker.offsets[i] = offset
    worker.count++
    if worker.count == reader.batchSize {
        reader.proceed(true)
    }
}

// Called directly or via segReader.doInstance() -- start processing the worker on
// its own goroutine and (if more work is indicated) pull the next available worker 
// from the channel.
//
func (reader *segReader) proceed(more bool) {
    go reader.active.process()
    if more {
        reader.active = <- reader.avail
    }
}

// Shut down all segment workers, allowing them to be garbage collected, by
// launching the current active one (even if empty) then draining the channel.
//
func (reader *segReader) close() {
    go reader.active.process()
    for i := 0; i < runtime.NumCPU(); i++ {
        <- reader.avail
    }
}

func (worker *segWorker) process() {
    if worker.count == 0 {
        return
    }
    start := worker.offsets[0]
    in := worker.heap.mappedFile.MapAt(start)
    for i := 0; i < worker.count; i++ {
        in.Skip(uint32(worker.offsets[i] - in.Offset()))
        tag := in.GetByte()
        switch tag {
            case 0x21: // INSTANCE_DUMP
                worker.readInstance(in, worker.objectIds[i], worker.classes[i])
            case 0x22: // OBJECT_ARRAY
                // worker.readArray(in, true)
            case 0x23: // PRIMITIVE_ARRAY
                // worker.readArray(in, false)
            default:
                log.Fatalf("Unhandled record type %d in worker\n", tag)
        }
    }
    worker.count = 0
    worker.avail <- worker
}

func (worker *segWorker) readInstance(in *MappedSection, oid ObjectId, class *ClassDef) {

    heap := worker.heap
    in.Skip(4 + 2 * heap.idSize)
    in.Demand(4)
    length := in.GetUInt32()

    in.Demand(length)
    start := in.Offset()
    end := start + uint64(length)
    cursor := uint32(0)

    // heap.addInstance(oid, class, start, length)

    for _, offset := range class.RefOffsets() {
        in.Skip(offset - cursor)
        toHid := heap.readId(in)
        if toHid != 0 {
            worker.refsFrom = xappendOid(worker.refsFrom, oid)
            worker.refsTo = xappendHid(worker.refsTo, toHid)
        }
        cursor += heap.idSize
    }

    in.Skip(uint32(end - in.Offset()))
}

