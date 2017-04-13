package main

import (
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		local      *FileStat
		remote     *FileStat
		shouldSync bool
	}{
		{
			local:      &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			shouldSync: false,
		},
		{
			local:      &FileStat{Path: "file2.html", Size: 1, ModTime: time.Time{}},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			shouldSync: true,
		},
		{
			local:      &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			remote:     &FileStat{Path: "file.html", Size: 2, ModTime: time.Time{}},
			shouldSync: true,
		},
		{
			local:      &FileStat{Path: "file.html", Size: 2, ModTime: time.Time{}},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			shouldSync: true,
		},
		{
			local:      &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Now()},
			shouldSync: false,
		},
		{
			local:      &FileStat{Path: "file.html", Size: 1, ModTime: time.Now()},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Time{}},
			shouldSync: true,
		},
		{
			local:      &FileStat{Path: "file.html", Size: 1, ModTime: time.Now()},
			remote:     &FileStat{Path: "file.html", Size: 1, ModTime: time.Now().Add(-time.Minute)},
			shouldSync: true,
		},
	}

	for _, test := range tests {
		logger, buf := getTestLogger()
		localFiles := make(chan *FileStat)
		remoteFiles := make(chan *FileStat)
		go func() {
			localFiles <- test.local
			close(localFiles)
		}()
		go func() {
			remoteFiles <- test.remote
			close(remoteFiles)
		}()
		files := compare(localFiles, remoteFiles, logger)
		var updates []*FileStat
		for f := range files {
			updates = append(updates, f)
		}
		actual := len(updates) > 0
		if actual != test.shouldSync {
			t.Errorf("Expected sync %t, but got %t\n", test.shouldSync, actual)
			t.Errorf("%s\n", updates[0])
			t.Errorf("%s\n", buf)
		}

	}
}
