# s3-bulk-delete

Go app for deleting files from an Amazon S3 bucket using the Multi-Object Delete operation (batches of 1000).

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

## Roadmap

December 2018:

* Output completed batch numbers to skip-file
* Documentation
