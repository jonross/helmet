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

package util

import (
    "log"
    "math"
    "os"
    "sync"
    "syscall"
)

// MappedFile tracks a set of memory-mapped sections on a file and provides
// low-level data access like GetUInt32.  This is a helpful abstraction even
// given Go slices because unfortunately, UNIX cannot map > 2^31-1 bytes at
// a time, and we need to abstract away the case where code is reading data
// and suddenly finds a value that runs over the end of a section.  Awkward.
//
type MappedFile struct {
    filename string
    file *os.File
    // total size of the file
    size uint64
    // what have we mapped so far
    sections [][]byte
    // for concurrent modification
    lock *sync.Mutex
}

// This represents a single mapped section on a MappedFile.
//
type MappedSection struct {
    mf *MappedFile
    // As returned by syscall.Mmap
    base []byte
    // Where are we now
    offset int32
}

// Create a MappedFile.
//
func MapFile(filename string) (mf *MappedFile, err error) {
    mf = &MappedFile{filename: filename, sections: [][]byte{}}
    mf.file, err = os.Open(filename)
    if err != nil {
        return nil, err
    }
    info, err := mf.file.Stat()
    if err != nil {
        return nil, err
    }
    mf.size = uint64(info.Size())
    mf.lock = &sync.Mutex{}
    return
}

// Unmap all mapped sections from a mapped file.  Does not close the file.
//
func (mf *MappedFile) UnmapAll() {
    for _, bytes := range mf.sections {
        err := syscall.Munmap(bytes)
        if err != nil {
            log.Fatalf("Failed to unmap memory at %x\n", bytes)
        }
    }
    mf.sections = [][]byte{}
}

// Map the largest possible section starting at a given offset.  Normally this is called 
// automatically by Demand().  Panics on failure since there is no recovery.
//
func (mf *MappedFile) MapAt(offset uint64) *MappedSection {
    length := mf.size - offset
    if length > uint64(math.MaxInt32) {
        length = uint64(math.MaxInt32)
    }
    bytes, err := syscall.Mmap(int(mf.file.Fd()), int64(offset), int(length),
                               syscall.PROT_READ, syscall.MAP_SHARED)
    if err != nil {
        log.Fatalf("Can't map %s %d bytes at %d: %s\n", mf.filename, length, offset, err)
    }
    log.Printf("Mapping %s %d bytes at %d\n", mf.filename, length, offset)
    mf.addSection(bytes)
    return &MappedSection{mf, bytes, 0}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// Add a new section to the list of mapped sections.  This uses the lock field because multiple
// goroutines may be independently mapping different locations.
//
func (mf *MappedFile) addSection(bytes []byte) {
    mf.lock.Lock()
    defer mf.lock.Unlock()
    mf.sections = append(mf.sections, bytes);
}

