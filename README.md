s3-bulk-delete
==============

A Go app for bulk-deleting files from an Amazon S3 bucket using the Multi-Object Delete operation.

Accepts a file containing batch numbers to skip so the input file can be run again after interruption.

Usage
-----

The application will receive a list of S3 keys via standard input, one key per line. Example input:

		prefix_01/file_01.dat
		prefix_01/file_02.dat
		prefix_01/file_03.dat
		prefix_01/file_04.dat
		prefix_02/file_01.dat
		prefix_02/file_02.dat
		prefix_02/file_03.dat
		prefix_02/file_04.dat

Provide the AWS region and S3 bucket name via flags:

		<input-keys.txt s3-bulk-delete -region us-east-1 -bucket so-many-files

Request rate
------------

Through trial and error I have determined that the S3 Multi-Object Delete operation is governed by the [documented rate limit](https://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html) of 3500 DELETE requests per second (per bucket + prefix). It's allowed to exceed that rate for short durations, but sustained deletes of over 3500 objects per second will eventually result in throttled API responses.

Roadmap
-------

December 2018:

* Output completed batch numbers to skip-file
* App-level incremental/exponential backoff on API error

January 2018:

* Metrics
