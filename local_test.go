package main

import (
	"bytes"
	"log"
	"testing"
)

func TestLoadAllLocalFiles(t *testing.T) {
	logger, buf := getTestLogger()

	var exclude stringSlice
	fileChan, err := loadLocalFiles("./_testdata", exclude, logger)
	if err != nil {
		t.Error(err)
		return
	}

	files, err := sink(fileChan)
	if err != nil {
		t.Error(err)
		return
	}

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
	logger, buf := getTestLogger()
	var exclude stringSlice
	fileChan, err := loadLocalFiles("./_testdata/dir_45", exclude, logger)

	if err != nil {
		t.Error(err)
		return
	}

	files, err := sink(fileChan)
	if err != nil {
		t.Error(err)
		return
	}

	expected := 13
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}
}

func TestLoadNonExistingDirShouldFail(t *testing.T) {
	logger, _ := getTestLogger()
	var exclude stringSlice
	_, err := loadLocalFiles("./_testdata/XXX_SDASD", exclude, logger)
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
	logger, _ := getTestLogger()
	var exclude stringSlice
	_, err := loadLocalFiles("./_testdata/file_33.html", exclude, logger)
	if err == nil {
		t.Error("Expected an error")
		return
	}
	expected := "./_testdata/file_33.html is not a directory"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestLoadFilesExcludeAll(t *testing.T) {
	logger, buf := getTestLogger()
	exclude := stringSlice{"_testdata*"}
	fileChan, err := loadLocalFiles("./_testdata", exclude, logger)
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		return
	}

	files, err := sink(fileChan)
	if err != nil {
		t.Error(err)
		return
	}

	expected := 0
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}
}

func TestLoadFilesExcludeHTML(t *testing.T) {
	logger, buf := getTestLogger()
	exclude := stringSlice{"*.html"}
	fileChan, err := loadLocalFiles("./_testdata", exclude, logger)
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		return
	}

	files, err := sink(fileChan)
	if err != nil {
		t.Error(err)
		return
	}

	expected := 11
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}
}

func TestLoadFilesExclude70(t *testing.T) {
	logger, buf := getTestLogger()
	exclude := stringSlice{"*dir_45*"}
	fileChan, err := loadLocalFiles("./_testdata", exclude, logger)
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		return
	}

	files, err := sink(fileChan)
	if err != nil {
		t.Error(err)
		return
	}

	expected := 6
	actual := len(files)
	if actual != expected {
		t.Errorf("wanted %d files, got %d files", expected, actual)
		t.Errorf("%s\n", buf)
	}
}

func sink(in chan LocalFileResult) (map[string]*File, error) {
	out := make(map[string]*File)
	for f := range in {
		if f.err != nil {
			return out, f.err
		}
		out[f.file.path] = f.file
	}
	return out, nil
}

func getTestLogger() (*Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	return &Logger{
		Out:   log.New(buf, "[Out] ", log.Lshortfile),
		Err:   log.New(buf, "[Err] ", log.Lshortfile),
		Debug: log.New(buf, "[DEBUG] ", log.Lshortfile),
	}, buf
}
