package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadLocalFiles(basePath string, exclude StringSlice, logger *Logger) (chan *FileStat, error) {

	out := make(chan *FileStat)

	regulatedPath := filepath.ToSlash(basePath)

	if err := checkPathIsDir(regulatedPath); err != nil {
		return out, err
	}

	getFile := func(filePath string, stat os.FileInfo, err error) error {
		relativePath := relativePath(regulatedPath, filepath.ToSlash(filePath))
		for _, pattern := range exclude {
			if globMatch(pattern, relativePath) {
				logger.Debug.Printf("excluding %s\n", relativePath)
				if stat.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if stat.IsDir() {
			return nil
		}

		out <- &FileStat{
			Path:    relativePath,
			ModTime: stat.ModTime(),
			Size:    stat.Size(),
		}
		return nil
	}

	go func() {
		start := time.Now()
		logger.Debug.Printf("read local - start at %s", start)
		if err := filepath.Walk(basePath, getFile); err != nil {
			logger.Err.Println(err)
		}
		logger.Debug.Printf("read local - end, it took %s", time.Since(start))
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
	}
	return strings.TrimPrefix(strings.TrimPrefix(filePath, path), "/")
}
