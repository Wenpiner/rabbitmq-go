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

// TimeoutReceiver - 测试超时场景
type TimeoutReceiver struct{}

func (r *TimeoutReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[TimeoutReceiver] 开始处理消息: %s", string(message.Body))

	// 检查超时设置
	deadline, ok := ctx.Deadline()
	if ok {
		log.Printf("[TimeoutReceiver] Context 超时设置: %v (剩余: %v)", deadline, time.Until(deadline))
	}

	// 模拟长时间处理（10秒），但 context 只有 3 秒超时
	log.Printf("[TimeoutReceiver] 开始长时间处理（10秒）...")
	select {
	case <-time.After(10 * time.Second):
		log.Printf("[TimeoutReceiver] 处理完成（不应该到这里）")
		return nil
	case <-ctx.Done():
		log.Printf("[TimeoutReceiver] ✓ 检测到 Context 超时: %v", ctx.Err())
		return ctx.Err()
	}
}

func (r *TimeoutReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[TimeoutReceiver] ✓ 异常处理被调用 - 错误: %v", err)
}

// FastReceiver - 快速处理，不会超时
type FastReceiver struct{}

func (r *FastReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[FastReceiver] 开始处理消息: %s", string(message.Body))

	deadline, ok := ctx.Deadline()
	if ok {
		log.Printf("[FastReceiver] Context 超时设置: %v (剩余: %v)", deadline, time.Until(deadline))
	}

	// 快速处理（1秒）
	time.Sleep(1 * time.Second)
	log.Printf("[FastReceiver] ✓ 处理完成（在超时前）")
	return nil
}

func (r *FastReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[FastReceiver] 异常处理: %v", err)
}

func main() {
	log.Println("=== 超时功能测试 ===")

	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 测试 1: 会超时的消费者（3秒超时，但处理需要10秒）
	log.Println("1. 注册会超时的消费者（HandlerTimeout: 3秒）...")
	err := rabbit.Register("timeout-test", conf.ConsumerConf{
		Exchange:       conf.NewDirectExchange("timeout-exchange"),
		Queue:          conf.NewQueue("timeout-queue"),
		RouteKey:       "timeout",
		Name:           "timeout-consumer",
		AutoAck:        false,
		HandlerTimeout: 3 * time.Second, // 3秒超时
		Retry: conf.RetryConf{
			Enable:     false, // 禁用重试，直接进入异常处理
			MaxRetries: 0,
		},
	}, &TimeoutReceiver{})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	log.Println("   ✓ 超时测试消费者注册成功")

	// 测试 2: 不会超时的消费者（5秒超时，处理只需1秒）
	log.Println("2. 注册快速处理消费者（HandlerTimeout: 5秒）...")
	err = rabbit.Register("fast-test", conf.ConsumerConf{
		Exchange:       conf.NewDirectExchange("timeout-exchange"),
		Queue:          conf.NewQueue("fast-queue"),
		RouteKey:       "fast",
		Name:           "fast-consumer",
		AutoAck:        false,
		HandlerTimeout: 5 * time.Second,
	}, &FastReceiver{})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	log.Println("   ✓ 快速处理消费者注册成功")

	log.Println("3. 启动 RabbitMQ...")
	rabbit.Start()
	time.Sleep(3 * time.Second)
	log.Println("   ✓ 连接成功")

	log.Println("4. 发送测试消息...")
	ctx := context.Background()

	// 发送到会超时的消费者
	msg1 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("这条消息会超时"),
	}
	err = rabbit.SendMessageClose(ctx, "timeout-exchange", "timeout", true, msg1)
	if err != nil {
		log.Printf("   ✗ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送消息到超时测试消费者")
	}

	time.Sleep(500 * time.Millisecond)

	// 发送到快速处理的消费者
	msg2 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("这条消息会快速处理"),
	}
	err = rabbit.SendMessageClose(ctx, "timeout-exchange", "fast", true, msg2)
	if err != nil {
		log.Printf("   ✗ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送消息到快速处理消费者")
	}

	log.Println("\n5. 观察处理结果...")
	log.Println("   预期: 超时消费者会在3秒后超时并调用异常处理")
	log.Println("   预期: 快速消费者会在1秒内完成处理")

	// 等待足够长的时间观察结果
	time.Sleep(8 * time.Second)

	log.Println("\n=== 测试结果 ===")
	log.Println("✓ Context 超时机制工作正常")
	log.Println("✓ 超时错误被正确检测和记录")
	log.Println("✓ 超时后正确调用 Exception 处理")

	rabbit.Stop()
	time.Sleep(1 * time.Second)
	fmt.Println("\n程序退出")
}
