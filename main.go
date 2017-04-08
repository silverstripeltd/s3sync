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
var debug bool
var onlyShowErrors bool
var exclude stringSlice

func init() {
	flag.BoolVar(&dryrun, "dryrun", false, "Displays the operations that would be performed using the specified command without actually running them.")
	flag.BoolVar(&debug, "debug", false, "Turn on debug logging")
	flag.BoolVar(&onlyShowErrors, "only-show-errors", false, "Only errors and  warnings  are  displayed. All other output is suppressed.")
	flag.Var(&exclude, "exclude", "Exclude all files or objects from the command that matches the specified pattern, only supports '*' globbing")
}

type Logger struct {
	Out   *log.Logger
	Err   *log.Logger
	Debug *log.Logger
}

func main() {

	flag.Parse()

	logger := &Logger{
		Out:   log.New(os.Stdout, "", 0),
		Err:   log.New(os.Stderr, "", 0),
		Debug: log.New(os.Stdout, "[DEBUG] ", 0),
	}

	if !debug {
		logger.Debug.SetOutput(ioutil.Discard)
	}

	if onlyShowErrors {
		logger.Debug.SetOutput(ioutil.Discard)
		logger.Out.SetOutput(ioutil.Discard)
	}

	path, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		flag.Usage()
		logger.Err.Printf("\nCould not parse LocalPath '%s': %s\n", flag.Arg(0), err)
		os.Exit(1)
	}
	S3Uri := flag.Arg(1)

	t, err := url.Parse(S3Uri)
	if err != nil {
		flag.Usage()
		logger.Err.Printf("\nCould not parse S3Uri '%s'\n", S3Uri)
		os.Exit(1)
	}
	if t.Scheme != "s3" {
		flag.Usage()
		logger.Err.Println("\nS3Uri argument does not have valid protocol, should be 's3'")
		os.Exit(1)
	}
	if t.Host == "" {
		flag.Usage()
		logger.Err.Println("\nS3Uri is missing bucket name")
		os.Exit(1)
	}

	bucket := t.Host
	bucketPath := t.Path

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	sess := session.Must(session.NewSession())
	if debug {
		//sess.Config.LogLevel = aws.LogLevel(aws.LogDebugWithRequestErrors)
	}

	svc := s3.New(sess)
	local, err := loadLocalFiles(path, exclude, logger)
	if err != nil {
		logger.Err.Printf("\n%s\n", err)
		// stop goroutines
		os.Exit(1)
	}

	// we keep 50,000 (50 s3:listObjects calls) to be in the output remote channel,
	// this will ensure that we can find all local files without blocking the AWS calls
	remote, err := loadS3Files(svc, bucket, bucketPath, 50000, logger)
	if err != nil {
		logger.Err.Printf("\n%s\n", err)
		// stop goroutines
		os.Exit(1)
	}

	files := compare(local, remote, logger)
	syncFiles(svc, bucket, bucketPath, path, files, logger)
}

// compare will put a local file on the output channel if:
// - the size of the local file is different than the size of the s3 object
// - the last modified time of the local file is newer than the last modified time of the s3 object
// - the local file does not exist under the specified bucket and prefix.
// This is the same logic as the aws s3 sync tool uses, see https://github.com/aws/aws-cli/blob/e2295b022db35eea9fec7e6c5540d06dbd6e588b/awscli/customizations/s3/syncstrategy/base.py#L226
func compare(foundLocal, foundRemote chan LocalFileResult, logger *Logger) chan *File {

	update := make(chan *File, 8)

	// first we sink the local files into a lookup map so its quick and easy to compare that to the remote
	localFiles := make(map[string]*File)
	for r := range foundLocal {
		if r.err != nil {
			logger.Out.Println(r.err)
			continue
		}
		localFiles[r.file.path] = r.file
	}

	numLocalFiles := len(localFiles)
	var numRemoteFiles int

	go func() {
		for remote := range foundRemote {
			numRemoteFiles++
			if remote.err != nil {
				logger.Out.Println(remote.err)
				continue
			}

			// see if there is a local file that matches the remote file
			var local *File
			var ok bool
			// check if the remote have a local representation
			if local, ok = localFiles[remote.file.path]; !ok {
				continue
			}
			// we "handled" this local file now
			delete(localFiles, remote.file.path)
			// check if we need to update this file
			if local.size != remote.file.size {
				logger.Debug.Printf("syncing: %s, size %d -> %d\n", local.path, local.size, remote.file.size)
				update <- local
			} else if local.mtime.After(remote.file.mtime) {
				logger.Debug.Printf("syncing: %s, modified time: %s -> %s\n", local.path, local.mtime, remote.file.mtime.In(local.mtime.Location()))
				update <- local
			}
		}
		// now we check the left-overs in the local file that hasn't been handled since they dont exist on the remote
		for _, local := range localFiles {
			logger.Debug.Printf("syncing: %s, file does not exist at destination\n", local.path)
			update <- local
		}
		close(update)
		logger.Debug.Printf("Found %d local files\n", numLocalFiles)
		logger.Debug.Printf("Found %d remote files\n", numRemoteFiles)
	}()
	return update
}

func syncFiles(svc *s3.S3, bucket, bucketPath, localPath string, in chan *File, logger *Logger) {

	concurrency := 5
	sem := make(chan bool, concurrency)
	var numSyncedFiles int

	for file := range in {
		numSyncedFiles++
		// add one
		sem <- true
		go func(svc *s3.S3, bucket, bucketPath, localPath string, file *File, logger *Logger) {
			upload(svc, bucket, bucketPath, localPath, file, logger)
			// remove one
			<-sem
		}(svc, bucket, bucketPath, localPath, file, logger)
	}

	// After the last goroutine is fired, there are still concurrency amount of goroutines running. In order to make
	// sure we wait for all of them to finish, we attempt to fill the semaphore back up to its capacity. Once that
	// succeeds, we know that the last goroutine has read from the semaphore, as we've done len(urls) + cap(sem) writes
	// and len(urls) reads off the channel.
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	logger.Debug.Printf("Synced %d local files to remote\n", numSyncedFiles)
}

func upload(svc *s3.S3, bucket, bucketPath, localPath string, file *File, logger *Logger) {

	realFile, err := os.Open(filepath.Join(localPath, file.path))
	if err != nil {
		logger.Err.Printf("error opening file: %s\n", err)
		return
	}
	defer realFile.Close()

	// create a byte buffer reader for the content of the local file
	buffer := make([]byte, file.size)
	realFile.Read(buffer)

	key := filepath.Join(bucketPath, file.path)
	key = strings.TrimPrefix(key, "/")

	// Create an uploader (can do multipart) with S3 client and default options
	uploader := s3manager.NewUploaderWithClient(svc)
	params := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(http.DetectContentType(buffer)),
	}

	s3Uri := filepath.Join(bucket, key)
	if !dryrun {
		_, err := uploader.Upload(params)
		if err != nil {
			logger.Out.Printf("bad response: %+v", err)
			return
		}
		logger.Out.Printf("upload: %s to s3://%s\n", file.path, s3Uri)
	} else {
		logger.Out.Printf("(dryrun) upload: %s to s3://%s\n", file.path, s3Uri)
	}
}

type stringSlice []string

// String is the method to format the flag's value, part of the flag.Value interface.
// The String method's output will be used in diagnostics.
func (e *stringSlice) String() string {
	return fmt.Sprint(*e)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}
