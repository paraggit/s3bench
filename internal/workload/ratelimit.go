package workload

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter is an interface for rate limiting
type RateLimiter interface {
	Wait(ctx context.Context) error
	Tokens() float64
}

// FixedRateLimiter implements fixed QPS rate limiting using token bucket
type FixedRateLimiter struct {
	limiter *rate.Limiter
}

// NewFixedRateLimiter creates a new fixed rate limiter
func NewFixedRateLimiter(qps float64) *FixedRateLimiter {
	if qps <= 0 {
		// Unlimited
		return &FixedRateLimiter{
			limiter: rate.NewLimiter(rate.Inf, 1),
		}
	}

	// Allow some burst capacity (10% of QPS or minimum 1)
	burst := int(math.Max(qps*0.1, 1))

	return &FixedRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(qps), burst),
	}
}

// Wait waits until a token is available
func (f *FixedRateLimiter) Wait(ctx context.Context) error {
	return f.limiter.Wait(ctx)
}

// Tokens returns approximate number of available tokens
func (f *FixedRateLimiter) Tokens() float64 {
	return float64(f.limiter.Burst())
}

// PoissonRateLimiter implements Poisson arrival process rate limiting
type PoissonRateLimiter struct {
	lambda float64 // mean rate (events per second)
	rng    *rand.Rand
	mu     sync.Mutex
}

// NewPoissonRateLimiter creates a new Poisson rate limiter
func NewPoissonRateLimiter(lambda float64, seed int64) *PoissonRateLimiter {
	if lambda <= 0 {
		lambda = math.Inf(1) // unlimited
	}

	return &PoissonRateLimiter{
		lambda: lambda,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// Wait waits for the next event based on Poisson distribution
func (p *PoissonRateLimiter) Wait(ctx context.Context) error {
	p.mu.Lock()

	// Generate inter-arrival time from exponential distribution
	// For Poisson process with rate λ, inter-arrival times follow Exp(λ)
	u := p.rng.Float64()
	interArrival := -math.Log(u) / p.lambda

	p.mu.Unlock()

	if math.IsInf(interArrival, 0) || interArrival <= 0 {
		return nil
	}

	duration := time.Duration(interArrival * float64(time.Second))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// Tokens returns the lambda parameter (mean rate)
func (p *PoissonRateLimiter) Tokens() float64 {
	return p.lambda
}

// NoopRateLimiter implements unlimited rate limiting
type NoopRateLimiter struct{}

// NewNoopRateLimiter creates a no-op rate limiter
func NewNoopRateLimiter() *NoopRateLimiter {
	return &NoopRateLimiter{}
}

// Wait immediately returns without waiting
func (n *NoopRateLimiter) Wait(ctx context.Context) error {
	return nil
}

// Tokens returns infinity
func (n *NoopRateLimiter) Tokens() float64 {
	return math.Inf(1)
}

// NewRateLimiter creates a rate limiter based on type and parameters
func NewRateLimiter(rateType string, rateLimit float64, seed int64) RateLimiter {
	if rateLimit <= 0 {
		return NewNoopRateLimiter()
	}

	switch rateType {
	case "fixed":
		return NewFixedRateLimiter(rateLimit)
	case "poisson":
		return NewPoissonRateLimiter(rateLimit, seed)
	default:
		return NewFixedRateLimiter(rateLimit)
	}
}
