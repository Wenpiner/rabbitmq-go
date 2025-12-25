package rabbitmq

import (
	"errors"
	"testing"

	"github.com/wenpiner/rabbitmq-go/conf"
)

// TestExponentialRetryCalculation tests exponential backoff delay calculation
func TestExponentialRetryCalculation(t *testing.T) {
	strategy := conf.NewExponentialRetry(5, 1000, 2.0, 300000, false)

	tests := []struct {
		retryNum      int32
		expectedDelay int32
	}{
		{0, 1000},   // 1000 * 2^0 = 1000
		{1, 2000},   // 1000 * 2^1 = 2000
		{2, 4000},   // 1000 * 2^2 = 4000
		{3, 8000},   // 1000 * 2^3 = 8000
		{4, 16000},  // 1000 * 2^4 = 16000
		{5, 32000},  // 1000 * 2^5 = 32000
		{10, 300000}, // Should be capped at MaxDelay
	}

	for _, tt := range tests {
		delay := strategy.CalculateDelay(tt.retryNum)
		if delay != tt.expectedDelay {
			t.Errorf("CalculateDelay(%d) = %d, expected %d", tt.retryNum, delay, tt.expectedDelay)
		}
	}
}

// TestLinearRetryCalculation tests linear backoff delay calculation
func TestLinearRetryCalculation(t *testing.T) {
	strategy := &conf.LinearRetry{
		MaxRetries:   3,
		InitialDelay: 3000,
	}

	tests := []struct {
		retryNum      int32
		expectedDelay int32
	}{
		{0, 3000},  // 3000 * (0+1) = 3000
		{1, 6000},  // 3000 * (1+1) = 6000
		{2, 9000},  // 3000 * (2+1) = 9000
		{3, 12000}, // 3000 * (3+1) = 12000
	}

	for _, tt := range tests {
		delay := strategy.CalculateDelay(tt.retryNum)
		if delay != tt.expectedDelay {
			t.Errorf("CalculateDelay(%d) = %d, expected %d", tt.retryNum, delay, tt.expectedDelay)
		}
	}
}

// TestShouldRetry tests retry decision logic
func TestShouldRetry(t *testing.T) {
	strategy := conf.NewExponentialRetry(3, 1000, 2.0, 300000, false)

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
	expConf := conf.NewExponentialRetryConf(5, 1000, 2.0)
	expStrategy := conf.CreateRetryStrategy(expConf)
	if _, ok := expStrategy.(*conf.ExponentialRetry); !ok {
		t.Error("Expected ExponentialRetry strategy")
	}

	// Test linear strategy
	linearConf := conf.NewLinearRetryConf(3, 3000)
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
		if linear.MaxRetries != 3 || linear.InitialDelay != 3000 {
			t.Errorf("Expected legacy values (3, 3000), got (%d, %d)", linear.MaxRetries, linear.InitialDelay)
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
	customStrategy := conf.NewExponentialRetry(10, 500, 1.5, 60000, true)

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

