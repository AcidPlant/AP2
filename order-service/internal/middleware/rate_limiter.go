package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiterConfig struct {
	MaxRequests int
	Window      time.Duration
}

func RateLimiter(rdb *redis.Client, cfg RateLimiterConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", clientIP)
		ctx := context.Background()

		pipe := rdb.Pipeline()
		incrResult := pipe.Incr(ctx, key)
		pipe.ExpireNX(ctx, key, cfg.Window)
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[RateLimiter] redis error for %s: %v – allowing request", clientIP, err)
			c.Next()
			return
		}

		count := incrResult.Val()

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.MaxRequests))
		remaining := cfg.MaxRequests - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if int(count) > cfg.MaxRequests {
			ttl, _ := rdb.TTL(ctx, key).Result()
			c.Header("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"limit":       cfg.MaxRequests,
				"window":      cfg.Window.String(),
				"retry_after": ttl.String(),
			})
			return
		}

		c.Next()
	}
}
