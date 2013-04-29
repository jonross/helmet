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

// Manages a pool of segWorkers in separate goroutines that can independently read
// portions of a heap segment.  Allows us to do as much segment processing as possible
// in parallel
//
type segReader struct {
    // parent reader
    *HProfReader
    // All active & inactive workers
    workers []*segWorker
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
    // parent reader
    *segReader
    // ID of first object
    firstOid ObjectId
    // what are their class defs
    classes []*ClassDef
    // where are they found in the heap dump
    offsets []uint64
    // how many to process
    count int
    // references found
    refs RefBag
}

// Create a segment reader with one worker per CPU.
//
func makeSegReader(hr *HProfReader) *segReader {

    reader := &segReader{
        HProfReader: hr,
        workers: make([]*segWorker, runtime.NumCPU()),
        avail: make(chan *segWorker),
        active: nil,
        // Most segments are 1GB so partition for 100 work cycles
        batchSize: RecordsPerGB/100,
    }

    // Create each worker and put it on the available list.

    go func() {
        for i, _ := range reader.workers {
            reader.workers[i] = &segWorker{
                id: i + 1,
                segReader: reader,
                firstOid: 0,
                classes: make([]*ClassDef, reader.batchSize),
                offsets: make([]uint64, reader.batchSize),
                count: 0,
            }
            reader.avail <- reader.workers[i]
        }
    }()

    // Tee up the first worker to take doInstance() calls.

    reader.active = <-reader.avail
    return reader
}

// Add a location to be processed to the active segment worker.  If its queue
// is full, tell it to proceed() and ready the next worker.
//
func (reader *segReader) doInstance(offset uint64, oid ObjectId, class *ClassDef) {
    worker := reader.active
    i := worker.count
    if i == 0 {
        worker.firstOid = oid
    }
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

// Primary work loop for a segment worker.  This handles the subset of HPROF tags
// that segReader.doInstance() hands off to us.
//
func (worker *segWorker) process() {
    if worker.count > 0 {
        start := worker.offsets[0]
        in := worker.MappedFile.MapAt(start)
        oid := worker.firstOid
        for i := 0; i < worker.count; i++ {
            in.Skip(uint32(worker.offsets[i] - in.Offset()))
            tag := in.GetByte()
            switch tag {
                case 0x21: // INSTANCE_DUMP
                    worker.readInstance(in, oid, worker.classes[i])
                case 0x22: // OBJECT_ARRAY
                    worker.readArray(in, oid, true)
                case 0x23: // PRIMITIVE_ARRAY
                    worker.readArray(in, oid, false)
                default:
                    log.Fatalf("Unhandled record type %d in worker\n", tag)
            }
            oid++
        }
        worker.count = 0
    }
    worker.avail <- worker
}

// Shut down all segment workers, allowing them to be garbage collected, by launching
// the current active one (even if empty) then draining the "available" channel.
//
func (reader *segReader) close() []*RefBag {
    reader.proceed(false)
    bags := []*RefBag{}
    for i := 0; i < runtime.NumCPU(); i++ {
        worker := <-reader.avail
        bags = append(bags, &worker.refs)
    }
    return bags
}

// The busy side of Heap.readInstance; record the instance data + references
// to other objects.
//
func (worker *segWorker) readInstance(in *MappedSection, oid ObjectId, class *ClassDef) {

    // header is
    //
    // instance id      HeapId      (already known)
    // stack serial     uint32      (ignored)
    // class id         HeapId      (already known)
    // length           uint32

    in.Skip(4 + 2 * worker.IdSize)
    in.Demand(4)
    length := in.GetUInt32()

    in.Demand(length)
    start := in.Offset()
    end := start + uint64(length)
    cursor := uint32(0)

    for _, offset := range class.RefOffsets() {
        in.Skip(offset - cursor)
        toHid := worker.readId(in)
        if toHid != 0 {
            worker.refs.Add(oid, toHid)
        }
        cursor += worker.IdSize
    }

    in.Skip(uint32(end - in.Offset()))
}

// The busy side of Heap.readArray; record the instance data + references
// to other objects.
//
func (worker *segWorker) readArray(in *MappedSection, oid ObjectId, isObjects bool) {

    // header is
    //
    // instance id      HeapId      (already known)
    // stack serial     uint32      (ignored)
    // # elements       uint32

    in.Skip(worker.IdSize + 4)
    in.Demand(4)
    count := in.GetUInt32()

    if (isObjects) {
        in.Skip(worker.IdSize) // already know class
        in.Demand(count * worker.IdSize)
        for i := uint32(0); i < count; i++ {
            toHid := worker.readId(in)
            if toHid != 0 {
                worker.refs.Add(oid, toHid)
            }
        }
    } else {
        in.Demand(1)
        jtype := worker.readJType(in)
        in.Skip(count * jtype.Size)
    }
}


