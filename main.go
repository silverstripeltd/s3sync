package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var dryrun bool
var verbose bool

// @todo, ignore files beginning with .
func main() {

	if os.Getenv("DRYRUN") != "" {
		dryrun = true
	}
	if os.Getenv("VERBOSE") != "" {
		verbose = true
	}

	debug := log.New(os.Stdout, "[DEBUG] ", log.Ltime)
	if verbose == false {
		debug.SetOutput(ioutil.Discard)
	}

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	localPath := os.Args[1]
	bucket := os.Args[2]
	bucketPath := os.Args[3]

	localFiles := loadLocalFiles(localPath, debug)
	s3Files := loadS3Files(svc, bucket, bucketPath, debug)

	foundLocal := <-localFiles
	debug.Printf("found %d local files", len(foundLocal))
	foundRemote := <-s3Files
	debug.Printf("found %d s3 files", len(foundRemote))

	files := compare(foundLocal, foundRemote, debug)

	syncFiles(svc, bucket, bucketPath, localPath, files, debug)

}

/**
 this is the A local file will require uploading if:
- the size of the local file is different than the size of the s3 object
- the last modified time of the local file is newer than the last modified time of the s3 object
- the local file does not exist under the specified bucket and prefix.
*/
func compare(foundLocal, foundRemote map[string]*File, debug *log.Logger) chan *File {

	update := make(chan *File, 8)

	go func() {
		for path, local := range foundLocal {
			remote := foundRemote[path]
			if remote == nil {
				debug.Printf("404 - %s need upload\n", path)
				update <- local
			} else if local.size != remote.size {
				debug.Printf("size different - %s need upload\n", path)
				update <- local
			} else if local.mtime.After(remote.mtime) {
				debug.Printf("mtime different - %s need upload\n", path)
				update <- local
			}
		}
		close(update)
	}()
	return update
}

func syncFiles(svc *s3.S3, bucket, bucketPath, localPath string, in chan *File, debug *log.Logger) {

	concurrency := 5
	sem := make(chan bool, concurrency)

	for file := range in {
		// add one
		sem <- true
		go func(svc *s3.S3, bucket, bucketPath, localPath string, file *File, debug *log.Logger) {
			upload(svc, bucket, bucketPath, localPath, file, debug)
			// remove one
			<-sem
		}(svc, bucket, bucketPath, localPath, file, debug)
	}

	// After the last goroutine is fired, there are still concurrency amount of goroutines running. In order to make
	// sure we wait for all of them to finish, we attempt to fill the semaphore back up to its capacity. Once that
	// succeeds, we know that the last goroutine has read from the semaphore, as we've done len(urls) + cap(sem) writes
	// and len(urls) reads off the channel.
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func upload(svc *s3.S3, bucket, bucketPath, localPath string, file *File, debug *log.Logger) {

	debug.Printf("opening file %s\n", localPath+file.path)
	lFile, err := os.Open(localPath + file.path)
	if err != nil {
		fmt.Printf("err opening file: %s", err)
	}
	defer lFile.Close()

	// create a byte buffer reader for the content of the local file
	buffer := make([]byte, file.size)
	lFile.Read(buffer)
	fileBody := bytes.NewReader(buffer)

	fileType := http.DetectContentType(buffer)

	key := filepath.Join(bucketPath, file.path)

	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          fileBody,
		ContentLength: aws.Int64(file.size),
		ContentType:   aws.String(fileType),
	}

	if !dryrun {
		debug.Printf("start upload of %s\n", file.path)
		_, err = svc.PutObject(params)
		if err != nil {
			fmt.Printf("bad response: %s", err)
			return
		}
		debug.Printf("upload done of %s\n", file.path)
	} else {
		fmt.Printf("Would have uploaded %s\n", file.path)
	}

}
