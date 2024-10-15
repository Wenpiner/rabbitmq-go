package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"strconv"
)

func (g *RabbitMQ) SendDelayMsg(exchangeName, routingKey string, msg amqp.Delivery, delay int32) error {
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

	// 声明延迟队列名称
	queueName := fmt.Sprintf("%s_queue_delay", exchangeName)
	// 声明延时队列
	_, err = channel.QueueDeclarePassive(
		queueName, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    exchangeName,
			"x-dead-letter-routing-key": routingKey,
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
					"x-dead-letter-exchange":    exchangeName,
					"x-dead-letter-routing-key": routingKey,
				},
			)
			if err != nil {
				return err
			}
		} else {
			log.Println("Failed to open a channel: ", err)
			return e
		}
	}
	defer func(channel *amqp.Channel) {
		err := channel.Close()
		if err != nil {
			log.Printf("关闭channel失败 err :%s \n", err)
		}
	}(channel)

	// 发送消息
	err = channel.PublishWithContext(
		context.Background(),
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
