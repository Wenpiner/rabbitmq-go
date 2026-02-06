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

// 示例 5: 并发处理
// 演示如何配置并发处理消息以提高吞吐量

var (
	processedCount int32
	activeWorkers  int32
	maxWorkers     int32
)

func main() {
	log.Println("=== RabbitMQ 并发处理示例 ===")

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

	// 创建处理器 - 模拟耗时操作
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			// 增加活跃工作者计数
			current := atomic.AddInt32(&activeWorkers, 1)
			defer atomic.AddInt32(&activeWorkers, -1)

			// 更新最大并发数
			for {
				max := atomic.LoadInt32(&maxWorkers)
				if current <= max {
					break
				}
				if atomic.CompareAndSwapInt32(&maxWorkers, max, current) {
					break
				}
			}

			// 模拟耗时处理（500ms）
			log.Printf("🔄 [Worker %d] 开始处理: %s (当前并发: %d)",
				current, string(msg.Body()), current)

			time.Sleep(500 * time.Millisecond)

			processed := atomic.AddInt32(&processedCount, 1)
			log.Printf("✅ [Worker %d] 处理完成 (已处理: %d)",
				current, processed)

			return nil
		},
	)

	// 注册消费者 - 配置并发数为 10
	concurrency := 10
	err := client.RegisterConsumer("concurrent-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "concurrent-queue",
			Durable: false,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "concurrent-exchange",
			Type:         "fanout",
			Durable:      false,
		}),
		rabbitmq.WithAutoAck(false),
		rabbitmq.WithPrefetchCount(concurrency), // 预取数量等于并发数
		rabbitmq.WithHandler(handler),
		rabbitmq.WithConcurrency(concurrency), // 设置并发数
		rabbitmq.WithHandlerTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	log.Printf("✅ 消费者已注册 (并发数: %d)\n", concurrency)
	time.Sleep(1 * time.Second)

	// 发送大量消息
	messageCount := 50
	log.Printf("📤 发送 %d 条消息...\n", messageCount)

	messages := make([]amqp.Publishing, messageCount)
	for i := 0; i < messageCount; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Message #%d", i+1)),
		}
	}

	start := time.Now()
	err = client.PublishBatch(ctx, "concurrent-exchange", "", messages)
	if err != nil {
		log.Fatalf("批量发送失败: %v", err)
	}

	log.Printf("✅ 已发送 %d 条消息\n", messageCount)

	// 等待所有消息处理完成
	log.Println("⏳ 等待消息处理...")
	for {
		processed := atomic.LoadInt32(&processedCount)
		if processed >= int32(messageCount) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	elapsed := time.Since(start)
	log.Printf("\n📊 处理完成统计:")
	log.Printf("   总消息数: %d", messageCount)
	log.Printf("   总耗时: %v", elapsed)
	log.Printf("   平均耗时: %v/条", elapsed/time.Duration(messageCount))
	log.Printf("   最大并发数: %d", atomic.LoadInt32(&maxWorkers))
	log.Printf("   吞吐量: %.2f 条/秒", float64(messageCount)/elapsed.Seconds())

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan

	log.Println("🛑 正在优雅关闭...")
}

