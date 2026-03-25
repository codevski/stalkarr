package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
)

type loginAttempt struct {
	failures int
	lockedAt *time.Time
	lastSeen time.Time
}

var (
	loginAttempts   = make(map[string]*loginAttempt)
	loginAttemptsMu sync.Mutex
)

func init() {
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			loginAttemptsMu.Lock()
			for ip, a := range loginAttempts {
				if time.Since(a.lastSeen) > lockoutDuration*2 {
					delete(loginAttempts, ip)
				}
			}
			loginAttemptsMu.Unlock()
		}
	}()
}

func recordFailure(ip string) (bool, time.Duration) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	a, exists := loginAttempts[ip]
	if !exists {
		a = &loginAttempt{}
		loginAttempts[ip] = a
	}
	a.lastSeen = time.Now()

	// If currently locked, don't add more failures just report remaining time
	if a.lockedAt != nil {
		remaining := lockoutDuration - time.Since(*a.lockedAt)
		if remaining > 0 {
			return true, remaining
		}
		a.failures = 0
		a.lockedAt = nil
	}

	a.failures++
	if a.failures >= maxFailedAttempts {
		now := time.Now()
		a.lockedAt = &now
		return true, lockoutDuration
	}

	return false, 0
}

func checkLockout(ip string) (bool, time.Duration) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	a, exists := loginAttempts[ip]
	if !exists {
		return false, 0
	}

	if a.lockedAt != nil {
		remaining := lockoutDuration - time.Since(*a.lockedAt)
		if remaining > 0 {
			return true, remaining
		}
		a.failures = 0
		a.lockedAt = nil
	}

	return false, 0
}

func clearFailures(ip string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, ip)
}

func lockoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		locked, remaining := checkLockout(ip)
		if locked {
			mins := int(remaining.Minutes()) + 1
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":            fmt.Sprintf("Too many failed attempts. Try again in %d minute(s).", mins),
				"locked_out":       true,
				"retry_after_secs": int(remaining.Seconds()),
			})
			return
		}
		c.Next()
	}
}
