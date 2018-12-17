# s3-bulk-delete

Go app for deleting files from an Amazon S3 bucket using the Multi-Object Delete operation (batches of 1000).

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

## Known Bugs

* Exits on all S3 errors instead of retrying with backoff

## TODO

* Get client retry/backoff working
* Figure out actual rate limit for S3 DeleteObjects API call
* Output completed batch numbers to skip-file

