package main

import "time"

type File struct {
	path  string
	size  int64
	mtime time.Time
}
