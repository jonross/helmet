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
    "os"
    "strings"
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
        if strings.Trim(line, " \t") == "" {
            continue
        }
        gnureadline.AddHistory(line)
        session.run(line)
    }
}

// Execute a console command.
//
func (session *Session) run(command string) {
    parsers := NewParsers()
    ok, _, result := parsers.Command.Parse(command)
    if ok {
        action := result.(Action)
        action.Run(session)
    } else {
        // TODO better error messages
        fmt.Printf("Syntax error in command\n")
    }
}

// Execute a search (called from generated parser function.)
//
func (session *Session) runSearch(query *Query) {
    histo := session.Heap.NewHisto()
    SearchHeap(session.Heap, query, histo)
    histo.Print(os.Stdout)
}

// Create map of default session settings.
//
func DefaultSettings() map[string]*Setting {
    settings := make(map[string]*Setting)
    settings["mingroupsize"] = &Setting{"", 1 << 20}
    return settings
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// A user action that can be executed in a session.  This is an interface type, with one struct
// defined per action, rather than just a function pointer, so that it's easier to test.
//
type Action interface {
    Run(*Session)
}

type SearchAction struct {
    Query *Query
}

func (action SearchAction) Run(session *Session) {
    session.runSearch(action.Query)
}

type SettingsAction struct {
    Name string
    Value int
}

func (action SettingsAction) Run(session *Session) {
    session.Settings[action.Name].IntValue = action.Value
}

type ErrorAction struct {
    Error error
}

func (action ErrorAction) Run(session *Session) {
    fmt.Println(action.Error.Error())
}
