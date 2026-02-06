package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	// Handle 处理消息
	Handle(ctx context.Context, msg *Message) error
	// OnError 错误处理
	OnError(ctx context.Context, msg *Message, err error)
}

// Message 消息封装
type Message struct {
	// 原始投递
	Delivery amqp.Delivery
	// 重试次数
	RetryCount int32
	// 消费者名称
	ConsumerName string
	// 追踪信息
	TraceInfo tracing.TraceInfo
}

// Body 获取消息体
func (m *Message) Body() []byte {
	return m.Delivery.Body
}

// Ack 确认消息
func (m *Message) Ack() error {
	return m.Delivery.Ack(false)
}

// Nack 拒绝消息
func (m *Message) Nack(requeue bool) error {
	return m.Delivery.Nack(false, requeue)
}

// Reject 拒绝消息
func (m *Message) Reject(requeue bool) error {
	return m.Delivery.Reject(requeue)
}

// Consumer 消费者
type Consumer struct {
	// 客户端引用
	client *Client

	// 消费者名称
	name string

	// 配置
	opts *ConsumerOptions

	// 当前状态
	state ConsumerState
	mu    sync.RWMutex

	// Channel
	channel *amqp.Channel

	// Context 控制
	ctx    context.Context
	cancel context.CancelFunc

	// WaitGroup
	wg sync.WaitGroup

	// 消息通道
	messages <-chan amqp.Delivery
}

// newConsumer 创建消费者
func newConsumer(client *Client, name string, opts *ConsumerOptions) *Consumer {
	ctx, cancel := context.WithCancel(client.ctx)
	return &Consumer{
		client: client,
		name:   name,
		opts:   opts,
		state:  ConsumerIdle,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动消费者
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.state != ConsumerIdle && c.state != ConsumerStopped {
		c.mu.Unlock()
		return fmt.Errorf("消费者状态无效: %s", c.state)
	}
	c.state = ConsumerStarting
	c.mu.Unlock()

	// 获取 channel
	channel, err := c.client.getChannel()
	if err != nil {
		c.setState(ConsumerStopped)
		return fmt.Errorf("创建 channel 失败: %w", err)
	}
	c.channel = channel

	// 绑定队列（如果配置了交换机）
	cfg := c.opts.Config
	if cfg.Exchange.ExchangeName != "" && cfg.Queue.Name != "" {
		if err := c.client.Bind(cfg.Exchange, cfg.Queue, cfg.RouteKey); err != nil {
			c.setState(ConsumerStopped)
			return fmt.Errorf("绑定队列失败: %w", err)
		}
	}

	// 设置 QoS
	if cfg.Qos.Enable {
		if err := channel.Qos(cfg.Qos.PrefetchCount, cfg.Qos.PrefetchSize, cfg.Qos.Global); err != nil {
			c.setState(ConsumerStopped)
			return fmt.Errorf("设置 QoS 失败: %w", err)
		}
	}

	// 注册消费者
	messages, err := channel.Consume(
		cfg.Queue.Name, c.name,
		cfg.AutoAck, cfg.Exclusive,
		cfg.NoLocal, cfg.NoWait, nil)
	if err != nil {
		c.setState(ConsumerStopped)
		return fmt.Errorf("注册消费者失败: %w", err)
	}
	c.messages = messages

	// 启动消息处理
	c.setState(ConsumerRunning)
	c.wg.Add(1)
	go c.consumeLoop()

	c.client.opts.Logger.Info("消费者已启动",
		logger.String("name", c.name),
		logger.String("queue", cfg.Queue.Name))

	return nil
}

// setState 设置状态
func (c *Consumer) setState(state ConsumerState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// State 获取状态
func (c *Consumer) State() ConsumerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// consumeLoop 消费循环
func (c *Consumer) consumeLoop() {
	defer c.wg.Done()
	defer c.setState(ConsumerStopped)

	for {
		select {
		case <-c.ctx.Done():
			c.client.opts.Logger.Debug("消费者停止",
				logger.String("name", c.name))
			return

		case delivery, ok := <-c.messages:
			if !ok {
				c.client.opts.Logger.Warn("消费者 channel 关闭",
					logger.String("name", c.name))
				return
			}

			// 处理消息
			c.handleMessage(delivery)
		}
	}
}

// handleMessage 处理单条消息
func (c *Consumer) handleMessage(delivery amqp.Delivery) {
	// 准备 context
	timeout := c.opts.HandlerTimeout
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	// 提取追踪信息
	traceInfo := tracing.ExtractFromHeaders(delivery.Headers)
	if traceInfo.TraceID == "" {
		traceInfo.TraceID = tracing.GenerateTraceID()
	}
	traceInfo.ParentSpanID = traceInfo.SpanID
	traceInfo.SpanID = tracing.GenerateSpanID()
	ctx = tracing.InjectToContext(ctx, traceInfo)

	// 获取重试次数
	retryCount := int32(0)
	if v, ok := delivery.Headers["retry_nums"].(int32); ok {
		retryCount = v
	}

	// 封装消息
	msg := &Message{
		Delivery:     delivery,
		RetryCount:   retryCount,
		ConsumerName: c.name,
		TraceInfo:    traceInfo,
	}

	// 调用处理器
	err := c.opts.Handler.Handle(ctx, msg)
	if err != nil {
		c.handleError(ctx, msg, err)
		return
	}

	// 处理成功，确认消息
	if !c.opts.Config.AutoAck {
		if err := delivery.Ack(false); err != nil {
			c.client.opts.Logger.Error("确认消息失败",
				logger.String("name", c.name),
				logger.Error(err))
		}
	}
}

// handleError 处理错误
func (c *Consumer) handleError(ctx context.Context, msg *Message, err error) {
	strategy := c.getRetryStrategy()

	// 检查是否需要重试
	if strategy != nil && strategy.ShouldRetry(msg.RetryCount, err) {
		delay := strategy.CalculateDelay(msg.RetryCount)
		c.client.opts.Logger.Info("消息将重试",
			logger.String("name", c.name),
			logger.Int32("retry", msg.RetryCount+1),
			logger.Duration("delay", delay))

		// 重试逻辑：发送到延迟队列
		if err := c.sendToDelayQueue(ctx, msg, delay); err != nil {
			c.client.opts.Logger.Error("发送到延迟队列失败",
				logger.String("name", c.name),
				logger.Error(err))
		}
	} else {
		// 调用错误处理器
		c.opts.Handler.OnError(ctx, msg, err)
	}

	// 确认原消息
	if !c.opts.Config.AutoAck {
		if err := msg.Delivery.Ack(false); err != nil {
			c.client.opts.Logger.Error("确认消息失败",
				logger.String("name", c.name),
				logger.Error(err))
		}
	}
}

// getRetryStrategy 获取重试策略
func (c *Consumer) getRetryStrategy() conf.RetryStrategy {
	if c.opts.RetryStrategy != nil {
		return c.opts.RetryStrategy
	}

	// 从配置创建
	return conf.CreateRetryStrategy(c.opts.Config.Retry)
}

// sendToDelayQueue 发送到延迟队列
func (c *Consumer) sendToDelayQueue(ctx context.Context, msg *Message, delay time.Duration) error {
	delayQueueName := fmt.Sprintf("%s_delay_%d", c.opts.Config.Queue.Name, delay.Milliseconds())

	channel, err := c.client.getChannel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// 声明延迟队列
	_, err = channel.QueueDeclare(
		delayQueueName, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    c.opts.Config.Exchange.ExchangeName,
			"x-dead-letter-routing-key": c.opts.Config.RouteKey,
			"x-message-ttl":             delay.Milliseconds(),
		})
	if err != nil {
		return fmt.Errorf("声明延迟队列失败: %w", err)
	}

	// 更新重试信息
	headers := msg.Delivery.Headers
	if headers == nil {
		headers = make(amqp.Table)
	}
	headers["retry_nums"] = msg.RetryCount + 1
	headers["last_error"] = msg.Delivery.Body

	// 发送消息
	return channel.PublishWithContext(ctx, "", delayQueueName, false, false, amqp.Publishing{
		ContentType:  msg.Delivery.ContentType,
		Body:         msg.Delivery.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
	})
}

// Stop 停止消费者
func (c *Consumer) Stop() {
	c.mu.Lock()
	if c.state != ConsumerRunning {
		c.mu.Unlock()
		return
	}
	c.state = ConsumerStopping
	c.mu.Unlock()

	// 取消 context
	c.cancel()

	// 关闭 channel
	if c.channel != nil && !c.channel.IsClosed() {
		_ = c.channel.Close()
	}

	// 等待处理完成
	c.wg.Wait()

	c.client.opts.Logger.Info("消费者已停止",
		logger.String("name", c.name))
}

// Restart 重启消费者
func (c *Consumer) Restart(ctx context.Context) error {
	c.Stop()

	// 重置 context
	c.ctx, c.cancel = context.WithCancel(c.client.ctx)
	c.state = ConsumerIdle

	return c.Start(ctx)
}

