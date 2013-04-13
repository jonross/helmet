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
    "math"
    "os"
    "syscall"
)

// MMap memory-maps a portion of a file using syscall.Mmap, up to a maximum
// of math.MaxInt32 bytes (the limit imposed by mmap.)
//
func MMap(file *os.File, offset int64) ([]byte, error) {

    info, err := file.Stat()
    if err != nil {
        return nil, err
    }

    size := info.Size()
    if size > int64(math.MaxInt32) {
        size = int64(math.MaxInt32)
    }

    return syscall.Mmap(int(file.Fd()), offset, int(size),
                           syscall.PROT_READ, syscall.MAP_SHARED)
}

// MUnmap unmaps a mapped file read with MMap.
//
func MUnmap(data []byte) error {
    return syscall.Munmap(data)
}

