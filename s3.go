package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
	"time"
)

func loadS3Files(svc *s3.S3, bucket, path string, buffer int, logger *Logger) (chan LocalFileResult, error) {
	out := make(chan LocalFileResult, buffer)

	// s3 doesn't like the key to start with /
	path = strings.TrimPrefix(path, "/")

	// initial check if we can read this bucket with prefix
	_, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(path),
	})
	if err != nil {
		return out, fmt.Errorf("Could not s3:listObjects on 's3://%s/%s':\n%s", bucket, path, err)
	}

	go func() {
		start := time.Now()
		logger.Debug.Printf("read s3 - start at %s", start)
		trawlS3(svc, path, bucket, path, out, nil, logger)
		logger.Debug.Printf("read s3 - stop, it took %s", time.Now().Sub(start))
		close(out)
	}()
	return out, nil
}

func trawlS3(svc *s3.S3, path string, bucket, prefix string, out chan LocalFileResult, token *string, logger *Logger) {
	list, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:            aws.String(bucket),
		Prefix:            aws.String(prefix),
		ContinuationToken: token,
	})

	if err != nil {
		out <- LocalFileResult{
			err: err,
		}
		return
	}

	for _, object := range list.Contents {
		// strip out the full path of the object, begin after path
		p := strings.TrimPrefix(*object.Key, path)
		p = strings.TrimPrefix(p, "/")
		out <- LocalFileResult{
			file: &File{
				path:  p,
				size:  *object.Size,
				mtime: *object.LastModified,
			},
		}
	}

	if *list.IsTruncated {
		trawlS3(svc, path, bucket, prefix, out, list.NextContinuationToken, logger)
	}
}
