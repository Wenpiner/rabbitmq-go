package rabbitmq

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

// Client RabbitMQ 客户端 v2
type Client struct {
	// 配置
	opts *Options

	// 状态机
	sm *StateMachine

	// AMQP 连接
	conn *amqp.Connection

	// 消费者管理
	consumers   map[string]*Consumer
	consumersMu sync.RWMutex

	// Context 控制
	ctx    context.Context
	cancel context.CancelFunc

	// WaitGroup 等待所有 goroutine 完成
	wg sync.WaitGroup

	// 连接锁
	connMu sync.Mutex

	// 重连计数
	reconnectCount int
}

// New 创建新的 RabbitMQ 客户端
func New(opts ...Option) *Client {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		opts:      options,
		sm:        NewStateMachine(),
		consumers: make(map[string]*Consumer),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 注册状态变化监听
	client.sm.OnStateChange(client.onStateChange)

	return client
}

// onStateChange 状态变化处理
func (c *Client) onStateChange(from, to ConnectionState, event Event) {
	c.opts.Logger.Info("状态变化",
		logger.String("from", from.String()),
		logger.String("to", to.String()),
		logger.String("event", event.String()),
	)

	switch to {
	case StateConnected:
		// 连接成功后，重新注册所有消费者
		c.reconnectCount = 0
		go c.restartAllConsumers()

	case StateReconnecting:
		// 开始重连
		if c.opts.AutoReconnect {
			go c.reconnect()
		}

	case StateClosed:
		// 停止所有消费者
		c.stopAllConsumers()
	}
}

// Connect 连接到 RabbitMQ
func (c *Client) Connect(ctx context.Context) error {
	if !c.sm.CanTransition(EventConnect) {
		if c.sm.IsConnected() {
			return nil // 已经连接
		}
		return fmt.Errorf("cannot connect in state: %s", c.sm.Current())
	}

	if err := c.sm.Transition(EventConnect); err != nil {
		return err
	}

	return c.doConnect(ctx)
}

// doConnect 执行连接
func (c *Client) doConnect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	url := c.buildURL()
	c.opts.Logger.Info("正在连接 RabbitMQ", logger.String("url", c.sanitizeURL(url)))

	var conn *amqp.Connection
	var err error

	// 创建带超时的连接
	connectCtx, cancel := context.WithTimeout(ctx, c.opts.ConnectTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		if c.opts.Config.Scheme == "amqps" {
			tlsConfig := &tls.Config{InsecureSkipVerify: true}
			conn, err = amqp.DialTLS(url, tlsConfig)
		} else {
			conn, err = amqp.Dial(url)
		}
		close(done)
	}()

	select {
	case <-connectCtx.Done():
		_ = c.sm.Transition(EventConnectFailed)
		return fmt.Errorf("连接超时: %w", connectCtx.Err())
	case <-done:
		if err != nil {
			_ = c.sm.Transition(EventConnectFailed)
			return fmt.Errorf("连接失败: %w", err)
		}
	}

	c.conn = conn
	_ = c.sm.Transition(EventConnectSuccess)

	// 启动连接监控
	c.wg.Add(1)
	go c.monitorConnection()

	return nil
}

// monitorConnection 监控连接状态
func (c *Client) monitorConnection() {
	defer c.wg.Done()

	notifyClose := c.conn.NotifyClose(make(chan *amqp.Error, 1))

	select {
	case <-c.ctx.Done():
		return
	case err := <-notifyClose:
		if err != nil {
			c.opts.Logger.Warn("连接断开", logger.Error(err))
		}
		if c.sm.CanTransition(EventDisconnect) {
			_ = c.sm.Transition(EventDisconnect)
		}
	}
}

// reconnect 重连逻辑
func (c *Client) reconnect() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if c.sm.IsClosed() {
			return
		}

		c.reconnectCount++

		// 检查最大重连次数
		if c.opts.MaxReconnectAttempts > 0 && c.reconnectCount > c.opts.MaxReconnectAttempts {
			c.opts.Logger.Error("达到最大重连次数，停止重连",
				logger.Int("max_attempts", c.opts.MaxReconnectAttempts))
			return
		}

		c.opts.Logger.Info("尝试重连",
			logger.Int("attempt", c.reconnectCount))

		// 等待重连间隔
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.opts.ReconnectInterval):
		}

		// 转换到连接中状态
		if c.sm.Current() == StateDisconnected {
			if err := c.sm.Transition(EventConnect); err != nil {
				continue
			}
		}

		// 尝试连接
		if err := c.doConnect(c.ctx); err != nil {
			c.opts.Logger.Warn("重连失败", logger.Error(err))
			continue
		}

		// 连接成功
		return
	}
}

// restartAllConsumers 重启所有消费者
func (c *Client) restartAllConsumers() {
	c.consumersMu.RLock()
	defer c.consumersMu.RUnlock()

	for name, consumer := range c.consumers {
		if err := consumer.Restart(c.ctx); err != nil {
			c.opts.Logger.Error("重启消费者失败",
				logger.String("name", name),
				logger.Error(err))
		}
	}
}

// stopAllConsumers 停止所有消费者
func (c *Client) stopAllConsumers() {
	c.consumersMu.RLock()
	defer c.consumersMu.RUnlock()

	for _, consumer := range c.consumers {
		consumer.Stop()
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.sm.IsClosed() {
		return nil
	}

	c.opts.Logger.Info("正在关闭 RabbitMQ 客户端")

	// 取消 context
	c.cancel()

	// 转换到关闭状态
	if err := c.sm.Transition(EventClose); err != nil {
		c.opts.Logger.Warn("状态转换失败", logger.Error(err))
	}

	// 停止所有消费者
	c.stopAllConsumers()

	// 等待所有 goroutine 完成
	c.wg.Wait()

	// 关闭连接
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("关闭连接失败: %w", err)
		}
	}

	c.opts.Logger.Info("RabbitMQ 客户端已关闭")
	return nil
}

// buildURL 构建连接 URL
func (c *Client) buildURL() string {
	cfg := c.opts.Config
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		cfg.Scheme, cfg.Username, cfg.Password,
		cfg.Host, cfg.Port, cfg.VHost)
}

// sanitizeURL 隐藏密码的 URL
func (c *Client) sanitizeURL(url string) string {
	cfg := c.opts.Config
	return fmt.Sprintf("%s://%s:***@%s:%d/%s",
		cfg.Scheme, cfg.Username,
		cfg.Host, cfg.Port, cfg.VHost)
}

// State 获取当前状态
func (c *Client) State() ConnectionState {
	return c.sm.Current()
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.sm.IsConnected()
}

// getChannel 获取一个新的 channel
func (c *Client) getChannel() (*amqp.Channel, error) {
	if !c.sm.IsConnected() {
		return nil, fmt.Errorf("未连接到 RabbitMQ")
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil || c.conn.IsClosed() {
		return nil, fmt.Errorf("连接已关闭")
	}

	return c.conn.Channel()
}

// Bind 绑定队列到交换机
func (c *Client) Bind(exchange conf.ExchangeConf, queue conf.QueueConf, routeKey string) error {
	if !c.sm.IsConnected() {
		return fmt.Errorf("未连接到 RabbitMQ")
	}

	channel, err := c.getChannel()
	if err != nil {
		return err
	}
	defer channel.Close()

	args := exchange.Args
	if args == nil {
		args = conf.MQArgs{}
	}

	// 声明队列
	_, err = channel.QueueDeclare(
		queue.Name, queue.Durable, queue.AutoDelete,
		queue.Exclusive, queue.NoWait, args.ToTable())
	if err != nil {
		return fmt.Errorf("声明队列失败: %w", err)
	}

	// 声明交换机
	err = channel.ExchangeDeclare(
		exchange.ExchangeName, exchange.Type,
		exchange.Durable, exchange.AutoDelete,
		exchange.Internal, exchange.NoWait, args.ToTable())
	if err != nil {
		return fmt.Errorf("声明交换机失败: %w", err)
	}

	// 绑定队列
	err = channel.QueueBind(
		queue.Name, routeKey, exchange.ExchangeName,
		queue.NoWait, args.ToTable())
	if err != nil {
		return fmt.Errorf("绑定队列失败: %w", err)
	}

	return nil
}

// RegisterConsumer 注册消费者
func (c *Client) RegisterConsumer(name string, opts ...ConsumerOption) error {
	options := DefaultConsumerOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.Handler == nil {
		return fmt.Errorf("必须提供消息处理器")
	}

	c.consumersMu.Lock()
	defer c.consumersMu.Unlock()

	if _, exists := c.consumers[name]; exists {
		return fmt.Errorf("消费者已存在: %s", name)
	}

	consumer := newConsumer(c, name, options)
	c.consumers[name] = consumer

	// 如果已连接，立即启动消费者
	if c.sm.IsConnected() {
		if err := consumer.Start(c.ctx); err != nil {
			delete(c.consumers, name)
			return err
		}
	}

	return nil
}

// UnregisterConsumer 注销消费者
func (c *Client) UnregisterConsumer(name string) error {
	c.consumersMu.Lock()
	defer c.consumersMu.Unlock()

	consumer, exists := c.consumers[name]
	if !exists {
		return fmt.Errorf("消费者不存在: %s", name)
	}

	consumer.Stop()
	delete(c.consumers, name)
	return nil
}

// GetConsumer 获取消费者
func (c *Client) GetConsumer(name string) (*Consumer, bool) {
	c.consumersMu.RLock()
	defer c.consumersMu.RUnlock()
	consumer, ok := c.consumers[name]
	return consumer, ok
}

// ConsumerCount 获取消费者数量
func (c *Client) ConsumerCount() int {
	c.consumersMu.RLock()
	defer c.consumersMu.RUnlock()
	return len(c.consumers)
}

