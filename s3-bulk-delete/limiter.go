package main

import (
	"math"
	"sync"
	"time"

	limiter "github.com/momokatte/go-limiter"
)

type IntervalFailLimiter struct {
	iLim         *limiter.IntervalLimiter
	mu           sync.Mutex
	failCount    int
	failMax      int
	baseInterval time.Duration
	maxInterval  time.Duration
}

func NewIntervalFailLimiter(baseInterval, maxInterval time.Duration) *IntervalFailLimiter {
	l := &IntervalFailLimiter{
		iLim:         limiter.NewIntervalLimiter(baseInterval),
		baseInterval: baseInterval,
		maxInterval:  maxInterval,
	}
	if baseInterval == maxInterval {
		return l
	}
	for gpR2Duration(l.baseInterval, l.failMax) < l.maxInterval {
		l.failMax += 1
	}
	return l
}

func (l *IntervalFailLimiter) CheckWait() {
	l.iLim.CheckWait()
}

func (l *IntervalFailLimiter) Report(success bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if success && l.failCount > 0 {
		l.failCount -= 1
	} else if !success && l.failCount < l.failMax {
		l.failCount += 1
	} else {
		return
	}
	var d time.Duration
	if l.failCount == l.failMax {
		d = l.maxInterval
	} else {
		d = gpR2Duration(l.baseInterval, l.failCount)
	}
	l.iLim.SetInterval(d)
}

func gpR2Duration(scale time.Duration, index int) time.Duration {
	if index == 0 {
		return scale
	}
	return time.Duration(gpR2(scale.Nanoseconds(), index))
}

// GPr2 calculates a value in a geometric progression with the given scale factor and a common ratio of 2.
func gpR2(scale int64, index int) int64 {
	if index == 0 {
		return scale
	}
	rk := pow2int64(uint(index))
	if rk < math.MaxInt64/scale {
		return scale * rk
	}
	return math.MaxInt64
}

/*
Calculate power of 2, but don't go over 2^63.
*/
func pow2int64(exponent uint) int64 {
	if exponent > 63 {
		exponent = 63
	}
	return 1 << exponent
}
