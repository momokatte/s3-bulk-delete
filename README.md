s3-bulk-delete
==============

A Go app for bulk-deleting files from an Amazon S3 bucket using the Multi-Object Delete operation.

Supports stop/resume: optionally writes completed batch numbers to a file, and reads from that file on startup to load batch numbers to skip.


Usage
-----

The application will receive a list of S3 keys via standard input, one key per line. Example input:

> prefix_01/file_01.dat
> prefix_01/file_02.dat
> prefix_01/file_03.dat
> prefix_01/file_04.dat
> prefix_02/file_01.dat
> prefix_02/file_02.dat
> prefix_02/file_03.dat
> prefix_02/file_04.dat

Provide the AWS region and S3 bucket name via flags:

```bash
<s3-keys.txt s3-bulk-delete -region us-east-1 -bucket my-bucket -skip deleted-batches.txt
```


API Credentials
---------------

The application supports these AWS [credential providers](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials):

- environment variables
- shared credentials file
- EC2 Instance Metadata service


Specifying keys for deletion
----------------------------

The application does not retrieve key listings from S3; they must be provided from an external source.

If you have [Amazon S3 Inventory](https://docs.aws.amazon.com/AmazonS3/latest/dev/storage-inventory.html) enabled for your bucket, it provides file data that can be used to generate a list of keys to delete.

The [AWS command-line interface](https://docs.aws.amazon.com/cli/latest/reference/s3/ls.html) can list all keys with a specified prefix. Here's an example that uses some additional programs to filter out folder objects and file metadata:

```bash
aws --region=us-east-1 s3 ls --recursive s3://my-bucket/folderol/ | grep -v "\/$" | awk '{ print $4 }' >s3-keys.txt
```

If you just want to delete a few hundred or thousand keys with a particular prefix without previewing them, the AWS command-line interface provides a [recursive delete option](https://docs.aws.amazon.com/cli/latest/reference/s3/rm.html).


Request rate and concurrency
----------------------------

Through trial and error I have determined that the S3 Multi-Object Delete operation is governed by the [documented rate limit](https://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html) of 3500 DELETE requests per second (per bucket + prefix). It's allowed to exceed that rate for short durations, but sustained deletes of 3500 or more objects per second will eventually result in throttled API responses.

I've also found that API response times degrade with high concurrency and will sometimes result in "internal error" responses. 12 concurrent requests provides a good balance between response time and total batch throughput.

With the default settings, the application will realistically delete 3000 objects per second or just under 11 million objects per hour. If you have hundreds of millions of objects to delete under different prefixes, it would be best to split up the input keys and run multiple instances of the application.


API request costs
-----------------

DELETE requests are free for 'S3 Standard' and 'S3 Intelligent-Tiering' storage classes. Pricing for other storage classes is [here](https://aws.amazon.com/s3/pricing/#Request_pricing).


Roadmap
-------

January 2019:

- IAM role support
- More metrics
