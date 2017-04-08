package main

import "time"

type LocalFileResult struct {
	err  error
	file *File
}

type File struct {
	path  string
	size  int64
	mtime time.Time
}
