package main

import (
    "fmt"
    "./peg"
)

func hel() {

    dot := peg.Literal(".").Info(".")
    dotStar := peg.Literal(".*").Info(".*")
    digit := peg.AnyOf("0123456789").Info("digit")
    letter := peg.AnyOf("_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ").Info("letter")
    id := peg.Sequence(letter, peg.ZeroOrMoreOf(letter, digit)).Adjacent().Info("id")

    pkg := peg.Sequence(id, peg.ZeroOrMoreOf(peg.Sequence(dot, id)), peg.Optional(dotStar)).Adjacent().Info("package")

    pkg.Parse("foo")

}

func main() {

    digit := peg.AnyOf("0123456789").Info("digit")
    letter := peg.AnyOf("_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ").Info("letter")
    identifier := peg.Sequence(letter, peg.ZeroOrMoreOf(letter, digit)).Adjacent().Info("id")
    identifier.Handle(func (s string) { fmt.Print(s) })

    identifier.Parse("foobar")
    identifier.Parse("_xyz87")
    identifier.Parse("87_xyz")

    hel()
}


