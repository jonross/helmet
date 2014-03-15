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
    "math"
    "os"
    "syscall"
)

// MappedFile tracks a set of memory-mapped sections on a file and provides
// low-level data access like GetUInt32.  This is a helpful abstraction even
// given Go slices because unfortunately, UNIX cannot map > 2^31-1 bytes at
// a time, and we need to abstract away the case where code is reading data
// and suddenly finds a value that runs over the end of a section.
//
// TODO: unit test
//
type MappedFile struct {
    Filename string
    file *os.File
    // total size of the file
    Size uint64
}

// This represents a single mapped section on a MappedFile.
//
type MappedSection struct {
    // See above
    *MappedFile
    // As returned by syscall.Mmap
    base []byte
    // Size of mapped section
    size uint32
    // Offset of base within the file
    baseOffset uint64
    // Offset of current location from baseOffset
    localOffset uint32
}

//////////////////////////////////////////////////////////////////////////////////////////

// Create a MappedFile.
//
func MapFile(filename string) (mf *MappedFile, err error) {
    file, err := os.Open(filename)
    if err != nil {
        file.Close()
        return nil, err
    }
    info, err := file.Stat()
    if err != nil {
        file.Close()
        return nil, err
    }
    return &MappedFile{Filename: filename, file: file, Size: uint64(info.Size())}, nil
}

// Map the largest possible section starting at a given offset.  Normally this is called 
// automatically by Demand().  Panics on failure since there is no recovery.
//
func (mf *MappedFile) MapAt(offset uint64) *MappedSection {
    ms := &MappedSection{MappedFile: mf}
    ms.remapAt(offset)
    return ms
}

// Closes the file.  TODO: unmap all open sections
//
func (mf *MappedFile) Close() {
    mf.file.Close()
}

//////////////////////////////////////////////////////////////////////////////////////////

// Used by MappedFile.MapAt and MappedSection.Demand
//
func (ms *MappedSection) remapAt(offset uint64) {
    if ms.base != nil {
        ms.unmap()
    }
    // force alignment
    skew := offset % 8192
    offset = offset - skew
    // how much of the file is left & how much can we get
    length := ms.Size - offset
    if length > uint64(math.MaxInt32) {
        length = uint64(math.MaxInt32)
    }
    bytes, err := syscall.Mmap(int(ms.file.Fd()), int64(offset), int(length),
                               syscall.PROT_READ, syscall.MAP_SHARED)
    if err != nil {
        log.Fatalf("Can't map %s %d bytes at %d: %s\n", ms.Filename, length, offset, err)
    }
    ms.base = bytes
    ms.size = uint32(length)
    ms.baseOffset = offset
    // restore what we subtracted from the requested offset
    ms.localOffset = uint32(skew)
}

// Require at least count bytes in the current mapped section.  If not available, remap
// it from the current location.  Note only checks insufficient bytes in the section, not
// the file itself, so that a caller can make a conservative overestimate rather than
// calling Demand() more frequently with smaller amounts.  Returns false if no more data
// available.
//
func (ms *MappedSection) Demand(count uint32) bool {
    /*
        val overrun = nbytes - mapped.remaining
        if (overrun > 0)
            remap(offset + mapped.position)
    */
    remain := ms.size - ms.localOffset
    if remain >= count {
        return true
    }
    newOffset := ms.baseOffset + uint64(ms.localOffset)
    if newOffset < ms.Size {
        ms.remapAt(newOffset)
        return true
    }
    return false
}

// Unmap the section.
//
func (ms *MappedSection) unmap() {
    err := syscall.Munmap(ms.base)
    if err != nil {
        log.Fatalf("Failed to unmap %s at %x: %s\n", ms.Filename, ms.baseOffset, err)
    }
}

// Read a byte at the current offset and advance the offset 1 byte.
//
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
//
func (ms *MappedSection) GetByte() byte {
    ret := ms.base[ms.localOffset]
    ms.localOffset++
    return ret
}

// Read an unsigned 16-bit integer at the current offset and advance the offset 2 bytes.
//
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
//
func (ms *MappedSection) GetUInt16() uint16 {
    buf := ms.base[ms.localOffset:]
    bits := uint16(buf[0]) << 8 |
            uint16(buf[1]) 
    ms.localOffset += 2
    return bits
}

// Read a signed 32-bit integer at the current offset and advance the offset 4 bytes.
//
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
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
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
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
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
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
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
//
func (ms *MappedSection) GetRaw(count uint32) []byte {
    buf := ms.base[ms.localOffset:ms.localOffset+count]
    ms.localOffset += count
    return buf
}

// Same as GetRaw() but convert it to a string.
//
// NOTE: this is not safe to call unless Demand() has already accounted for the
// required length.
//
func (ms *MappedSection) GetString(count uint32) string {
    buf := ms.base[ms.localOffset:ms.localOffset+count]
    ms.localOffset += count
    return string(buf)
}

// Skip over some of the section.
//
func (ms *MappedSection) Skip(count uint32) {
    // TODO account for Demand
    ms.localOffset += count
    /*
        val overrun = nbytes - mapped.remaining
        if (overrun > 0)
            remap(offset + mapped.limit + overrun)
        else
            mapped position(mapped.position + nbytes.asInstanceOf[Int])
    */
}

// Return the global file offset
//
func (ms *MappedSection) Offset() uint64 {
    return ms.baseOffset + uint64(ms.localOffset)
}

