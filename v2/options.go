package v2

import (
	"time"

	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
)

// Options 客户端配置选项
type Options struct {
	// 连接配置
	Config conf.RabbitConf

	// 日志器
	Logger logger.Logger

	// 重连间隔
	ReconnectInterval time.Duration

	// 最大重连次数，0表示无限重试
	MaxReconnectAttempts int

	// 连接超时
	ConnectTimeout time.Duration

	// 心跳检查间隔
	HeartbeatInterval time.Duration

	// 是否自动重连
	AutoReconnect bool
}

// Option 配置选项函数
type Option func(*Options)

// DefaultOptions 返回默认配置
func DefaultOptions() *Options {
	return &Options{
		Logger:               logger.NewDefaultLogger(logger.LevelInfo),
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 0, // 无限重试
		ConnectTimeout:       30 * time.Second,
		HeartbeatInterval:    500 * time.Millisecond,
		AutoReconnect:        true,
	}
}

// WithConfig 设置连接配置
func WithConfig(config conf.RabbitConf) Option {
	return func(o *Options) {
		o.Config = config
	}
}

// WithLogger 设置日志器
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// WithReconnectInterval 设置重连间隔
func WithReconnectInterval(d time.Duration) Option {
	return func(o *Options) {
		o.ReconnectInterval = d
	}
}

// WithMaxReconnectAttempts 设置最大重连次数
func WithMaxReconnectAttempts(n int) Option {
	return func(o *Options) {
		o.MaxReconnectAttempts = n
	}
}

// WithConnectTimeout 设置连接超时
func WithConnectTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ConnectTimeout = d
	}
}

// WithHeartbeatInterval 设置心跳检查间隔
func WithHeartbeatInterval(d time.Duration) Option {
	return func(o *Options) {
		o.HeartbeatInterval = d
	}
}

// WithAutoReconnect 设置是否自动重连
func WithAutoReconnect(auto bool) Option {
	return func(o *Options) {
		o.AutoReconnect = auto
	}
}

// ConsumerOptions 消费者配置
type ConsumerOptions struct {
	// 消费者配置
	Config conf.ConsumerConf

	// 消息处理器
	Handler MessageHandler

	// 处理超时
	HandlerTimeout time.Duration

	// 重试策略
	RetryStrategy conf.RetryStrategy

	// 并发处理数
	Concurrency int
}

// ConsumerOption 消费者配置选项函数
type ConsumerOption func(*ConsumerOptions)

// DefaultConsumerOptions 返回默认消费者配置
func DefaultConsumerOptions() *ConsumerOptions {
	return &ConsumerOptions{
		HandlerTimeout: 30 * time.Second,
		Concurrency:    1,
	}
}

// WithConsumerConfig 设置消费者配置
func WithConsumerConfig(config conf.ConsumerConf) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.Config = config
	}
}

// WithHandler 设置消息处理器
func WithHandler(h MessageHandler) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.Handler = h
	}
}

// WithHandlerTimeout 设置处理超时
func WithHandlerTimeout(d time.Duration) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.HandlerTimeout = d
	}
}

// WithRetryStrategy 设置重试策略
func WithRetryStrategy(s conf.RetryStrategy) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.RetryStrategy = s
	}
}

// WithConcurrency 设置并发处理数
func WithConcurrency(n int) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.Concurrency = n
	}
}

