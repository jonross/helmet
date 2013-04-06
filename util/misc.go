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

