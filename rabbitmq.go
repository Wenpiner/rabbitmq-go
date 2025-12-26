package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
)

type RabbitMQ struct {
	rabbitConf conf.RabbitConf
	conn       *amqp.Connection
	// Registered consumer information for reconnection after disconnection
	consumers map[string]conf.ConsumerConf
	// Consumer RabbitMQ channels
	channels map[string]*amqp.Channel
	// Consumer callback functions (supports both Receive and ReceiveWithContext interfaces)
	receivers map[string]interface{}
	// Connection mutex lock
	mu sync.Mutex
	// Connection success channel
	connected chan interface{}
	// Stop channel
	stop chan interface{}
	// Consumer message channel
	consumer chan amqp.Delivery
	// Current connection status
	isClose bool
	// Whether preparing to stop
	isStop bool
	// Whether already started
	isStart bool
	// Global context for controlling all goroutines
	ctx context.Context
	// Cancel function for stopping all goroutines
	cancel context.CancelFunc
	// WaitGroup for waiting all goroutines to complete
	wg sync.WaitGroup
	// Deprecated API warning map (each API warns only once)
	deprecationWarnings sync.Map
	// Logger for output
	logger logger.Logger
}

// warnDeprecation 打印废弃警告（每个 API 只警告一次）
func (g *RabbitMQ) warnDeprecation(oldAPI, newAPI, migrationURL string) {
	_, loaded := g.deprecationWarnings.LoadOrStore(oldAPI, true)
	if !loaded {
		g.logger.Warn("API 已废弃", logger.String("old_api", oldAPI), logger.String("new_api", newAPI))
		if migrationURL != "" {
			g.logger.Warn("查看迁移指南", logger.String("url", migrationURL))
		}
	}
}

// Channel 获取一个 RabbitMQ channel
// Deprecated: 此方法可能导致 channel 泄漏，请使用 WithPublisher 代替
//
// 迁移指南:
//
//	旧代码:
//	  channel, err := rabbit.Channel()
//	  defer channel.Close()
//	  channel.PublishWithContext(ctx, exchange, route, false, false, msg)
//
//	新代码:
//	  rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
//	      return ch.PublishWithContext(ctx, exchange, route, false, false, msg)
//	  })
//
//	或使用更简单的:
//	  rabbit.Publish(ctx, exchange, route, msg)
func (g *RabbitMQ) Channel() (channel *amqp.Channel, err error) {
	g.warnDeprecation("Channel()", "WithPublisher() or Publish()",
		"https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")
	return g.ChannelByName("")
}

// ChannelByName 获取一个命名的 RabbitMQ channel
// Deprecated: 此方法可能导致 channel 泄漏，请使用 WithPublisher 代替
//
// 迁移指南:
//
//	旧代码:
//	  channel, err := rabbit.ChannelByName("my-channel")
//	  defer channel.Close()
//	  channel.PublishWithContext(ctx, exchange, route, false, false, msg)
//
//	新代码:
//	  rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
//	      return ch.PublishWithContext(ctx, exchange, route, false, false, msg)
//	  })
func (g *RabbitMQ) ChannelByName(name string) (channel *amqp.Channel, err error) {
	g.warnDeprecation("ChannelByName()", "WithPublisher()",
		"https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")

	channelName := name
	if channelName == "" {
		channelName = "未指定"
	}

	if g.IsClose() {
		err = g.connect()
		if err != nil {
			g.logger.Error("重连失败", logger.String("channel", channelName), logger.Error(err))
			return
		}
	}
	// 声明通道
	if channel == nil || channel.IsClosed() {
		channel, err = g.conn.Channel()
		if err != nil {
			g.logger.Error("创建通道失败", logger.String("channel", channelName), logger.Error(err))
			return
		}
	}
	return
}

// SendMessageClose 发送消息并关闭 channel
// Deprecated: 请使用 Publish 代替，功能相同但 API 更简洁
//
// 迁移指南:
//
//	旧代码:
//	  err := rabbit.SendMessageClose(ctx, exchange, route, true, msg)
//
//	新代码:
//	  err := rabbit.Publish(ctx, exchange, route, msg)
func (g *RabbitMQ) SendMessageClose(ctx context.Context, exchange, route string, conn bool, msg amqp.Publishing) error {
	g.warnDeprecation("SendMessageClose()", "Publish()",
		"https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")

	// 内部使用新的 Publish 方法
	return g.Publish(ctx, exchange, route, msg)
}

// SendMessage 发送消息
// Deprecated: 返回 channel 容易导致泄漏，请使用 Publish 代替
//
// 迁移指南:
//
//	旧代码:
//	  channel, err := rabbit.SendMessage(ctx, exchange, route, true, msg)
//	  defer channel.Close()
//
//	新代码:
//	  err := rabbit.Publish(ctx, exchange, route, msg)
func (g *RabbitMQ) SendMessage(ctx context.Context, exchange string, route string, conn bool, msg amqp.Publishing) (
	channel *amqp.Channel, err error,
) {
	g.warnDeprecation("SendMessage()", "Publish()",
		"https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")

	// 内部使用新的 Publish 方法
	err = g.Publish(ctx, exchange, route, msg)
	return nil, err
}

// SendMessageWithTrace 发送消息并自动注入追踪信息
// Deprecated: 请使用 PublishWithTrace 代替
//
// 迁移指南:
//
//	旧代码:
//	  channel, err := rabbit.SendMessageWithTrace(ctx, exchange, route, true, msg)
//	  defer channel.Close()
//
//	新代码:
//	  err := rabbit.PublishWithTrace(ctx, exchange, route, msg)
func (g *RabbitMQ) SendMessageWithTrace(ctx context.Context, exchange string, route string, conn bool, msg amqp.Publishing) (
	channel *amqp.Channel, err error,
) {
	g.warnDeprecation("SendMessageWithTrace()", "PublishWithTrace()",
		"https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")

	// 内部使用新的 PublishWithTrace 方法
	err = g.PublishWithTrace(ctx, exchange, route, msg)
	return nil, err
}

func (g *RabbitMQ) Bind(exchange conf.ExchangeConf, queue conf.QueueConf, routeKey string) (err error) {
	if g.IsClose() {
		return errors.New("rabbitmq connect fail")
	}
	// 声明通道
	var channel *amqp.Channel
	channel, err = g.conn.Channel()
	if err != nil {
		return
	}
	defer func(channel *amqp.Channel) {
		_ = channel.Close()
	}(channel)

	var passiveChannel *amqp.Channel
	passiveChannel, err = g.conn.Channel()
	if err != nil {
		return
	}

	exchangeArgs := exchange.Args
	if exchangeArgs == nil {
		exchangeArgs = conf.MQArgs{}
	}

	// 读取队列
	_, err = passiveChannel.QueueDeclarePassive(
		queue.Name, queue.Durable, queue.AutoDelete, queue.Exclusive, queue.NoWait, exchangeArgs.ToTable(),
	)
	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			// 队列不存在,声明队列
			_, err = channel.QueueDeclare(
				queue.Name, queue.Durable, queue.AutoDelete, queue.Exclusive, queue.NoWait, exchangeArgs.ToTable(),
			)
			if err != nil {
				return
			}
		} else {
			err = e
			return
		}
	}
	passiveChannel, err = g.conn.Channel()
	if err != nil {
		return
	}
	// 绑定队列
	err = passiveChannel.ExchangeDeclarePassive(
		exchange.ExchangeName, exchange.Type, exchange.Durable, exchange.AutoDelete, exchange.Internal, exchange.NoWait,
		exchangeArgs.ToTable(),
	)
	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			// 交换机不存在,声明交换机
			err = channel.ExchangeDeclare(
				exchange.ExchangeName, exchange.Type, exchange.Durable, exchange.AutoDelete, exchange.Internal,
				exchange.NoWait,
				exchangeArgs.ToTable(),
			)
			if err != nil {
				return err
			}
		}
	}

	// 绑定路由
	err = channel.QueueBind(
		queue.Name, routeKey, exchange.ExchangeName, queue.NoWait, exchangeArgs.ToTable(),
	)
	return err

}

func NewRabbitMQ(rabbitConf conf.RabbitConf) *RabbitMQ {
	ctx, cancel := context.WithCancel(context.Background())
	return &RabbitMQ{
		rabbitConf: rabbitConf,
		isClose:    true,
		consumer:   make(chan amqp.Delivery),
		consumers:  make(map[string]conf.ConsumerConf),
		channels:   make(map[string]*amqp.Channel),
		receivers:  make(map[string]interface{}),
		connected:  make(chan interface{}),
		stop:       make(chan interface{}),
		isStop:     false,
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger.NewDefaultLogger(logger.LevelInfo), // 默认使用 Info 级别
	}
}

// SetLogger 设置自定义日志器。
//
// 参数:
//   - l: 实现了 Logger 接口的日志器
//
// 示例:
//
//	// 使用 NoopLogger 关闭所有日志
//	rabbit.SetLogger(logger.NewNoopLogger())
//
//	// 使用 Debug 级别的 DefaultLogger
//	rabbit.SetLogger(logger.NewDefaultLogger(logger.LevelDebug))
//
//	// 使用自定义日志器（如 Zap）
//	zapLogger, _ := zap.NewProduction()
//	rabbit.SetLogger(adapters.NewZapAdapter(zapLogger))
func (g *RabbitMQ) SetLogger(l logger.Logger) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.logger = l
}

// GetLogger 获取当前使用的日志器。
//
// 返回:
//   - logger.Logger: 当前的日志器实例
//
// 注意:
//   - 该方法主要用于测试，一般不需要在业务代码中使用
func (g *RabbitMQ) GetLogger() logger.Logger {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.logger
}

func (g *RabbitMQ) onChan() {
	// 设置心跳
	ticker := time.NewTicker(500 * time.Millisecond)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-g.ctx.Done():
			// Context 被取消，停止监控
			g.logger.Debug("监控协程停止")
			return
		case <-ticker.C:
			// 检查连接是否断开
			if g.IsClose() && !g.isStop {
				// 连接断开，进行重连
				go g.connect()
			}
		case <-g.connected:
			// 连接成功，进行消费者注册
			g.logger.Info("RabbitMQ 连接成功，开始注册消费者")
			g.registerAll()
		case <-g.stop:
			{
				// 停止
				// 关闭所有消费者
				g.logger.Info("停止 RabbitMQ", logger.Int("channels", len(g.channels)))
				for key, channel := range g.channels {
					if channel != nil && !channel.IsClosed() {
						g.logger.Debug("关闭消费者通道", logger.String("key", key))
						_ = channel.Close()
					}
				}
				g.logger.Info("关闭 RabbitMQ 连接")
				if g.conn != nil && !g.conn.IsClosed() {
					_ = g.conn.Close()
				}
			}
			return
		}
	}
}

func (g *RabbitMQ) registerAll() {
	for key, consumer := range g.consumers {
		e := g.register(key, consumer, g.receivers[key])
		if e != nil {
			g.logger.Error("注册消费者失败", logger.Error(e))
		}
	}
}

func (g *RabbitMQ) register(key string, consumer conf.ConsumerConf, receiver interface{}) error {
	// 判断是否存在消费者
	if v, ok := g.channels[key]; ok && v != nil && !v.IsClosed() {
		return nil
	}
	g.channels[key] = nil
	g.receivers[key] = receiver
	g.consumers[key] = consumer
	if g.IsClose() {
		if !g.isStart {
			return nil
		}
		g.logger.Error("RabbitMQ 连接失败")
		return errors.New("rabbitmq connect fail")
	}
	// 创建消费者通道
	channel, err := g.ChannelByName(key)
	if err != nil {
		g.logger.Error("创建消费者通道失败", logger.String("key", key), logger.Error(err))
		return err
	} else {
		g.logger.Debug("消费者通道创建成功", logger.String("key", key))
	}
	g.channels[key] = channel
	// 如果存在队列，则先进行队列声明
	if consumer.Exchange.ExchangeName != "" && consumer.Queue.Name != "" {
		err = g.Bind(consumer.Exchange, consumer.Queue, consumer.RouteKey)
		if err != nil {
			g.logger.Error("绑定队列失败", logger.Error(err))
		}
	}
	// 设置QoS
	if consumer.Qos.Enable {
		err = g.channels[key].Qos(consumer.Qos.PrefetchCount, consumer.Qos.PrefetchSize, consumer.Qos.Global)
		if err != nil {
			g.logger.Error("设置 QoS 失败", logger.String("key", key), logger.Error(err))
			return err
		}
		g.logger.Debug("QoS 设置成功", logger.String("key", key), logger.Int("prefetch_count", consumer.Qos.PrefetchCount))
	}
	// 注册消费者
	msgs, err := g.channels[key].Consume(
		consumer.Queue.Name,
		consumer.Name,
		consumer.AutoAck,
		consumer.Exclusive,
		consumer.NoLocal,
		consumer.NoWait,
		nil,
	)
	if err != nil {
		g.logger.Error("注册消费者失败", logger.Error(err))
		return err
	}
	// 启动消费者 goroutine，使用 WaitGroup 跟踪
	g.wg.Add(1)
	go func(k string) {
		defer g.wg.Done()
		g.logger.Debug("启动消费者协程", logger.String("key", k))

		for {
			select {
			case <-g.ctx.Done():
				// Context 被取消，停止消费
				g.logger.Debug("消费者协程停止", logger.String("key", k))
				return
			case d, ok := <-msgs:
				if !ok {
					// Channel 关闭
					g.logger.Warn("消费者通道关闭", logger.String("key", k))
					if !g.isStop {
						// 重连
						go g.connect()
					}
					return
				}
				// 为每个消息处理创建独立的 goroutine，并使用 WaitGroup 跟踪
				g.wg.Add(1)
				go func(delivery amqp.Delivery) {
					defer g.wg.Done()
					g.handler(k, delivery)
				}(d)
			}
		}
	}(key)
	return nil
}

// SendDelayMsgByKey 发送延时消息，通过key（支持 context）
func (g *RabbitMQ) SendDelayMsgByKey(ctx context.Context, key string, msg amqp.Delivery, delay int32) error {
	if g.IsClose() {
		err := g.connect()
		if err != nil {
			return err
		}
	}
	// 声明通道
	var err error
	var channel *amqp.Channel
	channel, err = g.conn.Channel()
	if err != nil {
		return err
	}
	queueName := g.consumers[key].Queue.Name + "_delay"
	// 声明延时队列
	_, err = channel.QueueDeclarePassive(
		queueName, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    g.consumers[key].Exchange.ExchangeName,
			"x-dead-letter-routing-key": g.consumers[key].RouteKey,
		},
	)

	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			channel, err = g.conn.Channel()
			if err != nil {
				return err
			}
			_, err = channel.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				amqp.Table{
					"x-dead-letter-exchange":    g.consumers[key].Exchange.ExchangeName,
					"x-dead-letter-routing-key": g.consumers[key].RouteKey,
				},
			)
			if err != nil {
				return err
			}
		} else {
			g.logger.Error("打开通道失败", logger.Error(err))
			return e
		}
	}
	defer func(channel *amqp.Channel) {
		err := channel.Close()
		if err != nil {
			g.logger.Error("关闭通道失败", logger.Error(err))
		}
	}(channel)

	// 发送消息 - 使用传入的 context
	err = channel.PublishWithContext(
		ctx, // 使用传入的 context 替代 context.Background()
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			Headers:         msg.Headers,
			ContentType:     msg.ContentType,
			ContentEncoding: msg.ContentEncoding,
			DeliveryMode:    msg.DeliveryMode,
			Priority:        msg.Priority,
			CorrelationId:   msg.CorrelationId,
			ReplyTo:         msg.ReplyTo,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			UserId:          msg.UserId,
			AppId:           msg.AppId,
			Body:            msg.Body,
			Expiration:      strconv.Itoa(int(delay)),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// SendDelayMsgByKeyCompat 兼容旧版本的延时消息发送（不推荐使用）
// Deprecated: 使用 SendDelayMsgByKey 并传入 context
func (g *RabbitMQ) SendDelayMsgByKeyCompat(key string, msg amqp.Delivery, delay int32) error {
	return g.SendDelayMsgByKey(context.Background(), key, msg, delay)
}

func funcName(key string, g *RabbitMQ, d amqp.Delivery, channel *amqp.Channel) {
	// 判断是否需要ack
	if !g.consumers[key].AutoAck {
		// Check if channel is still open before attempting to ACK
		if channel == nil || channel.IsClosed() {
			g.logger.Warn("ACK 跳过，通道已关闭", logger.String("key", key))
			return
		}

		// 手动ack
		err := d.Ack(false)
		if err != nil {
			// Check if error is due to closed channel/connection
			if errors.Is(err, amqp.ErrClosed) {
				g.logger.Warn("ACK 失败，连接已关闭", logger.String("key", key))
			} else {
				var amqpErr *amqp.Error
				if errors.As(err, &amqpErr) && (amqpErr.Code == amqp.ChannelError || amqpErr.Code == amqp.ConnectionForced) {
					g.logger.Warn("ACK 失败，连接已关闭", logger.String("key", key))
				} else {
					g.logger.Error("ACK 失败", logger.Error(err))
				}
			}
		}
	}
}

func (g *RabbitMQ) Register(key string, consumer conf.ConsumerConf, receiver interface{}) error {
	// 注册消费者
	return g.register(key, consumer, receiver)

}

// getRetryStrategy returns the retry strategy for the given consumer key
// Priority: 1. Custom strategy from ReceiveWithRetry interface
//  2. Strategy from consumer configuration
//  3. Legacy linear retry for backward compatibility
func (g *RabbitMQ) getRetryStrategy(key string) conf.RetryStrategy {
	// 1. Check if receiver implements custom retry strategy interface
	if receiver, ok := g.receivers[key].(conf.ReceiveWithRetry); ok {
		return receiver.GetRetryStrategy()
	}

	// 2. Get retry configuration from consumer
	consumer, exists := g.consumers[key]
	if !exists {
		// Fallback to legacy linear retry for backward compatibility
		return &conf.LinearRetry{
			MaxRetries:   3,
			InitialDelay: 3 * time.Second,
		}
	}

	retryConf := consumer.Retry

	// 3. Check if retry configuration is zero value (not configured)
	// Use legacy linear retry for backward compatibility
	if retryConf == (conf.RetryConf{}) {
		return &conf.LinearRetry{
			MaxRetries:   3,
			InitialDelay: 3 * time.Second,
		}
	}

	// 4. Create strategy from configuration
	return conf.CreateRetryStrategy(retryConf)
}

// getHandlerTimeout 获取指定消费者的处理超时时间
func (g *RabbitMQ) getHandlerTimeout(key string) time.Duration {
	if consumer, ok := g.consumers[key]; ok {
		return consumer.GetHandlerTimeout()
	}
	return 30 * time.Second // 默认值
}

// StartWithContext 启动 RabbitMQ 连接，支持 context 超时控制
func (g *RabbitMQ) StartWithContext(ctx context.Context) error {
	g.isStart = true
	g.logger.Info("RabbitMQ 启动")

	// 启动监控 goroutine
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.onChan()
	}()

	// 等待连接成功或超时
	errChan := make(chan error, 1)
	go func() {
		errChan <- g.connect()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("start timeout: %w", ctx.Err())
	}
}

// Start 启动 RabbitMQ 连接（兼容旧版本）
func (g *RabbitMQ) Start() {
	g.isStart = true
	// 输出日志
	g.logger.Info("RabbitMQ 启动")
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.onChan()
	}()
	// 等待连接成功
	err := g.connect()
	if err != nil {
		return
	}
}

// StopWithContext 优雅关闭 RabbitMQ，等待所有消息处理完成或超时
func (g *RabbitMQ) StopWithContext(ctx context.Context) error {
	g.logger.Info("优雅停止 RabbitMQ")
	g.isStop = true

	// 取消全局 context，通知所有 goroutine 停止
	g.cancel()

	// 等待所有 goroutine 完成或超时
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		g.logger.Info("所有协程已停止")
	case <-ctx.Done():
		g.logger.Warn("优雅停止超时，强制停止")
		return fmt.Errorf("graceful shutdown timeout: %w", ctx.Err())
	}

	// 发送停止信号
	select {
	case g.stop <- true:
	default:
	}

	return nil
}

// Stop 停止 RabbitMQ（兼容旧版本）
func (g *RabbitMQ) Stop() {
	g.isStop = true
	g.cancel()
	select {
	case g.stop <- true:
	default:
	}
}

func (g *RabbitMQ) IsClose() bool {
	return g.conn == nil || g.conn.IsClosed()
}

func (g *RabbitMQ) connect() error {
	lock := g.mu.TryLock()
	if !lock {
		return nil
	}
	defer g.mu.Unlock()
	for g.IsClose() {
		// 检查 context 是否被取消
		select {
		case <-g.ctx.Done():
			g.logger.Debug("连接被取消")
			return g.ctx.Err()
		default:
		}

		if !g.isClose {
			g.isClose = true
			g.logger.Debug("RabbitMQ 连接关闭")
		}
		var err error
		g.logger.Info("连接 RabbitMQ", logger.String("url", getRabbitURL(g.rabbitConf)))
		if g.rabbitConf.Scheme == "amqps" {
			c := &tls.Config{
				InsecureSkipVerify: true,
			}
			g.conn, err = amqp.DialTLS(getRabbitURL(g.rabbitConf), c)
		} else {
			g.conn, err = amqp.Dial(getRabbitURL(g.rabbitConf))
		}
		if err != nil {
			// 延迟5秒后重试连接
			g.logger.Error("连接 RabbitMQ 失败", logger.Error(err))

			// 使用 context 控制的等待
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-g.ctx.Done():
				g.logger.Debug("重连被取消")
				return g.ctx.Err()
			}
		}
	}
	g.logger.Info("RabbitMQ 连接成功")
	if g.isClose {
		// 尝试发送连接成功信号，但不阻塞
		// 如果没有接收者（例如未调用 Start()），则跳过
		select {
		case g.connected <- true:
		default:
			// 没有接收者，跳过
		}
	}
	g.isClose = false
	return nil
}

func getRabbitURL(rabbitConf conf.RabbitConf) string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%d/%s", rabbitConf.Scheme, rabbitConf.Username, rabbitConf.Password,
		rabbitConf.Host, rabbitConf.Port, rabbitConf.VHost,
	)
}
