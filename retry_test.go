package rabbitmq

import (
	"errors"
	"testing"
	"time"

	"github.com/wenpiner/rabbitmq-go/conf"
)

// TestExponentialRetryCalculation tests exponential backoff delay calculation
func TestExponentialRetryCalculation(t *testing.T) {
	strategy := conf.NewExponentialRetry(5, time.Second, 2.0, 5*time.Minute, false)

	tests := []struct {
		retryNum      int32
		expectedDelay time.Duration
	}{
		{0, time.Second},                // 1s * 2^0 = 1s
		{1, 2 * time.Second},            // 1s * 2^1 = 2s
		{2, 4 * time.Second},            // 1s * 2^2 = 4s
		{3, 8 * time.Second},            // 1s * 2^3 = 8s
		{4, 16 * time.Second},           // 1s * 2^4 = 16s
		{5, 32 * time.Second},           // 1s * 2^5 = 32s
		{10, 5 * time.Minute},           // Should be capped at MaxDelay
	}

	for _, tt := range tests {
		delay := strategy.CalculateDelay(tt.retryNum)
		if delay != tt.expectedDelay {
			t.Errorf("CalculateDelay(%d) = %v, expected %v", tt.retryNum, delay, tt.expectedDelay)
		}
	}
}

// TestLinearRetryCalculation tests linear backoff delay calculation
func TestLinearRetryCalculation(t *testing.T) {
	strategy := &conf.LinearRetry{
		MaxRetries:   3,
		InitialDelay: 3 * time.Second,
	}

	tests := []struct {
		retryNum      int32
		expectedDelay time.Duration
	}{
		{0, 3 * time.Second},  // 3s * (0+1) = 3s
		{1, 6 * time.Second},  // 3s * (1+1) = 6s
		{2, 9 * time.Second},  // 3s * (2+1) = 9s
		{3, 12 * time.Second}, // 3s * (3+1) = 12s
	}

	for _, tt := range tests {
		delay := strategy.CalculateDelay(tt.retryNum)
		if delay != tt.expectedDelay {
			t.Errorf("CalculateDelay(%d) = %v, expected %v", tt.retryNum, delay, tt.expectedDelay)
		}
	}
}

// TestShouldRetry tests retry decision logic
func TestShouldRetry(t *testing.T) {
	strategy := conf.NewExponentialRetry(3, time.Second, 2.0, 5*time.Minute, false)

	tests := []struct {
		retryNum int32
		err      error
		expected bool
	}{
		{0, errors.New("error"), true},  // First retry
		{1, errors.New("error"), true},  // Second retry
		{2, errors.New("error"), true},  // Third retry
		{3, errors.New("error"), false}, // Max retries reached
		{0, nil, false},                 // No error, no retry
	}

	for _, tt := range tests {
		result := strategy.ShouldRetry(tt.retryNum, tt.err)
		if result != tt.expected {
			t.Errorf("ShouldRetry(%d, %v) = %v, expected %v", tt.retryNum, tt.err, result, tt.expected)
		}
	}
}

// TestCreateRetryStrategy tests strategy creation from configuration
func TestCreateRetryStrategy(t *testing.T) {
	// Test exponential strategy
	expConf := conf.NewExponentialRetryConf(5, time.Second, 2.0)
	expStrategy := conf.CreateRetryStrategy(expConf)
	if _, ok := expStrategy.(*conf.ExponentialRetry); !ok {
		t.Error("Expected ExponentialRetry strategy")
	}

	// Test linear strategy
	linearConf := conf.NewLinearRetryConf(3, 3*time.Second)
	linearStrategy := conf.CreateRetryStrategy(linearConf)
	if _, ok := linearStrategy.(*conf.LinearRetry); !ok {
		t.Error("Expected LinearRetry strategy")
	}

	// Test disabled retry
	disabledConf := conf.RetryConf{Enable: false}
	disabledStrategy := conf.CreateRetryStrategy(disabledConf)
	if _, ok := disabledStrategy.(*conf.NoRetry); !ok {
		t.Error("Expected NoRetry strategy")
	}

	// Test zero MaxRetries
	zeroConf := conf.RetryConf{Enable: true, MaxRetries: 0}
	zeroStrategy := conf.CreateRetryStrategy(zeroConf)
	if _, ok := zeroStrategy.(*conf.NoRetry); !ok {
		t.Error("Expected NoRetry strategy for zero MaxRetries")
	}
}

// TestBackwardCompatibility tests that old code without retry config still works
func TestBackwardCompatibility(t *testing.T) {
	rabbit := NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// Register consumer without retry configuration (old style)
	consumer := conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("test"),
		Queue:    conf.NewQueue("test"),
		RouteKey: "",
		Name:     "test",
		AutoAck:  false,
		// No Retry field set - should use legacy linear retry
	}

	receiver := &TestReceive{}
	rabbit.consumers["test"] = consumer
	rabbit.receivers["test"] = receiver

	// Get retry strategy - should return legacy linear retry
	strategy := rabbit.getRetryStrategy("test")
	if linear, ok := strategy.(*conf.LinearRetry); !ok {
		t.Error("Expected LinearRetry for backward compatibility")
	} else {
		if linear.MaxRetries != 3 || linear.InitialDelay != 3*time.Second {
			t.Errorf("Expected legacy values (3, 3s), got (%d, %v)", linear.MaxRetries, linear.InitialDelay)
		}
	}
}

// CustomReceive implements ReceiveWithRetry interface for testing
type CustomReceive struct {
	TestReceive
	strategy conf.RetryStrategy
}

func (c *CustomReceive) GetRetryStrategy() conf.RetryStrategy {
	return c.strategy
}

// TestCustomRetryStrategy tests custom retry strategy via interface
func TestCustomRetryStrategy(t *testing.T) {
	customStrategy := conf.NewExponentialRetry(10, 500*time.Millisecond, 1.5, time.Minute, true)

	receiver := &CustomReceive{
		strategy: customStrategy,
	}

	// Verify interface implementation
	var _ conf.ReceiveWithRetry = receiver

	// Verify custom strategy is returned
	if receiver.GetRetryStrategy() != customStrategy {
		t.Error("Custom strategy not returned correctly")
	}
}

