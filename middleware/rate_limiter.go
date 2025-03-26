package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// rateLimiterStore holds a map of IP addresses to their rate limiters.
type rateLimiterStore struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
}

var limiterStore = &rateLimiterStore{
	limiters: make(map[string]*rate.Limiter),
}

// getLimiter returns the rate limiter for a given IP, creating one if it doesn't exist.
func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	limiter, exists := s.limiters[ip]
	if !exists {
		// Configure rate: 20 requests per minute with burst capacity of 5.
		limiter = rate.NewLimiter(rate.Every(time.Minute/200), 200)
		s.limiters[ip] = limiter
	}
	return limiter
}

// RateLimitMiddleware limits requests per IP address.
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
		ip := getClientIP(c)
		limiter := limiterStore.getLimiter(ip)
		if !limiter.Allow() {
			logger.Warn("Rate limit exceeded", zap.String("ip", ip))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Try again later."})
			return
		}
		c.Next()
	}
}
