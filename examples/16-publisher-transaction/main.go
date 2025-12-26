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

	fmt.Println("=== 示例 1: 事务模式 - 成功提交 ===")
	err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 发送多条消息
		for i := 1; i <= 3; i++ {
			msg := amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("Transaction Message %d", i)),
			}
			if err := ch.PublishWithContext(ctx, "", "tx-queue", false, false, msg); err != nil {
				return err
			}
			fmt.Printf("  发送消息 %d\n", i)
		}
		return nil // 自动提交
	})
	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}
	fmt.Println("✓ 事务提交成功，所有消息已发送")

	fmt.Println("\n=== 示例 2: 事务模式 - 自动回滚 ===")
	err = rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 发送第一条消息
		msg1 := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Message before error"),
		}
		if err := ch.PublishWithContext(ctx, "", "tx-queue", false, false, msg1); err != nil {
			return err
		}
		fmt.Println("  发送消息 1")

		// 模拟错误
		fmt.Println("  模拟错误发生...")
		return fmt.Errorf("simulated error")
	})
	if err != nil {
		fmt.Printf("✓ 事务回滚成功: %v\n", err)
		fmt.Println("  所有消息都未发送")
	}

	fmt.Println("\n=== 示例 3: 批量事务发送 ===")
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Batch Tx Message 1")},
		{ContentType: "text/plain", Body: []byte("Batch Tx Message 2")},
		{ContentType: "text/plain", Body: []byte("Batch Tx Message 3")},
	}

	err = rabbit.PublishBatchTx(ctx, "", "tx-queue", messages)
	if err != nil {
		log.Fatalf("Batch transaction failed: %v", err)
	}
	fmt.Printf("✓ 批量事务发送成功，%d 条消息已发送\n", len(messages))

	fmt.Println("\n=== 示例 4: 订单系统场景 ===")
	// 模拟订单系统：需要同时发送订单创建和库存扣减消息
	// 要求：要么全部成功，要么全部失败
	orderMessages := []amqp.Publishing{
		{
			ContentType: "application/json",
			Body:        []byte(`{"type":"order_created","order_id":"ORD-12345","amount":99.99}`),
		},
		{
			ContentType: "application/json",
			Body:        []byte(`{"type":"inventory_deducted","product_id":"PROD-456","quantity":1}`),
		},
		{
			ContentType: "application/json",
			Body:        []byte(`{"type":"payment_processed","order_id":"ORD-12345","status":"success"}`),
		},
	}

	err = rabbit.PublishBatchTx(ctx, "order-exchange", "order.events", orderMessages)
	if err != nil {
		log.Printf("订单事件发送失败，已回滚: %v", err)
	} else {
		fmt.Println("✓ 订单事件发送成功")
		fmt.Println("  - 订单创建事件")
		fmt.Println("  - 库存扣减事件")
		fmt.Println("  - 支付处理事件")
		fmt.Println("  所有事件原子性发送完成")
	}

	fmt.Println("\n=== 示例 5: 事务 vs 非事务性能对比 ===")
	testMessages := make([]amqp.Publishing, 10)
	for i := 0; i < 10; i++ {
		testMessages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Performance Test Message %d", i+1)),
		}
	}

	// 非事务批量发送
	fmt.Println("非事务批量发送...")
	err = rabbit.PublishBatch(ctx, "", "perf-queue", testMessages)
	if err != nil {
		log.Printf("Non-tx batch failed: %v", err)
	} else {
		fmt.Println("✓ 非事务批量发送完成")
	}

	// 事务批量发送
	fmt.Println("事务批量发送...")
	err = rabbit.PublishBatchTx(ctx, "", "perf-queue", testMessages)
	if err != nil {
		log.Printf("Tx batch failed: %v", err)
	} else {
		fmt.Println("✓ 事务批量发送完成")
	}

	fmt.Println("\n注意：事务模式会降低性能，仅在需要原子性保证时使用")

	fmt.Println("\n✓ 所有示例完成！")
}
