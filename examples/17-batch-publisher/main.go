package main

import (
	"context"
	"fmt"
	"log"

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

	fmt.Println("=== 示例 1: 基本使用 ===")
	publisher := rabbit.NewBatchPublisher("", "batch-queue")

	// 添加消息
	for i := 1; i <= 5; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Message %d", i)),
		}
		if err := publisher.Add(ctx, msg); err != nil {
			log.Fatalf("Failed to add message: %v", err)
		}
	}

	fmt.Printf("已添加 %d 条消息\n", publisher.Len())

	// 手动刷新
	if err := publisher.Flush(ctx); err != nil {
		log.Fatalf("Failed to flush: %v", err)
	}
	fmt.Println("✓ 消息已刷新发送")
	fmt.Printf("剩余消息: %d\n", publisher.Len())

	fmt.Println("\n=== 示例 2: 自动刷新 ===")
	autoPublisher := rabbit.NewBatchPublisher("", "auto-batch-queue").
		SetBatchSize(3).
		SetAutoFlush(true)

	fmt.Println("批次大小: 3, 自动刷新: 启用")

	// 添加消息，每 3 条自动刷新
	for i := 1; i <= 10; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Auto Flush Message %d", i)),
		}
		if err := autoPublisher.Add(ctx, msg); err != nil {
			log.Fatalf("Failed to add message: %v", err)
		}
		fmt.Printf("添加消息 %d, 当前队列: %d\n", i, autoPublisher.Len())
	}

	// 刷新剩余消息
	if err := autoPublisher.Close(ctx); err != nil {
		log.Fatalf("Failed to close: %v", err)
	}
	fmt.Println("✓ 所有消息已发送")

	fmt.Println("\n=== 示例 3: 日志收集系统 ===")
	logPublisher := rabbit.NewBatchPublisher("logs-exchange", "app.logs").
		SetBatchSize(100).
		SetAutoFlush(true)

	defer logPublisher.Close(ctx)

	// 模拟日志收集
	fmt.Println("模拟收集 250 条日志...")
	for i := 1; i <= 250; i++ {
		logEntry := amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(fmt.Sprintf(`{"level":"info","message":"Log entry %d","timestamp":"%d"}`, i, i)),
		}
		if err := logPublisher.Add(ctx, logEntry); err != nil {
			log.Printf("Failed to add log: %v", err)
		}

		// 每 50 条打印一次进度
		if i%50 == 0 {
			fmt.Printf("  已收集 %d 条日志\n", i)
		}
	}

	fmt.Println("✓ 日志收集完成")

	fmt.Println("\n=== 示例 4: 使用不同的刷新模式 ===")
	// 创建批量发送器
	mixedPublisher := rabbit.NewBatchPublisher("", "mixed-queue")

	// 添加一些消息
	for i := 1; i <= 5; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Mixed Message %d", i)),
		}
		mixedPublisher.Add(ctx, msg)
	}

	// 使用普通刷新
	fmt.Println("使用普通刷新...")
	if err := mixedPublisher.Flush(ctx); err != nil {
		log.Printf("Flush failed: %v", err)
	} else {
		fmt.Println("✓ 普通刷新完成")
	}

	// 添加更多消息
	for i := 6; i <= 10; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Mixed Message %d", i)),
		}
		mixedPublisher.Add(ctx, msg)
	}

	// 使用带确认的刷新
	fmt.Println("使用带确认的刷新...")
	if err := mixedPublisher.FlushWithConfirm(ctx); err != nil {
		log.Printf("FlushWithConfirm failed: %v", err)
	} else {
		fmt.Println("✓ 带确认刷新完成")
	}

	// 添加更多消息
	for i := 11; i <= 15; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Mixed Message %d", i)),
		}
		mixedPublisher.Add(ctx, msg)
	}

	// 使用事务刷新
	fmt.Println("使用事务刷新...")
	if err := mixedPublisher.FlushTx(ctx); err != nil {
		log.Printf("FlushTx failed: %v", err)
	} else {
		fmt.Println("✓ 事务刷新完成")
	}

	fmt.Println("\n✓ 所有示例完成！")
}
