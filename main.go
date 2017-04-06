package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var dryrun bool
var quiet bool
var debug bool

func init() {
	flag.BoolVar(&dryrun, "dryrun", false, "Displays the operations that would be performed using the specified command without actually running them.")
	flag.BoolVar(&quiet, "quiet", false, "Does not display the operations performed from the specified command")
	flag.BoolVar(&debug, "debug", false, "Turn on debug logging")

}

// @todo, ignore files beginning with .
func main() {

	flag.Parse()

	debugLogger := log.New(os.Stdout, "[DEBUG] ", log.Ltime)
	if !debug {
		debugLogger.SetOutput(ioutil.Discard)
	}

	// @todo, check that localPath exists and is readable
	path, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		flag.Usage()
		fmt.Printf("\nCould not parse LocalPath '%s': %s\n", flag.Arg(0), err)
		os.Exit(1)
	}
	S3Uri := flag.Arg(1)

	t, err := url.Parse(S3Uri)
	if err != nil {
		flag.Usage()
		fmt.Printf("\nCould not parse S3Uri '%s'\n", S3Uri)
		os.Exit(1)
	}
	if t.Scheme != "s3" {
		flag.Usage()
		fmt.Println("\nS3Uri argument does not have valid protocol, should be 's3'")
		os.Exit(1)
	}
	if t.Host == "" {
		flag.Usage()
		fmt.Println("\nS3Uri is missing bucket name")
		os.Exit(1)
	}

	bucket := t.Host
	bucketPath := t.Path

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	sess := session.Must(session.NewSession())
	if debug {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebugWithRequestErrors)
	}
	svc := s3.New(sess)
	localFiles := loadLocalFiles(path, debugLogger)
	s3Files := loadS3Files(svc, bucket, bucketPath, debugLogger)

	foundLocal := <-localFiles
	debugLogger.Printf("found %d local files", len(foundLocal))

	foundRemote := <-s3Files
	debugLogger.Printf("found %d s3 files", len(foundRemote))

	files := compare(foundLocal, foundRemote, debugLogger)
	syncFiles(svc, bucket, bucketPath, path, files, debugLogger)

}

/**
 this is the A local file will require uploading if:
- the size of the local file is different than the size of the s3 object
- the last modified time of the local file is newer than the last modified time of the s3 object
- the local file does not exist under the specified bucket and prefix.

see https://github.com/aws/aws-cli/blob/e2295b022db35eea9fec7e6c5540d06dbd6e588b/awscli/customizations/s3/syncstrategy/base.py#L226
*/
func compare(foundLocal, foundRemote map[string]*File, debug *log.Logger) chan *File {

	update := make(chan *File, 8)

	go func() {
		for path, local := range foundLocal {
			remote := foundRemote[path]
			if remote == nil {
				debug.Printf("syncing: %s, file does not exist at destination\n", path)
				update <- local
			} else if local.size != remote.size {
				debug.Printf("syncing: %s, size %d -> %d\n", path, local.size, remote.size)
				update <- local
			} else if local.mtime.After(remote.mtime) {
				debug.Printf("syncing: %s, modified time: %s -> %s\n", path, local.mtime, remote.mtime.In(local.mtime.Location()))
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

	realFile, err := os.Open(filepath.Join(localPath, file.path))
	if err != nil {
		fmt.Printf("error opening file: %s\n", err)
		return
	}
	defer realFile.Close()

	// create a byte buffer reader for the content of the local file
	buffer := make([]byte, file.size)
	realFile.Read(buffer)
	fileBody := bytes.NewReader(buffer)
	fileType := http.DetectContentType(buffer)

	key := filepath.Join(bucketPath, file.path)
	key = strings.TrimPrefix(key, "/")

	// Create an uploader (can do multipart) with S3 client and default options
	uploader := s3manager.NewUploaderWithClient(svc)
	params := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   fileBody,
		//ContentLength: aws.Int64(file.size),
		ContentType: aws.String(fileType),
	}

	//params := &s3.PutObjectInput{
	//	Bucket:        aws.String(bucket),
	//	Key:           aws.String(key),
	//	Body:          fileBody,
	//	ContentLength: aws.Int64(file.size),
	//	ContentType:   aws.String(fileType),
	//}

	s3Uri := filepath.Join(bucket, key)
	if !dryrun {
		debug.Printf("start upload of %s\n", file.path)
		_, err := uploader.Upload(params)
		//_, err := svc.PutObject(params)
		if err != nil {
			fmt.Printf("bad response: %+v", err)
			return
		}
		fmt.Printf("upload: %s to s3://%s\n", file.path, s3Uri)
	} else {
		fmt.Printf("(dryrun) upload: %s to s3://%s\n", file.path, s3Uri)
	}

}
