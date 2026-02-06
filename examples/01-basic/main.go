package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

// 示例 1: 基础的发布和消费
// 演示如何创建客户端、注册消费者和发布消息

func main() {
	log.Println("=== RabbitMQ 基础示例 ===")

	// 1. 创建客户端
	client := rabbitmq.New(
		rabbitmq.WithConfig(conf.RabbitConf{
			Scheme:   "amqp",
			Host:     "localhost",
			Port:     5672,
			Username: "guest",
			Password: "guest",
			VHost:    "/",
		}),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
		rabbitmq.WithReconnectInterval(5*time.Second),
		rabbitmq.WithAutoReconnect(true),
	)

	// 2. 连接到 RabbitMQ
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Println("✅ 已连接到 RabbitMQ")

	// 3. 创建消息处理器
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			log.Printf("📨 收到消息: %s", string(msg.Body()))
			return nil
		},
		rabbitmq.WithErrorHandler(func(ctx context.Context, msg *rabbitmq.Message, err error) {
			log.Printf("❌ 处理消息失败: %v", err)
		}),
	)

	// 4. 注册消费者
	err := client.RegisterConsumer("basic-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "basic-queue",
			Durable: true,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "basic-exchange",
			Type:         "direct",
			Durable:      true,
		}),
		rabbitmq.WithRouteKey("basic.key"),
		rabbitmq.WithAutoAck(false),
		rabbitmq.WithHandler(handler),
		rabbitmq.WithHandlerTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	log.Println("✅ 消费者已注册")

	// 5. 发布一些测试消息
	go func() {
		time.Sleep(2 * time.Second) // 等待消费者启动

		for i := 1; i <= 5; i++ {
			msg := amqp.Publishing{
				ContentType:  "text/plain",
				Body:         []byte("Hello RabbitMQ #" + string(rune('0'+i))),
				DeliveryMode: amqp.Persistent,
			}

			err := client.Publish(ctx, "basic-exchange", "basic.key", msg)
			if err != nil {
				log.Printf("❌ 发送消息失败: %v", err)
			} else {
				log.Printf("📤 已发送消息 #%d", i)
			}

			time.Sleep(1 * time.Second)
		}
	}()

	// 6. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan

	log.Println("🛑 正在优雅关闭...")
}

