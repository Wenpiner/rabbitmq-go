package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Logger == nil {
		t.Error("Logger 不应为 nil")
	}
	if opts.ReconnectInterval != 5*time.Second {
		t.Errorf("ReconnectInterval = %v, want 5s", opts.ReconnectInterval)
	}
	if opts.MaxReconnectAttempts != 0 {
		t.Errorf("MaxReconnectAttempts = %d, want 0", opts.MaxReconnectAttempts)
	}
	if opts.ConnectTimeout != 30*time.Second {
		t.Errorf("ConnectTimeout = %v, want 30s", opts.ConnectTimeout)
	}
	if opts.HeartbeatInterval != 500*time.Millisecond {
		t.Errorf("HeartbeatInterval = %v, want 500ms", opts.HeartbeatInterval)
	}
	if !opts.AutoReconnect {
		t.Error("AutoReconnect 应为 true")
	}
}

func TestWithConfig(t *testing.T) {
	cfg := conf.RabbitConf{
		Host:     "localhost",
		Port:     5672,
		Username: "guest",
		Password: "guest",
	}

	opts := DefaultOptions()
	WithConfig(cfg)(opts)

	if opts.Config.Host != "localhost" {
		t.Errorf("Config.Host = %s, want localhost", opts.Config.Host)
	}
	if opts.Config.Port != 5672 {
		t.Errorf("Config.Port = %d, want 5672", opts.Config.Port)
	}
}

func TestWithLogger(t *testing.T) {
	l := logger.NewNoopLogger()
	opts := DefaultOptions()
	WithLogger(l)(opts)

	if opts.Logger != l {
		t.Error("Logger 未正确设置")
	}
}

func TestWithReconnectInterval(t *testing.T) {
	opts := DefaultOptions()
	WithReconnectInterval(10 * time.Second)(opts)

	if opts.ReconnectInterval != 10*time.Second {
		t.Errorf("ReconnectInterval = %v, want 10s", opts.ReconnectInterval)
	}
}

func TestWithMaxReconnectAttempts(t *testing.T) {
	opts := DefaultOptions()
	WithMaxReconnectAttempts(5)(opts)

	if opts.MaxReconnectAttempts != 5 {
		t.Errorf("MaxReconnectAttempts = %d, want 5", opts.MaxReconnectAttempts)
	}
}

func TestWithConnectTimeout(t *testing.T) {
	opts := DefaultOptions()
	WithConnectTimeout(60 * time.Second)(opts)

	if opts.ConnectTimeout != 60*time.Second {
		t.Errorf("ConnectTimeout = %v, want 60s", opts.ConnectTimeout)
	}
}

func TestWithHeartbeatInterval(t *testing.T) {
	opts := DefaultOptions()
	WithHeartbeatInterval(1 * time.Second)(opts)

	if opts.HeartbeatInterval != 1*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 1s", opts.HeartbeatInterval)
	}
}

func TestWithAutoReconnect(t *testing.T) {
	opts := DefaultOptions()
	WithAutoReconnect(false)(opts)

	if opts.AutoReconnect {
		t.Error("AutoReconnect 应为 false")
	}
}

func TestOptionsChaining(t *testing.T) {
	cfg := conf.RabbitConf{Host: "test-host"}
	l := logger.NewNoopLogger()

	opts := DefaultOptions()
	WithConfig(cfg)(opts)
	WithLogger(l)(opts)
	WithReconnectInterval(3 * time.Second)(opts)
	WithMaxReconnectAttempts(10)(opts)
	WithAutoReconnect(false)(opts)

	if opts.Config.Host != "test-host" {
		t.Error("Config 未正确设置")
	}
	if opts.Logger != l {
		t.Error("Logger 未正确设置")
	}
	if opts.ReconnectInterval != 3*time.Second {
		t.Error("ReconnectInterval 未正确设置")
	}
	if opts.MaxReconnectAttempts != 10 {
		t.Error("MaxReconnectAttempts 未正确设置")
	}
	if opts.AutoReconnect {
		t.Error("AutoReconnect 未正确设置")
	}
}

// Consumer Options Tests

func TestDefaultConsumerOptions(t *testing.T) {
	opts := DefaultConsumerOptions()

	if opts.HandlerTimeout != 30*time.Second {
		t.Errorf("HandlerTimeout = %v, want 30s", opts.HandlerTimeout)
	}
	if opts.Concurrency != 1 {
		t.Errorf("Concurrency = %d, want 1", opts.Concurrency)
	}
	if opts.Handler != nil {
		t.Error("Handler 应为 nil")
	}
}

func TestWithConsumerConfig(t *testing.T) {
	cfg := conf.ConsumerConf{
		Name:     "test-consumer",
		AutoAck:  true,
		Queue:    conf.QueueConf{Name: "test-queue"},
		Exchange: conf.ExchangeConf{ExchangeName: "test-exchange"},
	}

	opts := DefaultConsumerOptions()
	WithConsumerConfig(cfg)(opts)

	if opts.Config.Name != "test-consumer" {
		t.Errorf("Config.Name = %s, want test-consumer", opts.Config.Name)
	}
	if !opts.Config.AutoAck {
		t.Error("Config.AutoAck 应为 true")
	}
}

// mockHandler 用于测试的 mock 处理器
type mockHandler struct{}

func (m *mockHandler) Handle(ctx context.Context, msg *Message) error {
	return nil
}

func (m *mockHandler) OnError(ctx context.Context, msg *Message, err error) {}

func TestWithHandler(t *testing.T) {
	h := &mockHandler{}
	opts := DefaultConsumerOptions()
	WithHandler(h)(opts)

	if opts.Handler != h {
		t.Error("Handler 未正确设置")
	}
}

func TestWithHandlerTimeout(t *testing.T) {
	opts := DefaultConsumerOptions()
	WithHandlerTimeout(60 * time.Second)(opts)

	if opts.HandlerTimeout != 60*time.Second {
		t.Errorf("HandlerTimeout = %v, want 60s", opts.HandlerTimeout)
	}
}

func TestWithRetryStrategy(t *testing.T) {
	strategy := &conf.LinearRetry{
		MaxRetries:   5,
		InitialDelay: time.Second,
	}

	opts := DefaultConsumerOptions()
	WithRetryStrategy(strategy)(opts)

	if opts.RetryStrategy != strategy {
		t.Error("RetryStrategy 未正确设置")
	}
}

func TestWithConcurrency(t *testing.T) {
	opts := DefaultConsumerOptions()
	WithConcurrency(10)(opts)

	if opts.Concurrency != 10 {
		t.Errorf("Concurrency = %d, want 10", opts.Concurrency)
	}
}

func TestConsumerOptionsChaining(t *testing.T) {
	h := &mockHandler{}
	strategy := &conf.NoRetry{}
	cfg := conf.ConsumerConf{Name: "chained-consumer"}

	opts := DefaultConsumerOptions()
	WithConsumerConfig(cfg)(opts)
	WithHandler(h)(opts)
	WithHandlerTimeout(45 * time.Second)(opts)
	WithRetryStrategy(strategy)(opts)
	WithConcurrency(5)(opts)

	if opts.Config.Name != "chained-consumer" {
		t.Error("Config 未正确设置")
	}
	if opts.Handler != h {
		t.Error("Handler 未正确设置")
	}
	if opts.HandlerTimeout != 45*time.Second {
		t.Error("HandlerTimeout 未正确设置")
	}
	if opts.RetryStrategy != strategy {
		t.Error("RetryStrategy 未正确设置")
	}
	if opts.Concurrency != 5 {
		t.Error("Concurrency 未正确设置")
	}
}

