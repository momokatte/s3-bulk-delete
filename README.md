# s3-bulk-delete

Go app for deleting files from an Amazon S3 bucket using the Multi-Object Delete operation (batches of 1000).

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

## Request rate

Through trial and error I have estimated that the S3 API will allow 3 DeleteObjects requests per second. This limit is probably per path prefix, per bucket. At that request rate, you'll probably have up to 6 requests "in flight" at any time and not benefit much from additional concurrency capacity.

I have set the default concurrency to 12 with the expectation that half of the operations will be waiting and ready to execute as soon as another one has completed.

## Roadmap

December 2018:

* Output completed batch numbers to skip-file
* App-level incremental/exponential backoff on API error
* Documentation

January 2018:

* Metrics
