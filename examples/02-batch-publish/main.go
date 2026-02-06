package main

import (
	"context"
	"fmt"
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

// 示例 2: 批量发布消息
// 演示如何高效地批量发送消息

func main() {
	log.Println("=== RabbitMQ 批量发布示例 ===")

	// 创建客户端
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
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 创建消费者统计接收到的消息
	var receivedCount int
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			receivedCount++
			log.Printf("📨 收到消息 #%d: %s", receivedCount, string(msg.Body()))
			return nil
		},
	)

	err := client.RegisterConsumer("batch-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "batch-queue",
			Durable: false,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "batch-exchange",
			Type:         "fanout",
			Durable:      false,
		}),
		rabbitmq.WithAutoAck(true),
		rabbitmq.WithHandler(handler),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 方式 1: 普通批量发送（高性能，无确认）
	log.Println("\n📤 方式 1: 普通批量发送")
	batchSize := 100
	messages := make([]amqp.Publishing, batchSize)
	for i := 0; i < batchSize; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Batch message %d", i+1)),
		}
	}

	start := time.Now()
	err = client.PublishBatch(ctx, "batch-exchange", "", messages)
	if err != nil {
		log.Fatalf("批量发送失败: %v", err)
	}
	log.Printf("✅ 发送 %d 条消息，耗时: %v", batchSize, time.Since(start))

	time.Sleep(2 * time.Second)

	// 方式 2: 带确认的批量发送（可靠，但较慢）
	log.Println("\n📤 方式 2: 带确认的批量发送")
	confirmMessages := make([]amqp.Publishing, 10)
	for i := 0; i < 10; i++ {
		confirmMessages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Confirmed batch message %d", i+1)),
		}
	}

	start = time.Now()
	err = client.PublishBatchWithConfirm(ctx, "batch-exchange", "", confirmMessages)
	if err != nil {
		log.Fatalf("带确认的批量发送失败: %v", err)
	}
	log.Printf("✅ 发送并确认 %d 条消息，耗时: %v", len(confirmMessages), time.Since(start))

	time.Sleep(2 * time.Second)

	log.Printf("\n📊 总共接收到 %d 条消息", receivedCount)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan

	log.Println("🛑 正在优雅关闭...")
}

