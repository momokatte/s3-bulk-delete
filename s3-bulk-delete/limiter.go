package main

import (
	"math"
	"sync"
	"time"

	backoff "github.com/momokatte/go-backoff"
)

type ConcurrencyFailLimiter struct {
	available    chan struct{}
	active       chan struct{}
	failed       chan struct{}
	restoreDelay int
}

func NewConcurrencyFailLimiter(capacity int, restoreDelay int) *ConcurrencyFailLimiter {
	l := &ConcurrencyFailLimiter{
		available:    make(chan struct{}, capacity),
		active:       make(chan struct{}, capacity),
		failed:       make(chan struct{}, capacity),
		restoreDelay: restoreDelay,
	}
	for i := 0; i < capacity; i += 1 {
		l.available <- struct{}{}
	}
	return l
}

func (l *ConcurrencyFailLimiter) CheckWait() {
	l.active <- <-l.available
	return
}

func (l *ConcurrencyFailLimiter) Report(success bool) {
	if success {
		l.available <- <-l.active

		select {
		case t := <-l.failed:
			go l.restore(t)
		default:
		}
		return
	}
	t := <-l.active
	if len(l.available) == 0 && len(l.active) == 0 {
		go l.restore(t)
		return
	}
	l.failed <- t
}

func (l *ConcurrencyFailLimiter) WaitDone(d time.Duration) {
	lastActive := time.Now()
	for {
		if len(l.available) != cap(l.available) {
			lastActive = time.Now()
		}
		if time.Now().Sub(lastActive) > d {
			return
		}
		time.Sleep(time.Duration(250) * time.Millisecond)
	}
}

func (l *ConcurrencyFailLimiter) restore(t struct{}) {
	time.Sleep(time.Duration(l.restoreDelay) * time.Millisecond)
	l.available <- t
}

type CappedBackoffLimiter struct {
	mu          sync.Mutex
	failCount   uint
	failMax     uint
	backOffFunc func(uint) uint
}

func NewCappedBackoffLimiter(failMax uint, backOffFunc func(uint) uint) (l *CappedBackoffLimiter) {
	l = &CappedBackoffLimiter{
		failMax:     failMax,
		backOffFunc: backOffFunc,
	}
	return
}

func (l *CappedBackoffLimiter) CheckWait() {
	if l.failCount == 0 {
		return
	}
	if sleep := l.backOffFunc(l.failCount); sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
}

func (l *CappedBackoffLimiter) Report(success bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if success && l.failCount > 0 {
		l.failCount -= 1
	} else if !success && l.failCount < l.failMax {
		l.failCount += 1
	}
}

func Pow2Exp(base uint) func(uint) uint {
	return func(failCount uint) uint {
		if failCount == 0 {
			return 0
		}
		exp := backoff.Pow2(failCount)
		if exp > math.MaxUint64/base {
			return math.MaxUint64
		}
		return base * exp
	}
}
