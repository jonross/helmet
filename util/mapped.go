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
// and suddenly finds a value that runs over the end of a section.
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
    // Underlying MappedFile
    mappedFile *MappedFile
    // As returned by syscall.Mmap
    base []byte
    // Size of mapped section
    size uint32
    // Offset of base within the file
    globalOffset uint64
    // Where are we now
    localOffset uint32
}

//////////////////////////////////////////////////////////////////////////////////////////

// Create a MappedFile.
//
func MapFile(filename string) (mf *MappedFile, err error) {
    mf = &MappedFile{filename: filename, sections: [][]byte{}}
    mf.file, err = os.Open(filename)
    if err != nil {
        mf.file.Close()
        return nil, err
    }
    info, err := mf.file.Stat()
    if err != nil {
        mf.file.Close()
        return nil, err
    }
    mf.size = uint64(info.Size())
    mf.lock = &sync.Mutex{}
    return
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
    return &MappedSection{mf, bytes, uint32(length), offset, 0}
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

// Does UnmapAll and closes the file.
//
func (mf *MappedFile) Close() {
    mf.UnmapAll()
    mf.file.Close()
}

//////////////////////////////////////////////////////////////////////////////////////////

// Require at least count bytes in the current mapped section.  If not available, remap
// it from the current location.  Note only checks insufficient bytes in the section, not
// the file itself, so that a caller can make a conservative overestimate rather than
// calling Demand() more frequently with smaller amounts.  Returns nil if no more data
// available.
//
func (ms *MappedSection) Demand(count uint32) *MappedSection {
    remain := ms.size - ms.localOffset
    if (remain < count) {
        newOffset := ms.globalOffset + uint64(ms.localOffset)
        if newOffset == ms.mappedFile.size {
            return nil
        }
        return ms.mappedFile.MapAt(newOffset)
    }
    return ms
}

// Read a byte at the current offset and advance the offset 1 byte.
//
func (ms *MappedSection) GetByte() byte {
    ret := ms.base[ms.localOffset]
    ms.localOffset++
    return ret
}

// Read a signed 32-bit integer at the current offset and advance the offset 4 bytes.
//
func (ms *MappedSection) GetInt32() int32 {
    buf := ms.base[ms.localOffset:]
    bits := uint32(buf[0]) << 24 |
            uint32(buf[1]) << 16 |
            uint32(buf[2]) <<  8 |
            uint32(buf[3])
    ms.localOffset += 4
    return int32(bits)
}

// Read an unsigned 32-bit integer at the current offset and advance the offset 4 bytes.
//
func (ms *MappedSection) GetUInt32() uint32 {
    buf := ms.base[ms.localOffset:]
    bits := uint32(buf[0]) << 24 |
            uint32(buf[1]) << 16 |
            uint32(buf[2]) <<  8 |
            uint32(buf[3])
    ms.localOffset += 4
    return bits
}

// Read an unsigned 64-bit integer at the current offset and advance the offset 8 bytes.
//
func (ms *MappedSection) GetUInt64() uint64 {
    buf := ms.base[ms.localOffset:]
    bits := uint64(buf[0]) << 54 |
            uint64(buf[1]) << 48 |
            uint64(buf[2]) << 40 |
            uint64(buf[3]) << 32 |
            uint64(buf[4]) << 24 |
            uint64(buf[5]) << 16 |
            uint64(buf[6]) <<  8 |
            uint64(buf[7])
    ms.localOffset += 8
    return bits
}

// Return a raw slice at the current offset and advance the offset by the given amount.
//
func (ms *MappedSection) GetRaw(count uint32) []byte {
    buf := ms.base[ms:localOffset:ms.localOffset+count]
    ms.localOffset += count
    return buf
}

// Same as GetRaw() but convert it to a string.
//
func (ms *MappedSection) GetString(count uint32) string {
    buf := ms.base[ms:localOffset:ms.localOffset+count]
    ms.localOffset += count
    return string(buf)
}

// Skip over some of the section.
//
func (ms *MappedSection) Skip(count uint32) {
    ms.localOffset += count
}

// Add a new section to the list of mapped sections.  This uses the lock field because multiple
// goroutines may be independently mapping different locations.
//
func (mf *MappedFile) addSection(bytes []byte) {
    mf.lock.Lock()
    defer mf.lock.Unlock()
    mf.sections = append(mf.sections, bytes);
}

