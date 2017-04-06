package main

import (
	"bytes"
	"log"
	"testing"
)

func TestLoadAllLocalFiles(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.New(buf, "[TEST] ", log.Lshortfile)
	fileChan, err := loadLocalFiles("./_testdata", logger)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	files := <-fileChan

	expected := 19
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}

	filename := "_testdata/file_33.html"
	file, ok := files[filename]
	if !ok {
		t.Errorf("Couldn't find file '%s' in file list", filename)
		return
	}

	if file.path != filename {
		t.Errorf("expected file.path ('%s') to be the same as the key ('%s') of the map", file.path, filename)
	}
}

func TestLoadSomeLocalFiles(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.New(buf, "[TEST] ", log.Lshortfile)
	fileChan, err := loadLocalFiles("./_testdata/dir_45", logger)

	if err != nil {
		t.Errorf(err.Error())
		return
	}

	files := <-fileChan

	expected := 13
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}
}

func TestLoadNonExistingDirShouldFail(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.New(buf, "[TEST] ", log.Lshortfile)
	_, err := loadLocalFiles("./_testdata/XXX_SDASD", logger)
	if err == nil {
		t.Error("Expected an error")
		return
	}
	expected := "stat ./_testdata/XXX_SDASD: no such file or directory"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestLoadFileShouldFail(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.New(buf, "[TEST] ", log.Lshortfile)
	_, err := loadLocalFiles("./_testdata/file_33.html", logger)
	if err == nil {
		t.Error("Expected an error")
		return
	}
	expected := "./_testdata/file_33.html is not a directory"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}