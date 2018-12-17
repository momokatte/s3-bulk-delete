package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"
)

type configuration struct {
	MFA         string
	Region      string
	Bucket      string
	Concurrency uint
	Quiet       bool
	SkipBatches string
}

func (conf *configuration) Load() error {
	flag.StringVar(&conf.MFA, "mfa", "", "MFA string")
	flag.StringVar(&conf.Region, "region", "", "AWS region name to connect to")
	flag.StringVar(&conf.Bucket, "bucket", "", "S3 bucket to delete files from")
	flag.UintVar(&conf.Concurrency, "concurrency", 1, "")
	flag.BoolVar(&conf.Quiet, "quiet", false, "Quiet mode")
	flag.StringVar(&conf.SkipBatches, "skip", "", "Skip file, containing batch numbers to skip")
	flag.Parse()
	return conf.Validate()
}

func (conf configuration) Validate() error {
	if conf.Bucket == "" {
		return errors.New("Bucket is required")
	}
	if conf.Concurrency < 1 {
		return errors.New("Concurrency must be greater than 0")
	}
	return nil
}

func loadConfig() (*configuration, error) {
	c := &configuration{}
	return c, c.Load()
}

func loadSkipFile(filename string) (map[int]bool, error) {
	ret := make(map[int]bool)
	f, err := os.Open(filename)
	if err != nil {
		return ret, err
	}
	err = scanInts(f, func(val int) error {
		if val < 0 {
			return errors.New("Skip file cannot contain negative numbers")
		}
		ret[val] = true
		return nil
	})
	if closeErr := f.Close(); closeErr != nil {
		return ret, closeErr
	}
	return ret, err
}

func scanInts(r io.Reader, f func(int) error) error {
	s := bufio.NewScanner(r)
	for {
		if ok := s.Scan(); !ok {
			if err := s.Err(); err != nil {
				return err
			}
			break
		}
		val, err := strconv.Atoi(strings.TrimSpace(s.Text()))
		if err != nil {
			return err
		}
		err = f(val)
		if err != nil {
			return err
		}
	}
	return nil
}
