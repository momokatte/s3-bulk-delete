s3-bulk-delete
==============

A Go app for bulk-deleting files from an Amazon S3 bucket using the Multi-Object Delete operation.

Supports stop/resume: optionally writes completed batch numbers to a file, and reads from that file on startup to load batch numbers to skip.


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

		<input-keys.txt s3-bulk-delete -region us-east-1 -bucket so-many-files -skip deleted-batches.txt


Request rate and concurrency
----------------------------

Through trial and error I have determined that the S3 Multi-Object Delete operation is governed by the [documented rate limit](https://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html) of 3500 DELETE requests per second (per bucket + prefix). It's allowed to exceed that rate for short durations, but sustained deletes of 3500 or more objects per second will eventually result in throttled API responses.

I've also found that API response times degrade with high concurrency and will sometimes result in "internal error" responses. 12 concurrent requests provides a good balance between response time and total batch throughput.


Roadmap
-------

January 2018:

* More metrics
