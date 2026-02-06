package v2

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/logger"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// PublisherFunc 发布函数类型
type PublisherFunc func(ctx context.Context, ch *amqp.Channel) error

// WithPublisher 提供 channel 给发布函数，自动管理生命周期
func (c *Client) WithPublisher(ctx context.Context, fn PublisherFunc) error {
	channel, err := c.getChannel()
	if err != nil {
		return fmt.Errorf("获取 channel 失败: %w", err)
	}
	defer func() {
		if err := channel.Close(); err != nil {
			c.opts.Logger.Error("关闭 channel 失败", logger.Error(err))
		}
	}()

	return fn(ctx, channel)
}

// Publish 发送单条消息
func (c *Client) Publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return c.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	})
}

// PublishWithTrace 发送带追踪信息的消息
func (c *Client) PublishWithTrace(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return c.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 提取追踪信息
		traceInfo := tracing.ExtractFromContext(ctx)
		if traceInfo.TraceID == "" {
			traceInfo.TraceID = tracing.GenerateTraceID()
			traceInfo.SpanID = tracing.GenerateSpanID()
		} else {
			traceInfo.ParentSpanID = traceInfo.SpanID
			traceInfo.SpanID = tracing.GenerateSpanID()
		}

		// 注入到消息 headers
		if msg.Headers == nil {
			msg.Headers = make(amqp.Table)
		}
		tracing.InjectToHeaders(msg.Headers, traceInfo)

		return ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)
	})
}

// PublishBatch 批量发送消息
func (c *Client) PublishBatch(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return c.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		for i, msg := range messages {
			select {
			case <-ctx.Done():
				return fmt.Errorf("批量发送在第 %d/%d 条取消: %w", i, len(messages), ctx.Err())
			default:
			}

			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("发送第 %d/%d 条失败: %w", i, len(messages), err)
			}
		}
		return nil
	})
}

// WithPublisherConfirm 提供启用 confirm 模式的 channel
func (c *Client) WithPublisherConfirm(ctx context.Context, fn PublisherFunc) error {
	return c.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		if err := ch.Confirm(false); err != nil {
			return fmt.Errorf("启用 confirm 模式失败: %w", err)
		}
		return fn(ctx, ch)
	})
}

// PublishWithConfirm 发送单条消息并等待确认
func (c *Client) PublishWithConfirm(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	return c.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

		if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
			return err
		}

		select {
		case confirm := <-confirms:
			if !confirm.Ack {
				return fmt.Errorf("消息未被确认")
			}
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}

// PublishBatchWithConfirm 批量发送消息并等待所有确认
func (c *Client) PublishBatchWithConfirm(ctx context.Context, exchange, routingKey string, messages []amqp.Publishing) error {
	if len(messages) == 0 {
		return nil
	}

	return c.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(messages)))

		// 发送所有消息
		for i, msg := range messages {
			select {
			case <-ctx.Done():
				return fmt.Errorf("批量发送在第 %d/%d 条取消: %w", i, len(messages), ctx.Err())
			default:
			}

			if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
				return fmt.Errorf("发送第 %d/%d 条失败: %w", i, len(messages), err)
			}
		}

		// 等待所有确认
		for i := 0; i < len(messages); i++ {
			select {
			case confirm := <-confirms:
				if !confirm.Ack {
					return fmt.Errorf("第 %d/%d 条消息未被确认", i+1, len(messages))
				}
			case <-ctx.Done():
				return fmt.Errorf("等待第 %d/%d 条确认时取消: %w", i+1, len(messages), ctx.Err())
			}
		}

		return nil
	})
}

// WithPublisherTx 提供启用事务的 channel
func (c *Client) WithPublisherTx(ctx context.Context, fn PublisherFunc) error {
	return c.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		if err := ch.Tx(); err != nil {
			return fmt.Errorf("开启事务失败: %w", err)
		}

		err := fn(ctx, ch)
		if err != nil {
			if rbErr := ch.TxRollback(); rbErr != nil {
				c.opts.Logger.Error("回滚事务失败", logger.Error(rbErr))
			}
			return fmt.Errorf("事务已回滚: %w", err)
		}

		if err := ch.TxCommit(); err != nil {
			return fmt.Errorf("提交事务失败: %w", err)
		}

		return nil
	})
}

