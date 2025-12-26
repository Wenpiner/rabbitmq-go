package main

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

func main() {
	log.Println("=== 废弃 API 警告示例 ===")
	log.Println()

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

	// 准备测试消息
	msg := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("test message"),
		Timestamp:   time.Now(),
	}

	log.Println("1. 使用废弃的 Channel() API")
	log.Println("   第一次调用会显示警告...")
	channel, err := rabbit.Channel()
	if err != nil {
		log.Printf("   错误: %v", err)
	} else if channel != nil {
		channel.Close()
		log.Println("   ✓ Channel 创建成功（但会显示废弃警告）")
	}
	log.Println()

	log.Println("2. 再次使用 Channel() API")
	log.Println("   第二次调用不会显示警告（每个 API 只警告一次）...")
	channel, err = rabbit.Channel()
	if err != nil {
		log.Printf("   错误: %v", err)
	} else if channel != nil {
		channel.Close()
		log.Println("   ✓ Channel 创建成功（无警告）")
	}
	log.Println()

	log.Println("3. 使用废弃的 ChannelByName() API")
	log.Println("   第一次调用会显示警告...")
	channel, err = rabbit.ChannelByName("test-channel")
	if err != nil {
		log.Printf("   错误: %v", err)
	} else if channel != nil {
		channel.Close()
		log.Println("   ✓ Channel 创建成功（但会显示废弃警告）")
	}
	log.Println()

	log.Println("4. 使用废弃的 SendMessage() API")
	log.Println("   第一次调用会显示警告...")
	_, err = rabbit.SendMessage(ctx, "", "test-queue", true, msg)
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（但会显示废弃警告）")
	}
	log.Println()

	log.Println("5. 使用废弃的 SendMessageClose() API")
	log.Println("   第一次调用会显示警告...")
	err = rabbit.SendMessageClose(ctx, "", "test-queue", true, msg)
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（但会显示废弃警告）")
	}
	log.Println()

	log.Println("6. 使用废弃的 SendMessageWithTrace() API")
	log.Println("   第一次调用会显示警告...")
	_, err = rabbit.SendMessageWithTrace(ctx, "", "test-queue", true, msg)
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（但会显示废弃警告）")
	}
	log.Println()

	log.Println("=== 推荐的新 API ===")
	log.Println()

	log.Println("7. 使用新的 Publish() API（推荐）")
	log.Println("   无废弃警告...")
	err = rabbit.Publish(ctx, "", "test-queue", msg)
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（无警告）")
	}
	log.Println()

	log.Println("8. 使用新的 PublishWithTrace() API（推荐）")
	log.Println("   无废弃警告...")
	err = rabbit.PublishWithTrace(ctx, "", "test-queue", msg)
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（无警告）")
	}
	log.Println()

	log.Println("9. 使用新的 WithPublisher() API（推荐）")
	log.Println("   无废弃警告...")
	err = rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		return ch.PublishWithContext(ctx, "", "test-queue", false, false, msg)
	})
	if err != nil {
		log.Printf("   错误: %v", err)
	} else {
		log.Println("   ✓ 消息发送成功（无警告）")
	}
	log.Println()

	log.Println("=== 总结 ===")
	log.Println("✓ 废弃的 API 仍然可以使用，但会显示警告")
	log.Println("✓ 每个废弃的 API 只会警告一次")
	log.Println("✓ 推荐迁移到新的 API 以消除警告")
	log.Println("✓ 查看迁移指南: https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md")
}
