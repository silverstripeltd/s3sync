# s3sync

[![CircleCI](https://circleci.com/gh/silverstripeltd/s3sync/tree/master.svg?style=svg)](https://circleci.com/gh/silverstripeltd/s3sync/tree/master)

s3sync syncs files from a local directory to a AWS S3 bucket than the aws cli tool does. It does this by being very specific in what what IO operations it does. This can make a difference when there are 100 00 files when only a few files have been updated. 
 
## Installation

Either download binaries for your platform from [releases](https://github.com/silverstripeltd/s3sync/releases) or use the go installation method:

```bash
$ go get github.com/silverstripeltd/s3sync/releases
```

## Usage


```
s3sync [options] source_directory s3://bucket_name/prefix

  -debug
    	Turn on debug logging.
  -dryrun
    	Displays the operations that would be performed using the specified command without actually running them.
  -exclude value
    	Exclude all files or objects from the command that matches the specified pattern, only supports '*' "globbing".
  -only-show-errors
    	Only errors and warnings are displayed. All other output is suppressed.
  -profile string
    	Use a specific profile from your credential file.
  -region string
    	The region to use. Overrides config/env settings.
```

# Example benchmark
 
This benchmark was recorded on an AWS EC2 t2.nano instance with ~25 000 files where all but two files was sup to date.

### S3sync 3.5 seconds
```bash
$ time s3sync /var/www/ s3://sync_bucket/www --exclude *.bak
upload: folder/file_a to s3://sync_bucket/www/folder/file_a
upload: file_b to s3://sync_bucket/www/file_b

real    0m3.520s
user    0m1.068s
sys    0m0.100s
```

### AWS CLI tool

```bash 36.3 seconds
$ time aws s3 sync /var/www/ s3://sync_bucket/www --exclude *.bak
upload: folder/file_a to s3://sync_bucket/www/folder/file_a
upload: file_b to s3://sync_bucket/www/file_b

real    0m36.328s
user    0m16.784s
sys    0m1.664s
```


## development

### Updating vendor libraries

First ensure that you have installed [glide](https://glide.sh/), a dependency and vendor manager for go.

After you have glide installed, run `glide update`. For more information on how to use glide for adding 
dependencies, see the [glide documentation](https://glide.readthedocs.io/en/latest/).
 
Use the Makefile to run common operations

 - `make dev` run go fmt and go vet  
 - `make install` runs go install
 - `make release` creates binaries for linux, mac os x and windows
 - `make clean` - removes all release binaries
 - `make test` - runs all the test and runs linters and other tools, see Makefile for necessary pre-requisites.
 - `make cover` - generats test coverage and opens the result in a browser 	


