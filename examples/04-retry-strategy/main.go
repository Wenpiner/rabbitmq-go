package main

import (
	"context"
	"errors"
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

// 示例 4: 重试策略
// 演示不同的重试策略：指数退避、线性退避、无重试

var (
	processCount int32
	failCount    int32
)

func main() {
	log.Println("=== RabbitMQ 重试策略示例 ===")

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

	// 创建一个会失败的处理器（前3次失败，第4次成功）
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			count := atomic.AddInt32(&processCount, 1)
			retryCount := msg.RetryCount

			log.Printf("📨 处理消息 (第 %d 次尝试，重试次数: %d): %s",
				count, retryCount, string(msg.Body()))

			// 模拟前3次失败
			if retryCount < 3 {
				atomic.AddInt32(&failCount, 1)
				return errors.New("模拟处理失败")
			}

			log.Printf("✅ 消息处理成功！")
			return nil
		},
		rabbitmq.WithErrorHandler(func(ctx context.Context, msg *rabbitmq.Message, err error) {
			log.Printf("❌ 错误处理器: %v (重试次数: %d)", err, msg.RetryCount)
		}),
	)

	// 注册消费者 - 使用指数退避重试策略
	err := client.RegisterConsumer("retry-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "retry-queue",
			Durable: true,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "retry-exchange",
			Type:         "direct",
			Durable:      true,
		}),
		rabbitmq.WithRouteKey("retry.key"),
		rabbitmq.WithAutoAck(false),
		rabbitmq.WithHandler(handler),
		rabbitmq.WithRetryStrategy(&conf.ExponentialRetry{
			MaxRetries:   5,
			InitialDelay: 1 * time.Second,
			Multiplier:   2.0,
			MaxDelay:     30 * time.Second,
			Jitter:       true,
		}),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 发送测试消息
	log.Println("\n📤 发送测试消息...")
	err = client.Publish(ctx, "retry-exchange", "retry.key", amqp.Publishing{
		ContentType:  "text/plain",
		Body:         []byte("Test message with retry"),
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		log.Fatalf("发送消息失败: %v", err)
	}

	log.Println("✅ 消息已发送")
	log.Println("\n⏳ 观察重试过程...")
	log.Println("   预期重试时间: ~1s → ~2s → ~4s → 成功")

	// 等待足够的时间让重试完成
	time.Sleep(15 * time.Second)

	log.Printf("\n📊 统计:")
	log.Printf("   总处理次数: %d", atomic.LoadInt32(&processCount))
	log.Printf("   失败次数: %d", atomic.LoadInt32(&failCount))

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan

	log.Println("🛑 正在优雅关闭...")
}

