package main

import (
	"fmt"
	"log"
	"time"
)

// Logger wraps loggers for stdout, stderr and debug output
type Logger struct {
	Out   *log.Logger
	Err   *log.Logger
	Debug *log.Logger
}

// A FileStat describes a local and remote file and can contain an error if the information
// was not possible to get
type FileStat struct {
	Err     error
	Path    string
	Size    int64
	ModTime time.Time
}

// StringSlice is usable for being able to use multiple flags of the same value, example:  -exclude "file1" -exclude "file2"
type StringSlice []string

// String is the method to format the flag's value, part of the flag.Value interface. The String method's output will be
// used in diagnostics.
func (s *StringSlice) String() string {
	return fmt.Sprint(*s)
}

// Set is the method to set the flag value, part of the flag.Value interface. Set's argument is a string to be parsed to
// set the flag. It's a comma-separated list, so we split it.
func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}