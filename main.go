package main

import (
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func main() {
	dryrun := flag.Bool("dryrun", false, "Displays the operations that would be performed using the specified command without actually running them.")
	debug := flag.Bool("debug", false, "Turn on debug logging.")
	onlyShowErrors := flag.Bool("only-show-errors", false, "Only errors and warnings are displayed. All other output is suppressed.")
	region := flag.String("region", "", "The region to use. Overrides config/env settings.")
	profile := flag.String("profile", "", "Use a specific profile from your credential file.")
	var exclude StringSlice
	flag.Var(&exclude, "exclude", "Exclude all files or objects from the command that matches the specified pattern, only supports '*' globbing.")

	flag.Parse()

	logger := NewLogger(*debug, *onlyShowErrors)

	localPath, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		flag.Usage()
		logger.Err.Printf("\nCould not parse LocalPath '%s': %s\n", flag.Arg(0), err)
		os.Exit(1)
	}

	s3URL, err := url.Parse(flag.Arg(1))
	if err != nil {
		flag.Usage()
		logger.Err.Printf("\nCould not parse S3Uri '%s'\n", flag.Arg(1))
		os.Exit(1)
	}
	if s3URL.Scheme != "s3" {
		flag.Usage()
		logger.Err.Println("\nS3Uri argument does not have valid protocol, should be 's3'")
		os.Exit(1)
	}
	if s3URL.Host == "" {
		flag.Usage()
		logger.Err.Println("\nS3Uri is missing bucket name")
		os.Exit(1)
	}

	sess, err := getSession(*profile, *region, logger)
	if err != nil {
		logger.Err.Printf("%v\n", err)
	}

	config := &Config{
		S3Service:    s3.New(sess),
		Bucket:       s3URL.Host,
		BucketPrefix: strings.TrimPrefix(s3URL.Path, "/"),
	}

	// load all local files that doesn't match exclude
	local := loadLocalFiles(localPath, exclude, logger)

	// we keep 50,000 (50 s3:listObjects calls) to be in the output remote channel,
	// this will ensure that we can find all local files without blocking the AWS calls
	remote := loadS3Files(config, 50000, logger)

	// find out which files that needs syncing
	files := compare(local, remote, logger)

	// sync all files to s3
	syncFiles(config, files, *dryrun, logger)
}

// compare will put a local file on the output channel if:
// - the size of the local file is different than the size of the s3 object
// - the last modified time of the local file is newer than the last modified time of the s3 object
// - the local file does not exist under the specified bucket and prefix.
// This is the same logic as the aws s3 sync tool uses, see https://github.com/aws/aws-cli/blob/e2295b022db35eea9fec7e6c5540d06dbd6e588b/awscli/customizations/s3/syncstrategy/base.py#L226
func compare(foundLocal, foundRemote chan *FileStat, logger *Logger) chan *FileStat {

	update := make(chan *FileStat, 8)

	// first we sink the local files into a lookup map so its quick and easy to compare that to the remote
	localFiles := make(map[string]*FileStat)
	for r := range foundLocal {
		if r.Err != nil {
			logger.Err.Println(r.Err)
			continue
		}
		localFiles[r.Name] = r
	}

	numLocalFiles := len(localFiles)
	numRemoteFiles := 0

	go func() {
		defer close(update)

		for remote := range foundRemote {
			if remote.Err != nil {
				logger.Err.Printf("Remote %s\n", remote.Err)
				return
			}
			numRemoteFiles++
			if local, ok := localFiles[remote.Name]; ok {
				if local.Size != remote.Size {
					logger.Debug.Printf("syncing: %s, size %d -> %d\n", local.Name, local.Size, remote.Size)
					update <- local
				} else if local.ModTime.After(remote.ModTime) {
					logger.Debug.Printf("syncing: %s, modified time: %s -> %s\n", local.Name, local.ModTime, remote.ModTime.In(local.ModTime.Location()))
					update <- local
				}
				delete(localFiles, remote.Name)
			}
		}

		for _, local := range localFiles {
			logger.Debug.Printf("syncing: %s, file does not exist at destination\n", local.Name)
			update <- local
		}
		logger.Debug.Printf("Found %d local files\n", numLocalFiles)
		logger.Debug.Printf("Found %d remote files\n", numRemoteFiles)
	}()

	return update
}

// syncFiles takes a channel of *FileStat and tries to upload them to s3
func syncFiles(config *Config, in chan *FileStat, dryrun bool, logger *Logger) {

	concurrency := 5
	sem := make(chan bool, concurrency)
	var numSyncedFiles int

	for file := range in {
		// add one
		sem <- true
		go func(config *Config, file *FileStat, logger *Logger) {
			err := upload(config, file, dryrun, logger)
			if err != nil {
				logger.Err.Println(err)
			} else {
				numSyncedFiles++
			}
			// remove one
			<-sem
		}(config, file, logger)
	}

	// After the last goroutine is fired, there are still concurrency amount of goroutines running. In order to make
	// sure we wait for all of them to finish, we attempt to fill the semaphore back up to its capacity. Once that
	// succeeds, we know that the last goroutine has read from the semaphore, as we've done len(files) + cap(sem) writes
	// and len(files) reads off the channel.
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	logger.Debug.Printf("Synced %d local files to remote\n", numSyncedFiles)
}

func upload(config *Config, fileStat *FileStat, dryrun bool, logger *Logger) error {

	logger.Debug.Printf("will upload %s to s3://%s/%s\n", fileStat.Path, config.Bucket, config.BucketPrefix)

	file, err := os.Open(fileStat.Path)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Err.Printf("Problem closing file %s: %v", fileStat.Path, err)
		}
	}()

	key := filepath.Join(config.BucketPrefix, fileStat.Name)
	key = strings.TrimPrefix(key, "/")

	// Create an uploader (can do multipart) with S3 client and default options
	uploader := s3manager.NewUploaderWithClient(config.S3Service)
	params := &s3manager.UploadInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key),
		Body:   file,
	}

	s3Uri := filepath.Join(config.Bucket, key)
	if dryrun {
		logger.Out.Printf("(dryrun) upload: %s to s3://%s\n", fileStat.Name, s3Uri)
		return nil
	}

	if _, err = uploader.Upload(params); err != nil {
		return err
	}

	logger.Out.Printf("upload: %s to s3://%s\n", fileStat.Name, s3Uri)
	return nil
}

func getSession(profile, region string, logger *Logger) (*session.Session, error) {
	options := session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}
	if profile != "" {
		logger.Debug.Printf("Using credentials profile: %s\n", profile)
		options.Profile = profile
	}
	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return sess, err
	}
	sess.Config.Region = aws.String(getRegion(sess, region, logger))
	return sess, nil
}

func getRegion(p client.ConfigProvider, region string, logger *Logger) string {

	if region != "" {
		logger.Debug.Printf("Found region in CLI options: %s\n", region)
		return region
	}

	if os.Getenv("AWS_REGION") != "" {
		logger.Debug.Printf("Found region in ENV: %s\n", os.Getenv("AWS_REGION"))
		return os.Getenv("AWS_REGION")
	}

	cc := p.ClientConfig("s3")
	if *cc.Config.Region != "" {
		logger.Debug.Printf("Found region in client config: %s\n", *cc.Config.Region)
		return *cc.Config.Region
	}

	// check if running inside EC2, then grab the region from the EC2 metadata service
	md := ec2metadata.New(p)
	if md.Available() {
		reg, err := md.Region()
		if err != nil {
			logger.Err.Println(err)
		} else {
			logger.Debug.Printf("Found region in AWS EC2 metadata: %s\n", reg)
			return reg
		}
	}
	return ""
}
