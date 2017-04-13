package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"time"
)

func loadS3Files(conf *Config, buffer int, logger *Logger) chan *FileStat {
	out := make(chan *FileStat, buffer)
	go func() {
		start := time.Now()
		logger.Debug.Printf("read s3 - start at %s", start)
		continuationToken := listS3Files(conf, out, nil)
		for continuationToken != nil {
			continuationToken = listS3Files(conf, out, continuationToken)
		}
		logger.Debug.Printf("read s3 - stop, it took %s", time.Since(start))
		close(out)
	}()
	return out
}

func listS3Files(config *Config, out chan *FileStat, token *string) *string {
	list, err := config.S3Service.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:            aws.String(config.Bucket),
		Prefix:            aws.String(config.BucketPrefix),
		ContinuationToken: token,
	})
	if err != nil {
		out <- &FileStat{Err: err}
		return nil
	}
	for _, object := range list.Contents {
		out <- &FileStat{
			Name:    strings.TrimPrefix(*object.Key, config.BucketPrefix+"/"),
			Path:    *object.Key,
			Size:    *object.Size,
			ModTime: *object.LastModified,
		}
	}
	return list.NextContinuationToken
}
