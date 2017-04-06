package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadLocalFiles(basePath string, debug *log.Logger) (chan map[string]*File, error) {

	out := make(chan map[string]*File)

	files := make(map[string]*File)
	regulatedPath := filepath.ToSlash(basePath)

	if err := checkPathIsDir(regulatedPath); err != nil {
		return out, err
	}

	getFile := func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		p := relativePath(regulatedPath, filepath.ToSlash(filePath))

		stat, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		files[p] = &File{
			path:  p,
			mtime: stat.ModTime(),
			size:  stat.Size(),
		}
		return nil
	}

	go func() {
		start := time.Now()
		debug.Printf("read local - start at %s", start)
		err := filepath.Walk(basePath, getFile)
		debug.Printf("read local - end, it took %s", time.Now().Sub(start))
		if err != nil {
			fmt.Println(err)
		}
		out <- files
		close(out)
	}()

	return out, nil
}

// checkPathIsDir will check that path is an existing directory, will return an error otherwise
func checkPathIsDir(path string) error {
	finfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !finfo.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func relativePath(path string, filePath string) string {
	if path == "." {
		return strings.TrimPrefix(filePath, "/")
	} else {
		return strings.TrimPrefix(strings.TrimPrefix(filePath, path), "/")

	}
}
