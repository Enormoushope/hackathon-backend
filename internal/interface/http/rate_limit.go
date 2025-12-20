// rate_limit.go
// Gin用の簡易レートリミットミドルウェア（IP単位で1分あたりN回まで）
package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu     sync.Mutex
	users  map[string][]time.Time
	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		users:  make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (rl *rateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		rl.mu.Lock()
		times := rl.users[ip]
		// 過去window分だけ残す
		var filtered []time.Time
		for _, t := range times {
			if now.Sub(t) < rl.window {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) >= rl.limit {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "リクエストが多すぎます。しばらく待って再度お試しください。"})
			return
		}
		filtered = append(filtered, now)
		rl.users[ip] = filtered
		rl.mu.Unlock()
		c.Next()
	}
}
