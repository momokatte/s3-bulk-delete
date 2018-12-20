# s3-bulk-delete

Go app for deleting files from an Amazon S3 bucket using the Multi-Object Delete operation (batches of 1000).

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

## Request rate

Through trial and error I have determined that the S3 Multi-Object Delete operation is governed by the [documented rate limit](https://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html) of 3500 DELETE requests per second (per bucket + prefix). It's allowed to exceed that rate for short durations, but sustained deletes of over 3500 objects per second will eventually result in throttled API responses.

## Roadmap

December 2018:

* Output completed batch numbers to skip-file
* App-level incremental/exponential backoff on API error
* Documentation

January 2018:

* Metrics
