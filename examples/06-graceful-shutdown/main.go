package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

// 示例 6: 优雅关闭
// 演示如何在关闭时确保所有消息都被处理完成

var (
	sentCount      int32
	processedCount int32
	processingCount int32
)

func main() {
	log.Println("=== RabbitMQ 优雅关闭示例 ===")

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

	// 创建处理器 - 模拟长时间处理
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			atomic.AddInt32(&processingCount, 1)
			defer atomic.AddInt32(&processingCount, -1)

			log.Printf("🔄 开始处理: %s (正在处理: %d)",
				string(msg.Body()), atomic.LoadInt32(&processingCount))

			// 模拟长时间处理（2秒）
			select {
			case <-time.After(2 * time.Second):
				processed := atomic.AddInt32(&processedCount, 1)
				log.Printf("✅ 处理完成: %s (已完成: %d/%d)",
					string(msg.Body()), processed, atomic.LoadInt32(&sentCount))
				return nil
			case <-ctx.Done():
				log.Printf("⚠️  处理被取消: %s", string(msg.Body()))
				return ctx.Err()
			}
		},
	)

	// 注册消费者
	err := client.RegisterConsumer("shutdown-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "shutdown-queue",
			Durable: false,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "shutdown-exchange",
			Type:         "fanout",
			Durable:      false,
		}),
		rabbitmq.WithAutoAck(false),
		rabbitmq.WithHandler(handler),
		rabbitmq.WithConcurrency(3),
		rabbitmq.WithHandlerTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 发送一些消息
	messageCount := 10
	log.Printf("\n📤 发送 %d 条消息...\n", messageCount)

	for i := 1; i <= messageCount; i++ {
		err := client.Publish(ctx, "shutdown-exchange", "", amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Message #%d", i)),
		})
		if err != nil {
			log.Printf("❌ 发送失败: %v", err)
		} else {
			atomic.AddInt32(&sentCount, 1)
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ 已发送 %d 条消息\n", atomic.LoadInt32(&sentCount))

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("🚀 应用运行中...")
	log.Println("💡 提示: 在消息处理过程中按 Ctrl+C 观察优雅关闭")
	log.Println("   客户端会等待所有正在处理的消息完成后再关闭\n")

	<-sigChan

	log.Println("\n🛑 收到关闭信号，开始优雅关闭...")
	log.Printf("   当前正在处理: %d 条消息", atomic.LoadInt32(&processingCount))
	log.Printf("   已处理: %d/%d 条消息", atomic.LoadInt32(&processedCount), atomic.LoadInt32(&sentCount))

	// 优雅关闭
	shutdownStart := time.Now()
	if err := client.Close(); err != nil {
		log.Printf("❌ 关闭失败: %v", err)
	}
	shutdownDuration := time.Since(shutdownStart)

	log.Printf("\n✅ 优雅关闭完成")
	log.Printf("   关闭耗时: %v", shutdownDuration)
	log.Printf("   最终处理: %d/%d 条消息", atomic.LoadInt32(&processedCount), atomic.LoadInt32(&sentCount))

	if atomic.LoadInt32(&processedCount) == atomic.LoadInt32(&sentCount) {
		log.Println("   🎉 所有消息都已处理完成！")
	} else {
		log.Println("   ⚠️  有消息未处理完成")
	}
}

