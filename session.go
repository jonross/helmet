/*
    Copyright (c) 2013 by Jonathan Ross (jonross@alum.mit.edu)

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
    "fmt"
    "code.google.com/p/go-gnureadline"
    "io"
    "log"
)

// Global state for user interaction.
//
type Session struct {
    *Heap
    Settings map[string]*Setting
}

// Session settings, may be changed with the "set" command.
//
type Setting struct {
    StringValue string
    IntValue int
}

// Start interactive console session.  Returns on console EOF or
// quit/exit action.
//
func (session *Session) interact() {
    for {
        line, err := gnureadline.Readline("> ")
        if err == io.EOF {
            return
        }
        if err != nil {
            log.Fatalf("readline error: %s\n", err)
        }
        fmt.Printf("i got '%s'\n", line)
        gnureadline.AddHistory(line)
        session.run(line)
    }
}

// Execute a console command.
//
func (session *Session) run(command string) {
    parsers := NewParsers()
    _, _, result := parsers.Command.Debug(4).Parse(command)
    action := result.(func(*Session))
    action(session)
}

// Create map of default session settings.
//
func DefaultSettings() map[string]*Setting {
    settings := make(map[string]*Setting)
    settings["mingroupsize"] = &Setting{"", 1 << 20}
    return settings
}
