package jobs

import (
	"fmt"
	"sync"
	"time"
)

type CooldownTracker struct {
	mu       sync.RWMutex
	lastRun  map[string]time.Time
	duration time.Duration
}

func NewCooldownTracker(cooldown time.Duration) *CooldownTracker {
	return &CooldownTracker{
		lastRun:  make(map[string]time.Time),
		duration: cooldown,
	}
}

func (c *CooldownTracker) IsReady(instanceID string, episodeID int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := fmt.Sprintf("%s:%d", instanceID, episodeID)
	last, ok := c.lastRun[key]
	if !ok {
		return true
	}
	return time.Since(last) >= c.duration
}

func (c *CooldownTracker) MarkRun(instanceID string, episodeID int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%s:%d", instanceID, episodeID)
	c.lastRun[key] = time.Now()
}

func (c *CooldownTracker) Prune() {
	c.mu.Lock()
	defer c.mu.Unlock()
	cutoff := time.Now().Add(-2 * c.duration)
	for k, t := range c.lastRun {
		if t.Before(cutoff) {
			delete(c.lastRun, k)
		}
	}
}

func (t *CooldownTracker) SetDuration(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.duration = d
}
