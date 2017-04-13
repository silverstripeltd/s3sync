package main

import (
	"bytes"
	"log"
	"testing"
)

func TestLoadAllLocalFiles(t *testing.T) {
	logger, buf := getTestLogger()

	var exclude StringSlice
	fileChan := loadLocalFiles("./_testdata", exclude, logger)

	files := sink(fileChan)

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

	if file.Err != nil {
		t.Errorf("Expected file.Err to be nil, got %v\n", file.Err)
	}
	if file.Path != filename {
		t.Errorf("Expected file.Path to be %s, got %s\n", filename, file.Path)
	}
	if file.Size != 34 {
		t.Errorf("Expected file.Path to be %d, got %d\n", 34, file.Size)
	}

	if file.Path != filename {
		t.Errorf("expected file.path ('%s') to be the same as the key ('%s') of the map", file.Path, filename)
	}
}

func BenchmarkLoadAllLocalFiles(b *testing.B) {
	logger, _ := getTestLogger()
	var exclude StringSlice
	for i := 0; i < b.N; i++ {
		loadLocalFiles("./_testdata", exclude, logger)
	}
}

func TestLoadSingleFile(t *testing.T) {
	logger, buf := getTestLogger()
	fileChan := loadLocalFiles("./_testdata/file_33.html", StringSlice{}, logger)
	files := sink(fileChan)
	if len(files) != 1 {
		t.Errorf("wanted %d files, got %d files", 1, len(files))
		t.Errorf("%s\n", buf)
	}

	filename := "./_testdata/file_33.html"
	_, ok := files[filename]
	if !ok {
		t.Errorf("Couldn't find file '%s' in file list", filename)
		t.Errorf("%+v", files)
		return
	}
}

func TestLoadFiles(t *testing.T) {
	tests := []struct {
		in      string
		out     int
		exclude StringSlice
	}{
		{in: "./_testdata/dir_45", out: 13},
		{in: "./_testdata/dir_45/", out: 13},
		{in: "./_testdata/XXX_SDASD", out: 0},
		{in: "./_testdata/file_33.html", out: 1},
		{in: "./_testdata", out: 0, exclude: StringSlice{"_testdata*"}},
		{in: "./_testdata", out: 11, exclude: StringSlice{"*.html"}},
		{in: "./_testdata", out: 11, exclude: StringSlice{"*.html"}},
		{in: "./_testdata", out: 6, exclude: StringSlice{"*dir_45*"}},
	}

	for _, test := range tests {
		logger, buf := getTestLogger()
		fileChan := loadLocalFiles(test.in, test.exclude, logger)
		files := sink(fileChan)
		if len(files) != test.out {
			t.Errorf("wanted %d files, got %d files", test.out, len(files))
			t.Errorf("%s\n", buf)
		}
	}
}

func sink(in chan *FileStat) map[string]*FileStat {
	out := make(map[string]*FileStat)
	for f := range in {
		if f.Err != nil {
			return out
		}
		out[f.Path] = f
	}
	return out
}

func getTestLogger() (*Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	return &Logger{
		Out:   log.New(buf, "[Out] ", log.Lshortfile),
		Err:   log.New(buf, "[Err] ", log.Lshortfile),
		Debug: log.New(buf, "[DEBUG] ", log.Lshortfile),
	}, buf
}
