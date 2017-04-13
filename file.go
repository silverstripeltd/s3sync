package main

import "time"

// A FileStat describes a local and remote file and can contain an error if the information
// was not possible to get
type FileStat struct {
	Err     error
	Path    string
	Size    int64
	ModTime time.Time
}
