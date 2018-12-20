package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	limiter "github.com/momokatte/go-limiter"
)

func main() {
	var mainErr error
	var conf *configuration

	conf, mainErr = loadConfig()
	if mainErr != nil {
		fmt.Fprintln(os.Stderr, mainErr.Error())
		os.Exit(1)
	}

	var s3Deleter *S3Deleter
	s3Deleter, mainErr = NewS3Deleter(conf.Region, conf.Bucket, conf.MFA, conf.Quiet, conf.Debug)
	if mainErr != nil {
		fmt.Fprintln(os.Stderr, mainErr.Error())
		os.Exit(1)
	}

	var skipBatches map[int]bool
	if conf.SkipBatches != "" {
		skipBatches, mainErr = loadSkipFile(conf.SkipBatches)
		if mainErr != nil {
			fmt.Fprintln(os.Stderr, mainErr.Error())
			os.Exit(1)
		}
	}

	interval := (1005 * conf.BatchSize / conf.RateLimit) + 1

	concurrency := 1
	if !conf.Serial {
		concurrency = 1000 / interval
		if concurrency < 1 {
			concurrency = 1
		} else if concurrency > 12 {
			concurrency = 12
		}
	}
	cLim := limiter.NewTokenChanLimiter(uint(concurrency))

	keysInput := make(chan string, conf.BatchSize*concurrency)
	batches := make(chan keyBatch, concurrency*2)
	batchesConsumed := make(chan bool)

	if !conf.Quiet {
		fmt.Fprintf(os.Stdout, "Concurrency: %d\n", concurrency)
		fmt.Fprintf(os.Stdout, "Request interval: %dms\n", interval)
	}

	// deleter (batch consumer)
	go func() {
		rLim := limiter.NewBurstRateLimiter(limiter.NewRate(1, time.Duration(interval)*time.Millisecond))

		for batch := range batches {
			// enforce concurrency limit
			t := cLim.AcquireToken()

			go func(batch keyBatch) {
				for {
					if rLim != nil {
						// enforce request rate limit
						rLim.CheckWait()
					}
					if !conf.Quiet {
						fmt.Fprintf(os.Stdout, "Deleting batch %d\n", batch.Num)
					}
					err := s3Deleter.DeleteKeys(batch.Keys)
					if err == nil {
						break
					}
					if bErr, ok := err.(BatchError); ok {
						if !strings.Contains(bErr.Messages[0], " try again") {
							for _, msg := range bErr.Messages {
								fmt.Fprintf(os.Stderr, "[Batch %d] %s\n", batch.Num, msg)
							}
							os.Exit(2)
						}
					}
					// log, but retry
					fmt.Fprintf(os.Stderr, "[Batch %d] %s\n", batch.Num, err.Error())
				}
				if !conf.Quiet {
					fmt.Fprintf(os.Stdout, "Deleted batch %d\n", batch.Num)
				}
				cLim.ReleaseToken(t)
			}(batch)
		}
		batchesConsumed <- true
	}()

	// batcher (key consumer)
	go func() {
		for batchNum := 1; true; batchNum += 1 {
			keys := make([]string, conf.BatchSize)
			var keyIdx int
			for key := range keysInput {
				keys[keyIdx] = key
				keyIdx += 1
				if keyIdx >= conf.BatchSize {
					break
				}
			}
			if keyIdx == 0 {
				// no more input
				break
			}
			if conf.SkipBatches != "" {
				if _, exists := skipBatches[batchNum]; exists {
					if !conf.Quiet {
						fmt.Fprintf(os.Stdout, "Skipped batch %d\n", batchNum)
					}
					continue
				}
			}
			batches <- keyBatch{batchNum, keys[:keyIdx]}
		}
		close(batches)
	}()

	// key producer
	mainErr = scanInputKeys(os.Stdin, keysInput)
	if mainErr != nil {
		fmt.Fprintln(os.Stderr, mainErr.Error())
		os.Exit(1)
	}
	close(keysInput)

	// wait until last batch of keys is consumed
	<-batchesConsumed

	// drain all the tokens after consumers are done with them
	for i := concurrency; i > 0; i -= 1 {
		_ = cLim.AcquireToken()
	}
}

func scanInputKeys(r io.Reader, keysInput chan<- string) error {
	s := bufio.NewScanner(r)
	for {
		if ok := s.Scan(); !ok {
			if err := s.Err(); err != nil {
				return err
			}
			break
		}
		b := make([]byte, len(s.Bytes()))
		_ = copy(b, s.Bytes())
		keysInput <- string(b)
	}
	return nil
}

type keyBatch struct {
	Num  int
	Keys []string
}
