package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	aws "github.com/aws/aws-sdk-go/aws"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	session "github.com/aws/aws-sdk-go/aws/session"
	s3 "github.com/aws/aws-sdk-go/service/s3"
)

type S3Deleter struct {
	client *s3.S3
	bucket string
	mfa    string
	quiet  bool
}

func NewS3Deleter(region string, bucket string, mfa string, quiet bool) (*S3Deleter, error) {
	config := aws.Config{
		Region:                        &region,
		CredentialsChainVerboseErrors: aws.Bool(true),
		HTTPClient: &http.Client{
			Timeout: time.Minute,
		},
		MaxRetries: aws.Int(4),
		Retryer:    awsclient.DefaultRetryer{4},
	}
	sess, err := session.NewSession(&config)
	if err != nil {
		return nil, err
	}
	d := &S3Deleter{
		client: s3.New(sess),
		bucket: bucket,
		mfa:    mfa,
		quiet:  quiet,
	}
	return d, nil
}

func (d *S3Deleter) DeleteKeys(keys []string) error {
	del := &s3.Delete{
		Objects: toOIDs(keys),
		Quiet:   &d.quiet,
	}
	doi := &s3.DeleteObjectsInput{
		Bucket: &d.bucket,
		Delete: del,
	}
	if d.mfa != "" {
		doi.MFA = &d.mfa
	}
	res, err := d.client.DeleteObjects(doi)
	if err != nil {
		return err
	}
	if len(res.Errors) != 0 {
		return d.newBatchError(res.Errors)
	}
	return nil
}

func (_ *S3Deleter) newBatchError(errs []*s3.Error) error {
	msgs := make([]string, len(errs))
	for i := 0; i < len(errs); i += 1 {
		msgs[i] = *errs[i].Message
	}
	return NewBatchError(msgs)
}

type BatchError struct {
	error
	Messages []string
}

func NewBatchError(msgs []string) error {
	if len(msgs) == 0 {
		panic("msgs must contain one or more messages")
	}
	msg := msgs[0]
	if more := len(msgs) - 1; more > 0 {
		msg += fmt.Sprintf(" (and %d more errors)", more)
	}
	return BatchError{errors.New(msg), msgs}
}

func toOIDs(keys []string) []*s3.ObjectIdentifier {
	ret := make([]*s3.ObjectIdentifier, len(keys))
	for i := 0; i < len(ret); i += 1 {
		oid := &s3.ObjectIdentifier{
			Key: &(keys[i]),
		}
		ret[i] = oid
	}
	return ret
}

// error messages:
//   Access Denied
//   We encountered an internal error. Please try again.
