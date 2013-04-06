package main

import (
    "flag"
    "log"
    "github.com/jonross/helmet/heap"
)

func main() {

    flag.Parse()
    args := flag.Args()

    switch {
        case len(args) == 0:
            log.Fatal("Missing heap filename")
        case len(args) > 1:
            log.Fatal("Extra args following heap filename")
    }

    _, err := heap.ReadHeap(flag.Arg(0))
    if err != nil {
        log.Fatal(err)
    }
}

