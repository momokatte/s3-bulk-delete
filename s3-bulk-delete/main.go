package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
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
	s3Deleter, mainErr = NewS3Deleter(conf.Region, conf.Bucket, conf.MFA, true)
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

	keysInput := make(chan string, 1000*conf.Concurrency)

	lim := limiter.NewTokenChanLimiter(conf.Concurrency)

	var wg sync.WaitGroup
	wg.Add(1)

	// key consumer
	go func() {
		for batchNum := 1; true; batchNum += 1 {
			// ideally move keys from heap to stack
			var keyBuf [1000]string
			var keyIdx int
			for key := range keysInput {
				keyBuf[keyIdx] = key
				keyIdx += 1
				if keyIdx >= 1000 {
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

			// rate limiting: fastest so far is 8 concurrency with rate limit of 1 request per second
			// enforce maximum concurrency
			t := lim.AcquireToken()
			// enforce maximum rate
			time.Sleep(time.Second)

			go func(keys []string, batchNum int) {
				if !conf.Quiet {
					fmt.Fprintf(os.Stdout, "Deleting batch %d\n", batchNum)
				}
				if err := s3Deleter.DeleteKeys(keys); err != nil {
					if bErr, ok := err.(BatchError); ok {
						for _, msg := range bErr.Messages {
							fmt.Fprintf(os.Stderr, "[Batch %d] %s\n", batchNum, msg)
						}
					} else {
						fmt.Fprintf(os.Stderr, "[Batch %d] %s\n", batchNum, err.Error())
					}
					os.Exit(2)
				}
				if !conf.Quiet {
					fmt.Fprintf(os.Stdout, "Deleted batch %d\n", batchNum)
				}
				lim.ReleaseToken(t)
			}(keyBuf[:keyIdx], batchNum)
		}
		wg.Done()
	}()

	// key producer
	mainErr = scanInputKeys(os.Stdin, keysInput)
	if mainErr != nil {
		fmt.Fprintln(os.Stderr, mainErr.Error())
		os.Exit(1)
	}
	close(keysInput)

	// wait until last batch of keys is consumed
	wg.Wait()

	// drain all the tokens after consumers are done with them
	for i := conf.Concurrency; i > 0; i -= 1 {
		_ = lim.AcquireToken()
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
