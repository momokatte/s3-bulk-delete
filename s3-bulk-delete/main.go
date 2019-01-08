package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
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
	if conf.SkipFile != "" {
		skipBatches, mainErr = loadSkipFile(conf.SkipFile)
		if mainErr != nil {
			if _, ok := mainErr.(*os.PathError); !ok {
				fmt.Fprintln(os.Stderr, mainErr.Error())
				os.Exit(1)
			}
		}
	}

	interval := (1000 * conf.BatchSize / conf.RateLimit)
	if interval < 1 {
		interval = 1
	}
	concurrency := conf.CFactor / interval
	if concurrency < 1 {
		concurrency = 1
	} else if concurrency > conf.CMax {
		concurrency = conf.CMax
	}

	keysInput := make(chan string, conf.BatchSize*concurrency)
	batches := make(chan keyBatch, concurrency*2)
	completed := make(chan int)
	done := make(chan int)

	if !conf.Quiet {
		fmt.Fprintf(os.Stdout, "Batch size: %d\n", conf.BatchSize)
		fmt.Fprintf(os.Stdout, "Request interval: %dms\n", interval)
		fmt.Fprintf(os.Stdout, "Maximum concurrency: %d\n", concurrency)
	}

	// accounting
	go func() {
		var err error
		var completedCount int
		var completedLog *os.File

		if conf.SkipFile != "" {
			completedLog, err = os.OpenFile(conf.SkipFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			defer completedLog.Close()
		}

		for batchNum := range completed {
			completedCount += 1
			if completedLog != nil {
				fmt.Fprintf(completedLog, "%d\n", batchNum)
			}
		}

		done <- completedCount
	}()

	lim := NewIntervalFailLimiter(time.Duration(interval)*time.Millisecond, time.Second*10)

	var deleterWG sync.WaitGroup
	deleterWG.Add(concurrency)

	// create static pool of deleter (batch consumer) goroutines
	for i := concurrency; i > 0; i -= 1 {
		go func() {
			for batch := range batches {
				// retry on most API errors until batch is deleted
				for {
					// enforce API request rate limit
					lim.CheckWait()
					if !conf.Quiet {
						fmt.Fprintf(os.Stdout, "Deleting batch %d\n", batch.Num)
					}
					start := time.Now()
					err := s3Deleter.DeleteKeys(batch.Keys)
					dur := time.Now().Sub(start)
					lim.Report(err == nil)
					if err == nil {
						if !conf.Quiet {
							fmt.Fprintf(os.Stdout, "Deleted batch %d (%s)\n", batch.Num, dur.String())
						}
						completed <- batch.Num
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
					// log error
					fmt.Fprintf(os.Stderr, "[Batch %d] %s\n", batch.Num, err.Error())
				}
			}
			deleterWG.Done()
		}()
	}

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
			if conf.SkipFile != "" {
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

	mainStart := time.Now()

	// key producer
	mainErr = scanInputKeys(os.Stdin, keysInput)
	if mainErr != nil {
		fmt.Fprintln(os.Stderr, mainErr.Error())
		os.Exit(1)
	}
	close(keysInput)

	deleterWG.Wait()

	close(completed)

	mainDur := time.Now().Sub(mainStart)

	completedCount := <-done

	if !conf.Quiet {
		r := float64(completedCount*conf.BatchSize) / mainDur.Seconds()

		fmt.Fprintf(os.Stdout, "Deleted batches: %d\n", completedCount)
		fmt.Fprintf(os.Stdout, "Duration: %s\n", mainDur.String())
		fmt.Fprintf(os.Stdout, "Rate: %d objects/sec\n", int(r))
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
