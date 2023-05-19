package server

import (
	"sync"

	"golang.org/x/time/rate"
)

type Throttle interface {
	IsThrottled(ip string) bool
}

// rateLimiter .
type rateLimiter struct {
	requests *rate.Limiter
	ip       map[string]*rate.Limiter
	mu       *sync.RWMutex
}

// NewThrottle .
func NewThrottle() Throttle {
	return &rateLimiter{
		requests: rate.NewLimiter(1, 20),
		ip:       make(map[string]*rate.Limiter),
		mu:       &sync.RWMutex{},
	}
}

func (i *rateLimiter) IsThrottled(ip string) bool {
	defer i.mu.Unlock()
	i.mu.Lock()

	if !i.requests.Allow() {
		return true
	}

	limiter, exists := i.ip[ip]
	if !exists {
		i.ip[ip] = rate.NewLimiter(1, 2)
	}
	return limiter.Allow()
}
