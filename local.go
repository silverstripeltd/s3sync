package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func loadLocalFiles(basePath string, exclude StringSlice, logger *Logger) chan *FileStat {

	out := make(chan *FileStat)

	regulatedPath := filepath.ToSlash(basePath)

	go func() {
		defer close(out)
		start := time.Now()
		logger.Debug.Printf("read local - start at %s", start)

		stat, err := os.Stat(regulatedPath)
		if err != nil {
			logger.Err.Printf("Error while reading local files: %v\n", err)
			return
		}

		if !stat.IsDir() {
			out <- &FileStat{
				Path:    basePath,
				ModTime: stat.ModTime(),
				Size:    stat.Size(),
			}
			return
		}

		err = filepath.Walk(basePath, func(filePath string, stat os.FileInfo, err error) error {
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
			if stat == nil || stat.IsDir() {
				return nil
			}
			out <- &FileStat{
				Path:    relativePath,
				ModTime: stat.ModTime(),
				Size:    stat.Size(),
			}
			return nil
		})
		if err != nil {
			logger.Err.Println(err)
		}

		logger.Debug.Printf("read local - end, it took %s", time.Since(start))
	}()

	return out
}

func relativePath(path string, filePath string) string {
	if path == "." {
		return strings.TrimPrefix(filePath, "/")
	}
	return strings.TrimPrefix(strings.TrimPrefix(filePath, path), "/")
}
