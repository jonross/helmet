package util

import (
    "os"
    "syscall"
)

type MappedFile {
    filename string
    file os.File
    size int64
    sections map[offset uint64][]byte
}

type MappedSection {
    mf *MappedFile
    size int32
    base []byte
    cur []byte
}

func mapFile(filename string) *MappedFile, err {

    mf = &MappedFile{filename: filename}

    mf.file, err = os.Open(filename)
    if err != nil {
        return nil, err
    }

    info, err := file.Stat()
    if err != nil {
        return nil, err
    }

}

func 

    mf.size = info.Size()
    if size > int64(math.MaxInt32) {
        size = int64(math.MaxInt32)
    }

    return syscall.Mmap(int(file.Fd()), offset, int(size),
                           syscall.PROT_READ, syscall.MAP_SHARED)
}

func (mf *MappedFile) close() {
    for _, section := range mf.sections {
        syscall.Munmap(section)
    }
}

func (mf *MappedFile) RemapAt(offset uint64) {
    
}

