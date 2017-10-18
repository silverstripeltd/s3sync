package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// Logger wraps loggers for stdout, stderr and debug output
type Logger struct {
	Out   *log.Logger
	Err   *log.Logger
	Debug *log.Logger
}

// NewLogger creates a new Logger ready for use
func NewLogger(debug, onlyShowErrors bool) *Logger {
	l := &Logger{
		Out:   log.New(os.Stdout, "", 0),
		Err:   log.New(os.Stderr, "", 0),
		Debug: log.New(os.Stdout, "[DEBUG] ", 0),
	}
	if !debug {
		l.Debug.SetOutput(ioutil.Discard)
	}

	if onlyShowErrors {
		l.Debug.SetOutput(ioutil.Discard)
		l.Out.SetOutput(ioutil.Discard)
	}
	return l
}

// Config contains common paths and configuration
type Config struct {
	S3Service    s3iface.S3API
	Bucket       string
	BucketPrefix string
	DryRun       bool
}

// A FileStat describes a local and remote file and can contain an error if the information
// was not possible to get
type FileStat struct {
	Err     error
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
}

func (f *FileStat) String() string {
	if f.Err != nil {
		return fmt.Sprintf("%v", f.Err)
	}
	return fmt.Sprintf("%s size: %d, mtime: %s", f.Name, f.Size, f.ModTime)
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
