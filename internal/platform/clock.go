package platform

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type ManualClock struct {
	mu  sync.RWMutex
	now time.Time
}

func NewManualClock(value time.Time) *ManualClock {
	return &ManualClock{now: value.UTC()}
}

func (c *ManualClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *ManualClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}
