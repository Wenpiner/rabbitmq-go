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

func main() {
	// 创建 RabbitMQ 实例
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 示例 1: 使用 WithPublisher 自定义发送逻辑
	fmt.Println("=== 示例 1: WithPublisher ===")
	ctx := context.Background()
	err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 在这个函数中，你可以完全控制 channel
		// channel 会在函数返回后自动关闭
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Hello from WithPublisher!"),
		}
		return ch.PublishWithContext(ctx, "", "test-queue", false, false, msg)
	})
	if err != nil {
		log.Printf("WithPublisher failed: %v", err)
	} else {
		fmt.Println("✓ Message sent successfully with WithPublisher")
	}

	// 示例 2: 使用 Publish 发送单条消息（最简单的方式）
	fmt.Println("\n=== 示例 2: Publish ===")
	err = rabbit.Publish(ctx, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Hello from Publish!"),
	})
	if err != nil {
		log.Printf("Publish failed: %v", err)
	} else {
		fmt.Println("✓ Message sent successfully with Publish")
	}

	// 示例 3: 使用 Publish 发送带超时的消息
	fmt.Println("\n=== 示例 3: Publish with Timeout ===")
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = rabbit.Publish(ctxWithTimeout, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Hello with timeout!"),
	})
	if err != nil {
		log.Printf("Publish with timeout failed: %v", err)
	} else {
		fmt.Println("✓ Message sent successfully with timeout")
	}

	// 示例 4: 使用 PublishBatch 批量发送消息（高性能）
	fmt.Println("\n=== 示例 4: PublishBatch ===")
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Batch Message 1")},
		{ContentType: "text/plain", Body: []byte("Batch Message 2")},
		{ContentType: "text/plain", Body: []byte("Batch Message 3")},
		{ContentType: "text/plain", Body: []byte("Batch Message 4")},
		{ContentType: "text/plain", Body: []byte("Batch Message 5")},
	}

	err = rabbit.PublishBatch(ctx, "", "test-queue", messages)
	if err != nil {
		log.Printf("PublishBatch failed: %v", err)
	} else {
		fmt.Printf("✓ %d messages sent successfully with PublishBatch\n", len(messages))
	}

	// 示例 5: 批量发送大量消息（性能对比）
	fmt.Println("\n=== 示例 5: Performance Comparison ===")

	// 方式 1: 使用旧的 SendMessageClose（每次创建新 channel）
	start := time.Now()
	for i := 0; i < 100; i++ {
		_ = rabbit.SendMessageClose(ctx, "", "test-queue", true, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Old way message %d", i)),
		})
	}
	oldWayDuration := time.Since(start)
	fmt.Printf("Old way (SendMessageClose): 100 messages in %v\n", oldWayDuration)

	// 方式 2: 使用新的 PublishBatch（复用 channel）
	batchMessages := make([]amqp.Publishing, 100)
	for i := 0; i < 100; i++ {
		batchMessages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("New way message %d", i)),
		}
	}

	start = time.Now()
	_ = rabbit.PublishBatch(ctx, "", "test-queue", batchMessages)
	newWayDuration := time.Since(start)
	fmt.Printf("New way (PublishBatch): 100 messages in %v\n", newWayDuration)

	if oldWayDuration > newWayDuration {
		speedup := float64(oldWayDuration) / float64(newWayDuration)
		fmt.Printf("✓ PublishBatch is %.2fx faster!\n", speedup)
	}

	// 示例 6: 使用 WithPublisher 进行批量发送（完全控制）
	fmt.Println("\n=== 示例 6: WithPublisher for Batch ===")
	err = rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 在同一个 channel 中发送多条消息
		for i := 0; i < 10; i++ {
			msg := amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("Custom batch message %d", i)),
			}
			if err := ch.PublishWithContext(ctx, "", "test-queue", false, false, msg); err != nil {
				return fmt.Errorf("failed to send message %d: %w", i, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("WithPublisher batch failed: %v", err)
	} else {
		fmt.Println("✓ 10 messages sent successfully with WithPublisher")
	}

	fmt.Println("\n=== All examples completed ===")
}
