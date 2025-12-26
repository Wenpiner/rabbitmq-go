package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/logger"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// PublisherFunc 是用户的发布函数类型
// 用户只需要实现这个函数，专注于业务逻辑
type PublisherFunc func(ctx context.Context, ch *amqp.Channel) error

// WithPublisher 提供一个 channel 给用户的发布函数
// 自动管理 channel 的创建和关闭，防止泄漏
//
// 参数:
//   - ctx: 上下文，用于超时控制和取消
//   - fn: 用户的发布函数，接收 channel 进行操作
//
// 返回:
//   - error: 连接失败、创建 channel 失败或用户函数返回的错误
//
// 示例:
//
//	err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
//	    return ch.PublishWithContext(ctx, "exchange", "route", false, false, msg)
//	})
func (g *RabbitMQ) WithPublisher(ctx context.Context, fn PublisherFunc) error {
	// 1. 检查连接状态
	if g.IsClose() {
		if err := g.connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	// 2. 创建 channel
	channel, err := g.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// 3. 确保 channel 被关闭（使用 defer）
	defer func() {
		if err := channel.Close(); err != nil {
			g.logger.Error("关闭 channel 失败", logger.Error(err))
		}
	}()

	// 4. 执行用户函数
	return fn(ctx, channel)
}

// Publish 发送单条消息到指定的 exchange 和 routing key
// 这是最常用的发送方法，内部自动管理 channel
//
// 参数:
//   - ctx: 上下文，用于超时控制
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - msg: 要发送的消息
//
// 返回:
//   - error: 发送失败的错误
//
// 示例:
//
//	err := rabbit.Publish(ctx, "my-exchange", "my-route", amqp.Publishing{
//	    ContentType: "text/plain",
//	    Body:        []byte("Hello World"),
//	})
func (g *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	})
}

// PublishBatch 批量发送消息到指定的 exchange 和 routing key
// 所有消息使用同一个 channel，性能优于多次调用 Publish
//
// 参数:
//   - ctx: 上下文，用于超时控制和取消
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - messages: 要发送的消息列表
//
// 返回:
//   - error: 发送失败的错误（包含失败的消息索引）
//
// 注意:
//   - 如果某条消息发送失败，会立即返回错误
//   - 已发送的消息不会回滚
//   - 如果需要事务保证，请使用 PublishBatchTx（阶段 3）
//
// 示例:
//
//	messages := []amqp.Publishing{
//	    {Body: []byte("Message 1")},
//	    {Body: []byte("Message 2")},
//	    {Body: []byte("Message 3")},
//	}
//	err := rabbit.PublishBatch(ctx, "my-exchange", "my-route", messages)
func (g *RabbitMQ) PublishBatch(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		for i, msg := range messages {
			// 检查 context 是否被取消
			select {
			case <-ctx.Done():
				return fmt.Errorf("batch publish cancelled at message %d/%d: %w", i, len(messages), ctx.Err())
			default:
			}

			// 发送消息
			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("failed to publish message %d/%d: %w", i, len(messages), err)
			}
		}
		return nil
	})
}

// PublishWithTrace 发送带追踪信息的消息
// 自动从 context 中提取追踪信息并注入到消息 headers
//
// 参数:
//   - ctx: 上下文，包含追踪信息
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - msg: 要发送的消息
//
// 返回:
//   - error: 发送失败的错误
//
// 追踪行为:
//   - 如果 ctx 中没有 trace ID，会自动生成新的
//   - 如果 ctx 中有 trace ID，会生成新的 span ID
//   - 追踪信息会注入到消息的 Headers 中
//
// 示例:
//
//	// 自动生成追踪信息
//	err := rabbit.PublishWithTrace(ctx, "exchange", "route", msg)
//
//	// 使用已有的追踪信息
//	traceInfo := tracing.TraceInfo{
//	    TraceID: "existing-trace-id",
//	    SpanID:  "parent-span-id",
//	}
//	ctx = tracing.InjectToContext(ctx, traceInfo)
//	err := rabbit.PublishWithTrace(ctx, "exchange", "route", msg)
func (g *RabbitMQ) PublishWithTrace(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 提取追踪信息
		traceInfo := tracing.ExtractFromContext(ctx)

		// 如果没有 trace ID，生成新的
		if traceInfo.TraceID == "" {
			traceInfo.TraceID = tracing.GenerateTraceID()
			traceInfo.SpanID = tracing.GenerateSpanID()
		} else {
			// 生成新的 span ID
			traceInfo.ParentSpanID = traceInfo.SpanID
			traceInfo.SpanID = tracing.GenerateSpanID()
		}

		// 注入到消息 headers
		if msg.Headers == nil {
			msg.Headers = make(amqp.Table)
		}
		tracing.InjectToHeaders(msg.Headers, traceInfo)

		// 记录追踪日志
		g.logger.Debug(tracing.FormatTraceLog(ctx, fmt.Sprintf("发送消息到 exchange: %s, route: %s", exchange, routingKey)))

		// 发送消息
		return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	})
}

// PublishBatchWithTrace 批量发送带追踪信息的消息
// 所有消息共享同一个 trace ID，但每条消息有独立的 span ID
//
// 参数:
//   - ctx: 上下文，包含追踪信息
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - messages: 要发送的消息列表
//
// 返回:
//   - error: 发送失败的错误
//
// 追踪行为:
//   - 所有消息共享同一个 trace ID
//   - 每条消息生成独立的 span ID
//   - 第一条消息的 parent span 是 ctx 中的 span
//   - 后续消息的 parent span 是 ctx 中的 span（平级关系）
//
// 示例:
//
//	messages := []amqp.Publishing{
//	    {Body: []byte("Message 1")},
//	    {Body: []byte("Message 2")},
//	}
//	err := rabbit.PublishBatchWithTrace(ctx, "exchange", "route", messages)
func (g *RabbitMQ) PublishBatchWithTrace(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 提取追踪信息
		traceInfo := tracing.ExtractFromContext(ctx)

		// 如果没有 trace ID，生成新的
		if traceInfo.TraceID == "" {
			traceInfo.TraceID = tracing.GenerateTraceID()
			traceInfo.SpanID = tracing.GenerateSpanID()
		}

		// 批量发送
		for i, msg := range messages {
			// 检查 context 是否被取消
			select {
			case <-ctx.Done():
				return fmt.Errorf("batch publish cancelled at message %d/%d: %w", i, len(messages), ctx.Err())
			default:
			}

			// 为每条消息生成新的 span ID
			msgTraceInfo := tracing.TraceInfo{
				TraceID:      traceInfo.TraceID,
				SpanID:       tracing.GenerateSpanID(),
				ParentSpanID: traceInfo.SpanID,
			}

			// 注入追踪信息
			if msg.Headers == nil {
				msg.Headers = make(amqp.Table)
			}
			tracing.InjectToHeaders(msg.Headers, msgTraceInfo)

			// 发送消息
			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("failed to publish message %d/%d: %w", i, len(messages), err)
			}
		}

		g.logger.Debug(tracing.FormatTraceLog(ctx, fmt.Sprintf("批量发送 %d 条消息到 exchange: %s, route: %s", len(messages), exchange, routingKey)))
		return nil
	})
}

// WithPublisherConfirm 提供一个启用了 confirm 模式的 channel
// 用户可以在函数中使用 NotifyPublish 获取确认
//
// 参数:
//   - ctx: 上下文
//   - fn: 用户的发布函数
//
// 返回:
//   - error: 错误信息
//
// 示例:
//
//	err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
//	    confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
//
//	    err := ch.PublishWithContext(ctx, exchange, route, false, false, msg)
//	    if err != nil {
//	        return err
//	    }
//
//	    select {
//	    case confirm := <-confirms:
//	        if !confirm.Ack {
//	            return fmt.Errorf("message not confirmed")
//	        }
//	    case <-ctx.Done():
//	        return ctx.Err()
//	    }
//
//	    return nil
//	})
func (g *RabbitMQ) WithPublisherConfirm(ctx context.Context, fn PublisherFunc) error {
	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 启用 confirm 模式
		if err := ch.Confirm(false); err != nil {
			return fmt.Errorf("failed to enable confirm mode: %w", err)
		}

		// 执行用户函数
		return fn(ctx, ch)
	})
}

// PublishBatchWithConfirm 批量发送消息并等待所有确认
// 提供可靠性保证，确保所有消息都被 broker 接收
//
// 参数:
//   - ctx: 上下文
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - messages: 要发送的消息列表
//
// 返回:
//   - error: 发送失败或确认失败的错误
//
// 注意:
//   - 会等待所有消息的确认
//   - 如果任何消息未被确认，返回错误
//   - 性能略低于不带确认的批量发送
//
// 示例:
//
//	err := rabbit.PublishBatchWithConfirm(ctx, "exchange", "route", messages)
func (g *RabbitMQ) PublishBatchWithConfirm(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return g.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 创建确认通道
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(messages)))

		// 发送所有消息
		for i, msg := range messages {
			select {
			case <-ctx.Done():
				return fmt.Errorf("batch publish cancelled at message %d/%d: %w", i, len(messages), ctx.Err())
			default:
			}

			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("failed to publish message %d/%d: %w", i, len(messages), err)
			}
		}

		// 等待所有确认
		for i := 0; i < len(messages); i++ {
			select {
			case confirm := <-confirms:
				if !confirm.Ack {
					return fmt.Errorf("message %d/%d not confirmed", i+1, len(messages))
				}
			case <-ctx.Done():
				return fmt.Errorf("waiting for confirmation cancelled at message %d/%d: %w", i+1, len(messages), ctx.Err())
			}
		}

		return nil
	})
}

// WithPublisherTx 提供一个启用了事务模式的 channel
// 用户函数执行成功后自动提交，失败则回滚
//
// 参数:
//   - ctx: 上下文
//   - fn: 用户的发布函数
//
// 返回:
//   - error: 错误信息
//
// 注意:
//   - 事务会降低性能，仅在需要原子性保证时使用
//   - 用户函数返回错误时会自动回滚
//   - 用户函数成功返回时会自动提交
//
// 示例:
//
//	err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
//	    // 发送多条消息
//	    for _, msg := range messages {
//	        if err := ch.PublishWithContext(ctx, exchange, route, false, false, msg); err != nil {
//	            return err // 自动回滚
//	        }
//	    }
//	    return nil // 自动提交
//	})
func (g *RabbitMQ) WithPublisherTx(ctx context.Context, fn PublisherFunc) error {
	return g.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 开启事务
		if err := ch.Tx(); err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// 执行用户函数
		err := fn(ctx, ch)

		if err != nil {
			// 回滚事务
			if rbErr := ch.TxRollback(); rbErr != nil {
				g.logger.Error("回滚事务失败", logger.Error(rbErr))
			}
			return fmt.Errorf("transaction rolled back: %w", err)
		}

		// 提交事务
		if err := ch.TxCommit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	})
}

// PublishBatchTx 在事务中批量发送消息
// 保证所有消息要么全部发送成功，要么全部失败
//
// 参数:
//   - ctx: 上下文
//   - exchange: 交换机名称
//   - routingKey: 路由键
//   - messages: 要发送的消息列表
//
// 返回:
//   - error: 发送失败的错误
//
// 注意:
//   - 提供原子性保证（全部成功或全部失败）
//   - 性能低于非事务批量发送
//   - 适用于需要强一致性的场景
//
// 示例:
//
//	// 这些消息要么全部发送成功，要么全部不发送
//	err := rabbit.PublishBatchTx(ctx, "exchange", "route", messages)
func (g *RabbitMQ) PublishBatchTx(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return g.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		for i, msg := range messages {
			// 检查 context 是否被取消
			select {
			case <-ctx.Done():
				return fmt.Errorf("batch publish cancelled at message %d/%d: %w", i, len(messages), ctx.Err())
			default:
			}

			// 发送消息
			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("failed to publish message %d/%d: %w", i, len(messages), err)
			}
		}
		return nil
	})
}

// BatchPublisher 批量发送辅助器
// 提供更友好的批量发送接口，支持自动刷新
type BatchPublisher struct {
	rabbit     *RabbitMQ
	exchange   string
	routingKey string
	messages   []amqp.Publishing
	batchSize  int
	autoFlush  bool
}

// NewBatchPublisher 创建批量发送器
//
// 参数:
//   - exchange: 交换机名称
//   - routingKey: 路由键
//
// 返回:
//   - *BatchPublisher: 批量发送器实例
//
// 示例:
//
//	publisher := rabbit.NewBatchPublisher("my-exchange", "my-route")
//	defer publisher.Close(ctx) // 确保最后刷新
func (g *RabbitMQ) NewBatchPublisher(exchange, routingKey string) *BatchPublisher {
	return &BatchPublisher{
		rabbit:     g,
		exchange:   exchange,
		routingKey: routingKey,
		messages:   make([]amqp.Publishing, 0),
		batchSize:  100, // 默认批次大小
		autoFlush:  false,
	}
}

// SetBatchSize 设置批次大小
// 当消息数量达到批次大小时，如果启用了 autoFlush，会自动刷新
func (bp *BatchPublisher) SetBatchSize(size int) *BatchPublisher {
	bp.batchSize = size
	return bp
}

// SetAutoFlush 设置是否自动刷新
// 启用后，当消息数量达到批次大小时会自动调用 Flush
func (bp *BatchPublisher) SetAutoFlush(auto bool) *BatchPublisher {
	bp.autoFlush = auto
	return bp
}

// Add 添加消息到批次
// 如果启用了 autoFlush 且达到批次大小，会自动刷新
//
// 参数:
//   - ctx: 上下文（仅在 autoFlush 时使用）
//   - msg: 要添加的消息
//
// 返回:
//   - error: 自动刷新时的错误
func (bp *BatchPublisher) Add(ctx context.Context, msg amqp.Publishing) error {
	bp.messages = append(bp.messages, msg)

	// 检查是否需要自动刷新
	if bp.autoFlush && len(bp.messages) >= bp.batchSize {
		return bp.Flush(ctx)
	}

	return nil
}

// Flush 刷新所有待发送的消息
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - error: 发送失败的错误
func (bp *BatchPublisher) Flush(ctx context.Context) error {
	if len(bp.messages) == 0 {
		return nil
	}

	err := bp.rabbit.PublishBatch(ctx, bp.exchange, bp.routingKey, bp.messages)
	if err != nil {
		return err
	}

	// 清空消息列表
	bp.messages = bp.messages[:0]
	return nil
}

// FlushWithConfirm 刷新所有待发送的消息（带确认）
func (bp *BatchPublisher) FlushWithConfirm(ctx context.Context) error {
	if len(bp.messages) == 0 {
		return nil
	}

	err := bp.rabbit.PublishBatchWithConfirm(ctx, bp.exchange, bp.routingKey, bp.messages)
	if err != nil {
		return err
	}

	bp.messages = bp.messages[:0]
	return nil
}

// FlushTx 刷新所有待发送的消息（事务模式）
func (bp *BatchPublisher) FlushTx(ctx context.Context) error {
	if len(bp.messages) == 0 {
		return nil
	}

	err := bp.rabbit.PublishBatchTx(ctx, bp.exchange, bp.routingKey, bp.messages)
	if err != nil {
		return err
	}

	bp.messages = bp.messages[:0]
	return nil
}

// Len 返回当前待发送的消息数量
func (bp *BatchPublisher) Len() int {
	return len(bp.messages)
}

// Close 关闭批量发送器，刷新所有待发送的消息
func (bp *BatchPublisher) Close(ctx context.Context) error {
	return bp.Flush(ctx)
}
