package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"time"
)

func loadS3Files(conf *Config, buffer int, logger *Logger) (chan *FileStat, error) {
	// svc, bucket, bucketPath
	out := make(chan *FileStat, buffer)

	// s3 doesn't like the key to start with /
	logger.Debug.Printf("load prefix: %s\n", conf.BucketPrefix)

	// initial check if we can read this bucket with prefix
	_, err := conf.S3Service.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(conf.Bucket),
		Prefix: aws.String(conf.BucketPrefix),
	})
	if err != nil {
		return out, fmt.Errorf("Could not s3:listObjects on 's3://%s/%s':\n%s", conf.Bucket, conf.BucketPrefix, err)
	}

	go func() {
		start := time.Now()
		logger.Debug.Printf("read s3 - start at %s", start)
		trawlS3(conf, out, nil, logger)
		logger.Debug.Printf("read s3 - stop, it took %s", time.Since(start))
		close(out)
	}()
	return out, nil
}

func trawlS3(config *Config, out chan *FileStat, token *string, logger *Logger) {
	list, err := config.S3Service.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:            aws.String(config.Bucket),
		Prefix:            aws.String(config.BucketPrefix),
		ContinuationToken: token,
	})

	if err != nil {
		out <- &FileStat{
			Err: err,
		}
		return
	}

	for _, object := range list.Contents {
		out <- &FileStat{
			Path:    strings.TrimPrefix(*object.Key, config.BucketPrefix+"/"),
			Size:    *object.Size,
			ModTime: *object.LastModified,
		}
	}

	if *list.IsTruncated {
		trawlS3(config, out, list.NextContinuationToken, logger)
	}
}
