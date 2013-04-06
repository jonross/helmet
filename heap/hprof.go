package heap

import (
    "log"
    "os"
    "github.com/jonross/helmet/util"
)

type Heap struct {
    filename string
    file *os.File
    bytes []byte
}

func ReadHeap(filename string) (heap *Heap, err error) {

    heap = &Heap{filename: filename}

    heap.file, err = os.Open(heap.filename)
    if err != nil {
        return nil, err
    }
    defer heap.file.Close()

    heap.bytes, err = util.MMap(heap.file, 0)
    if err != nil {
        return nil, err
    }
    defer util.MUnmap(heap.bytes)

    heap.read(heap.bytes)
    return
}

func (heap *Heap) read(in []byte) {

    version := string(in[:19])
    if version != "JAVA PROFILE 1.0.1\000" && version != "JAVA PROFILE 1.0.2\000" {
        log.Fatalf("Unknown heap version %s\n", version)
    }

    refSize := util.GetInt32(in[19:23])
    if refSize != 4 && refSize != 8 {
        log.Fatalf("Unknown reference size %d\n", refSize)
    }

    // skip timestamp
    in = in[31:]

    for len(in) > 0 {
        // tag := in[0]
        length := util.GetUInt32(in[5:9])
        // log.Printf("Record of length %d\n", length)
        in = in[9+length:]
    }


}

    /*
            if (tag < 0 || tag >= recordHandlers.length)
                panic("Bad tag %d at input position %d".format(tag, data.position))
            val handler = recordHandlers(tag)
            if (handler == null)
                panic("Unknown tag %d at input position %d".format(tag, data.position))
            handler(length)
    */
