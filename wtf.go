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
    "os"
)

var loggingTestOutput = false

// Wherein we write some functions that make me think, "WTF, Go -- why isn't this
// in a standard library?"

func IntMax(a int, b int) int {
    if (a > b) {
        return a
    }
    return b;
}

func IntAryReverse(a []int) {
    end := len(a) - 1
    if end > 0 {
        for start := 0; start < end; {
            a[start], a[end] = a[end], a[start]
            start++
            end--
        }
    }
}

func IntAryEq(a, b []int) bool {
    if len(a) != len(b) {
        return false
    }
    for i, x := range a {
        if x != b[i] {
            return false
        }
    }
    return true
}

// So, Go, what made you think I don't want to see any output at all when I run tests?
// WHAT IF I'M DEBUGGING?
//
func LogTestOutput() {
    if loggingTestOutput {
        return
    }
    console, err := os.OpenFile("./test.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
    if err != nil {
        log.Fatalln(err)
    }
    log.SetOutput(console)
    os.Stdout = console
    loggingTestOutput = true
}

