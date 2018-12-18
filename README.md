# s3-bulk-delete

Go app for deleting files from an Amazon S3 bucket using the Multi-Object Delete operation (batches of 1000).

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

## Known Issues

* Exits on all S3 errors instead of retrying with backoff

## Roadmap

December 2018:

* Output completed batch numbers to skip-file
* Get client retry/backoff working
* Documentation

January 2019:

* Figure out actual rate limit for S3 DeleteObjects API call
* Provide optimal default settings for concurrency
