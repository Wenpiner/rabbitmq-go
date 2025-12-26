package main

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

type ContextReceiver struct{}

// 实现 ReceiveWithContext 接口
func (r *ContextReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[ContextReceiver] Processing message: %s", string(message.Body))

	// 模拟耗时操作，支持 context 取消
	select {
	case <-time.After(2 * time.Second):
		log.Printf("[ContextReceiver] Message processed successfully")
		return nil
	case <-ctx.Done():
		// Context 被取消或超时
		return fmt.Errorf("processing cancelled: %w", ctx.Err())
	}
}

func (r *ContextReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[ContextReceiver] Exception: %v", err)

	// 可以在这里执行异常处理，也支持 context 超时控制
	// 例如：发送告警、写入死信队列等
}

func main() {
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 注册消费者，设置 5 秒超时
	err := rabbit.Register("context-example", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("example"),
		Queue:          conf.NewQueue("context-queue"),
		RouteKey:       "",
		Name:           "context-consumer",
		AutoAck:        false,
		HandlerTimeout: 5 * time.Second, // 设置超时时间
	}, &ContextReceiver{})

	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	rabbit.Start()

	// 保持运行
	select {}
}
