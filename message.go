package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
)

// SendDelayMsgByArgs 发送延时消息（支持 context）
func (g *RabbitMQ) SendDelayMsgByArgs(ctx context.Context, exchangeName, routingKey string, msg amqp.Delivery, delay int32, args *conf.MQArgs) error {
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
	defer func() {
		if err := channel.Close(); err != nil {
			g.logger.Error("关闭 channel 失败", logger.Error(err))
		}
	}()
	// 合并参数
	if args == nil {
		args = &conf.MQArgs{}
	}
	(*args)["x-dead-letter-exchange"] = exchangeName
	(*args)["x-dead-letter-routing-key"] = routingKey

	// 声明延迟队列名称
	queueName := fmt.Sprintf("%s_queue_delay", exchangeName)
	// 声明延时队列
	_, err = channel.QueueDeclarePassive(
		queueName, true, false, false, false,
		args.ToTable(),
	)

	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			channel, err = g.conn.Channel()
			// 3. 确保 channel 被关闭（使用 defer）
			defer func() {
				if err := channel.Close(); err != nil {
					g.logger.Error("关闭 channel 失败", logger.Error(err))
				}
			}()
			if err != nil {
				return err
			}
			_, err = channel.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				args.ToTable(),
			)
			if err != nil {
				return err
			}
		} else {
			g.logger.Error("打开 channel 失败", logger.Error(err))
			return e
		}
	}
	defer func(channel *amqp.Channel) {
		err := channel.Close()
		if err != nil {
			g.logger.Error("关闭 channel 失败", logger.Error(err))
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

// SendDelayMsg 发送延时消息（支持 context）
func (g *RabbitMQ) SendDelayMsg(ctx context.Context, exchangeName, routingKey string, msg amqp.Delivery, delay int32) error {
	return g.SendDelayMsgByArgs(ctx, exchangeName, routingKey, msg, delay, nil)
}

// SendDelayMsgByArgsCompat 兼容旧版本的延时消息发送（不推荐使用）
// Deprecated: 使用 SendDelayMsgByArgs 并传入 context
func (g *RabbitMQ) SendDelayMsgByArgsCompat(exchangeName, routingKey string, msg amqp.Delivery, delay int32, args *conf.MQArgs) error {
	return g.SendDelayMsgByArgs(context.Background(), exchangeName, routingKey, msg, delay, args)
}

// SendDelayMsgCompat 兼容旧版本的延时消息发送（不推荐使用）
// Deprecated: 使用 SendDelayMsg 并传入 context
func (g *RabbitMQ) SendDelayMsgCompat(exchangeName, routingKey string, msg amqp.Delivery, delay int32) error {
	return g.SendDelayMsg(context.Background(), exchangeName, routingKey, msg, delay)
}
