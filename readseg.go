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

// Manages a pool of SegWorkers in separate goroutines that can independently read
// portions of a heap segment.  Allows us to do as much segment processing as possible
// in parallel
//
type SegReader struct {
    // parent reader
    *HProfReader
    // All active & inactive workers
    workers []*SegWorker
    // Unused workers are waiting to be taken off this channel
    avail chan *SegWorker
    // This worker is having its queue of ids/offsets built
    active *SegWorker
    // How large does the queue get before we process it
    batchSize int
}

type SegWorker struct {
    // unique id for debugging
    id int
    // parent reader
    *SegReader
    // object IDs to read
    oids []ObjectId
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
func NewSegReader(hr *HProfReader) *SegReader {

    reader := &SegReader{
        HProfReader: hr,
        workers: make([]*SegWorker, runtime.NumCPU()),
        avail: make(chan *SegWorker),
        active: nil,
        // Most segments are 1GB so partition for 100 work cycles
        batchSize: RecordsPerGB/100,
    }

    // Create each worker and put it on the available list.

    go func() {
        for i, _ := range reader.workers {
            reader.workers[i] = &SegWorker{
                id: i + 1,
                SegReader: reader,
                oids: make([]ObjectId, reader.batchSize),
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
func (reader *SegReader) doInstance(offset uint64, oid ObjectId, class *ClassDef) {
    worker := reader.active
    i := worker.count
    worker.oids[i] = oid
    worker.classes[i] = class
    worker.offsets[i] = offset
    worker.count++
    if worker.count == reader.batchSize {
        reader.proceed(true)
    }
}

// Called directly or via SegReader.doInstance() -- start processing the worker on
// its own goroutine and (if more work is indicated) pull the next available worker 
// from the channel.
//
func (reader *SegReader) proceed(more bool) {
    go reader.active.process()
    if more {
        reader.active = <- reader.avail
    }
}

// Primary work loop for a segment worker.  This handles the subset of HPROF tags
// that SegReader.doInstance() hands off to us.
//
func (worker *SegWorker) process() {
    if worker.count > 0 {
        start := worker.offsets[0]
        in := worker.MappedFile.MapAt(start)
        for i := 0; i < worker.count; i++ {
            // TODO demand first?
            in.Skip(uint32(worker.offsets[i] - in.Offset()))
            tag := in.GetByte()
            switch tag {
                case 0x21: // INSTANCE_DUMP
                    worker.readInstance(in, worker.oids[i], worker.classes[i])
                case 0x22: // OBJECT_ARRAY
                    worker.readArray(in, worker.oids[i], worker.classes[i])
                default:
                    log.Fatalf("Unhandled record type %d in worker\n", tag)
            }
        }
        worker.count = 0
    }
    worker.avail <- worker
}

// Shut down all segment workers, allowing them to be garbage collected, by launching
// the current active one (even if empty) then draining the "available" channel.
//
func (reader *SegReader) close() []*RefBag {
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
func (worker *SegWorker) readInstance(in *MappedSection, oid ObjectId, class *ClassDef) {

    // header is
    //
    // instance id      HeapId      (already known)
    // stack serial     uint32      (ignored)
    // class id         HeapId      (already known)
    // length           uint32

    in.Skip(4 + 2 * worker.IdSize)
    in.Demand(4)
    in.Demand(in.GetUInt32())
    cursor := uint32(0)

    for _, offset := range class.RefOffsets() {
        skip := offset - cursor
        in.Skip(skip)
        toHid := worker.readId(in)
        if toHid != 0 {
            // log.Printf("in readInst %d cid=%d a %s -> %x\n", oid, class.Cid, class.Name, toHid)
            worker.refs.AddReference(oid, toHid)
        }
        cursor += skip + worker.IdSize
    }
}

// The busy side of Heap.readArray; record the instance data + references
// to other objects.
//
func (worker *SegWorker) readArray(in *MappedSection, oid ObjectId, class *ClassDef) {

    // header is
    //
    // instance id      HeapId      (already known)
    // stack serial     uint32      (ignored)
    // # elements       uint32

    in.Demand(worker.IdSize + 8)
    in.Skip(worker.IdSize + 4)
    count := in.GetUInt32()

    in.Demand(count * (1 + worker.IdSize))
    in.Skip(worker.IdSize) // already know class
    for i := uint32(0); i < count; i++ {
        toHid := worker.readId(in)
        if toHid != 0 {
            // log.Printf("in readArray %d cid=%d a %s -> %x\n", oid, class.Cid, class.Name, toHid)
            worker.refs.AddReference(oid, toHid)
        }
    }

    // TODO - have added instance??
}
