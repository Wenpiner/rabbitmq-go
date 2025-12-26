package conf

import (
	"math"
	"math/rand"
	"time"
)

// RetryStrategy defines the interface for retry strategies
type RetryStrategy interface {
	// CalculateDelay calculates the delay duration for the given retry attempt
	CalculateDelay(retryNum int32) time.Duration

	// ShouldRetry determines whether to retry based on the retry count and error
	ShouldRetry(retryNum int32, err error) bool
}

// NoRetry is a strategy that disables retry
type NoRetry struct{}

func (n *NoRetry) CalculateDelay(retryNum int32) time.Duration {
	return 0
}

func (n *NoRetry) ShouldRetry(retryNum int32, err error) bool {
	return false
}

// LinearRetry implements linear backoff retry strategy
type LinearRetry struct {
	MaxRetries   int32
	InitialDelay time.Duration
}

func (l *LinearRetry) CalculateDelay(retryNum int32) time.Duration {
	return l.InitialDelay * time.Duration(retryNum+1)
}

func (l *LinearRetry) ShouldRetry(retryNum int32, err error) bool {
	if err == nil {
		return false
	}
	return retryNum < l.MaxRetries
}

// ExponentialRetry implements exponential backoff retry strategy
type ExponentialRetry struct {
	MaxRetries   int32
	InitialDelay time.Duration
	Multiplier   float64
	MaxDelay     time.Duration
	Jitter       bool
	rand         *rand.Rand
}

// NewExponentialRetry creates a new exponential retry strategy with random seed
func NewExponentialRetry(maxRetries int32, initialDelay time.Duration, multiplier float64, maxDelay time.Duration, jitter bool) *ExponentialRetry {
	return &ExponentialRetry{
		MaxRetries:   maxRetries,
		InitialDelay: initialDelay,
		Multiplier:   multiplier,
		MaxDelay:     maxDelay,
		Jitter:       jitter,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *ExponentialRetry) CalculateDelay(retryNum int32) time.Duration {
	// Calculate exponential delay: initialDelay * multiplier^retryNum
	delayFloat := float64(e.InitialDelay) * math.Pow(e.Multiplier, float64(retryNum))

	// Apply max delay limit
	if delayFloat > float64(e.MaxDelay) {
		delayFloat = float64(e.MaxDelay)
	}

	// Add random jitter (±25%) to avoid thundering herd
	if e.Jitter && e.rand != nil {
		jitterRange := delayFloat * 0.25
		jitterValue := jitterRange * (e.rand.Float64()*2 - 1)
		delayFloat += jitterValue
	}

	// Ensure delay is at least 0
	if delayFloat < 0 {
		delayFloat = 0
	}

	return time.Duration(delayFloat)
}

func (e *ExponentialRetry) ShouldRetry(retryNum int32, err error) bool {
	if err == nil {
		return false
	}
	return retryNum < e.MaxRetries
}

// CreateRetryStrategy creates a retry strategy based on the configuration
func CreateRetryStrategy(conf RetryConf) RetryStrategy {
	// If retry is disabled or MaxRetries is 0, return NoRetry
	if !conf.Enable || conf.MaxRetries == 0 {
		return &NoRetry{}
	}

	switch conf.Strategy {
	case "linear":
		return &LinearRetry{
			MaxRetries:   conf.MaxRetries,
			InitialDelay: conf.InitialDelay,
		}
	case "exponential":
		return NewExponentialRetry(
			conf.MaxRetries,
			conf.InitialDelay,
			conf.Multiplier,
			conf.MaxDelay,
			conf.Jitter,
		)
	default:
		// Default to exponential backoff
		return NewExponentialRetry(
			conf.MaxRetries,
			conf.InitialDelay,
			conf.Multiplier,
			conf.MaxDelay,
			conf.Jitter,
		)
	}
}
