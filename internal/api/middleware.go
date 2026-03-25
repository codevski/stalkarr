package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Next()
	}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors  = make(map[string]*visitor)
	visitorMu sync.Mutex
)

func getVisitor(ip string, burst int, rps float64) *rate.Limiter {
	visitorMu.Lock()
	defer visitorMu.Unlock()

	key := fmt.Sprintf("%s:%v", ip, rps)
	v, exists := visitors[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rps), burst)
		visitors[key] = &visitor{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func init() {
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			visitorMu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			visitorMu.Unlock()
		}
	}()
}

func rateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := getVisitor(c.ClientIP(), 20, 10)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func strictRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := getVisitor(c.ClientIP(), 10, 0.08)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, please wait"})
			return
		}
		c.Next()
	}
}

func resetRateLimiter() {
	visitorMu.Lock()
	defer visitorMu.Unlock()
	visitors = make(map[string]*visitor)
}
