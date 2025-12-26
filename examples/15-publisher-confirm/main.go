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
		Host:     "127.0.0.1",
		Port:     5672,
		Username: "guest",
		Password: "guest",
		VHost:    "/",
	})
	defer rabbit.Stop()

	ctx := context.Background()

	fmt.Println("=== 示例 1: 使用 WithPublisherConfirm 手动处理确认 ===")
	err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 创建确认通道
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

		// 发送消息
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Message with manual confirmation"),
		}
		if err := ch.PublishWithContext(ctx, "", "confirm-queue", false, false, msg); err != nil {
			return fmt.Errorf("failed to publish: %w", err)
		}

		// 等待确认
		select {
		case confirm := <-confirms:
			if confirm.Ack {
				fmt.Println("✓ 消息已确认")
			} else {
				return fmt.Errorf("message not confirmed")
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("confirmation timeout")
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Failed to publish with confirm: %v", err)
	}

	fmt.Println("\n=== 示例 2: 批量发送并等待所有确认 ===")
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Confirmed message 1")},
		{ContentType: "text/plain", Body: []byte("Confirmed message 2")},
		{ContentType: "text/plain", Body: []byte("Confirmed message 3")},
		{ContentType: "text/plain", Body: []byte("Confirmed message 4")},
		{ContentType: "text/plain", Body: []byte("Confirmed message 5")},
	}

	err = rabbit.PublishBatchWithConfirm(ctx, "", "confirm-queue", messages)
	if err != nil {
		log.Fatalf("Failed to publish batch with confirm: %v", err)
	}
	fmt.Printf("✓ 批量发送 %d 条消息，全部已确认\n", len(messages))

	fmt.Println("\n=== 示例 3: 批量发送大量消息 ===")
	largeMessages := make([]amqp.Publishing, 100)
	for i := 0; i < 100; i++ {
		largeMessages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Large batch message %d", i+1)),
		}
	}

	start := time.Now()
	err = rabbit.PublishBatchWithConfirm(ctx, "", "confirm-queue", largeMessages)
	if err != nil {
		log.Fatalf("Failed to publish large batch: %v", err)
	}
	elapsed := time.Since(start)
	fmt.Printf("✓ 批量发送 %d 条消息，全部已确认，耗时: %v\n", len(largeMessages), elapsed)

	fmt.Println("\n=== 示例 4: 处理超时场景 ===")
	// 创建一个很短的超时 context
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// 等待 context 超时
	time.Sleep(10 * time.Millisecond)

	err = rabbit.PublishBatchWithConfirm(timeoutCtx, "", "confirm-queue", messages)
	if err != nil {
		fmt.Printf("✓ 预期的超时错误: %v\n", err)
	} else {
		log.Fatal("Expected timeout error but got none")
	}

	fmt.Println("\n=== 示例 5: 高级用法 - 批量发送并跟踪每条消息的确认 ===")
	err = rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 创建确认通道
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(messages)))

		// 发送所有消息
		for i, msg := range messages {
			if err := ch.PublishWithContext(ctx, "", "confirm-queue", false, false, msg); err != nil {
				return fmt.Errorf("failed to publish message %d: %w", i+1, err)
			}
			fmt.Printf("  发送消息 %d/%d\n", i+1, len(messages))
		}

		// 等待所有确认
		confirmedCount := 0
		for i := 0; i < len(messages); i++ {
			select {
			case confirm := <-confirms:
				if confirm.Ack {
					confirmedCount++
					fmt.Printf("  确认消息 %d/%d (DeliveryTag: %d)\n", confirmedCount, len(messages), confirm.DeliveryTag)
				} else {
					return fmt.Errorf("message %d not confirmed", i+1)
				}
			case <-time.After(5 * time.Second):
				return fmt.Errorf("confirmation timeout at message %d", i+1)
			}
		}

		fmt.Printf("✓ 所有 %d 条消息已确认\n", confirmedCount)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed in advanced usage: %v", err)
	}

	fmt.Println("\n✓ 所有示例完成！")
}
