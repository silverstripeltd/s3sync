package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
	"strings"
	"time"
)

func loadS3Files(svc *s3.S3, bucket, path string, debug *log.Logger) chan map[string]*File {

	out := make(chan map[string]*File)

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	go func() {
		start := time.Now()
		debug.Printf("read s3 - start at %s", start)
		f := make(map[string]*File)
		trawlS3(svc, path, bucket, path, f, nil, debug)
		debug.Printf("read s3 - stop, it took %s", time.Now().Sub(start))
		out <- f
		close(out)
	}()
	return out
}

func trawlS3(svc *s3.S3, path string, bucket, prefix string, files map[string]*File, token *string, debug *log.Logger) {
	list, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:            aws.String(bucket),
		Prefix:            aws.String(prefix),
		ContinuationToken: token,
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, object := range list.Contents {
		// strip out the full path of the object, begin after path
		p := strings.TrimPrefix(*object.Key, path)
		files[p] = &File{
			path:  p,
			size:  *object.Size,
			mtime: *object.LastModified,
		}
	}

	if *list.IsTruncated {
		trawlS3(svc, path, bucket, prefix, files, list.NextContinuationToken, debug)
	}
}
