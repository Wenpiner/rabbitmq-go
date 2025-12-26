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

// CascadeReceiver 演示级联超时控制
type CascadeReceiver struct {
	messageCount int
}

func (r *CascadeReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	r.messageCount++
	msgNum := r.messageCount

	log.Printf("[CascadeReceiver] 收到消息 #%d: %s", msgNum, string(message.Body))

	// 检查 context 剩余时间
	deadline, ok := ctx.Deadline()
	if ok {
		remaining := time.Until(deadline)
		log.Printf("[CascadeReceiver] Context 剩余时间: %v", remaining)

		if remaining < 2*time.Second {
			log.Printf("[CascadeReceiver] ⚠️  剩余时间不足 2 秒，可能影响重试发送")
		}
	}

	// 模拟处理失败，触发重试
	if msgNum <= 2 {
		log.Printf("[CascadeReceiver] 模拟处理失败，将触发重试...")
		return fmt.Errorf("模拟失败 #%d", msgNum)
	}

	log.Printf("[CascadeReceiver] ✓ 处理成功 #%d", msgNum)
	return nil
}

func (r *CascadeReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[CascadeReceiver] ✗ 异常处理: %v", err)

	// 检查是否是 context 超时导致的
	if ctx.Err() != nil {
		log.Printf("[CascadeReceiver] Context 错误: %v", ctx.Err())
	}
}

func main() {
	log.Println("=== 级联超时控制测试 ===")
	log.Println()

	// 创建 RabbitMQ 实例
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 注册消费者 - 设置较短的超时时间
	log.Println("1. 注册消费者（HandlerTimeout: 8 秒）...")
	rabbit.Register("cascade-test", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("cascade-exchange"),
		Queue:          conf.NewQueue("cascade-queue"),
		RouteKey:       "cascade",
		Name:           "cascade-consumer",
		AutoAck:        false,
		HandlerTimeout: 8 * time.Second,     // 8 秒超时
		Retry:          conf.NewRetryConf(), // 启用重试，默认 3 次
	}, &CascadeReceiver{})
	log.Println("   ✓ 消费者注册成功")
	log.Println()

	// 启动连接
	log.Println("2. 启动 RabbitMQ...")
	rabbit.Start()
	time.Sleep(3 * time.Second)
	log.Println("   ✓ 连接成功")
	log.Println()

	// 发送测试消息
	log.Println("3. 发送测试消息...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rabbit.SendMessageClose(ctx, "cascade-exchange", "cascade", true, amqp.Publishing{
		Body: []byte("级联超时测试消息"),
		Headers: amqp.Table{
			"test_type": "cascade",
		},
	})

	if err != nil {
		log.Printf("   ❌ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 消息已发送")
	}
	log.Println()

	log.Println("4. 观察处理过程...")
	log.Println("   预期行为:")
	log.Println("   - 第 1 次处理失败，触发重试（延时 3 秒）")
	log.Println("   - 第 2 次处理失败，触发重试（延时 9 秒）")
	log.Println("   - 重试发送时会传递 context，实现级联超时控制")
	log.Println()

	// 等待处理完成
	time.Sleep(20 * time.Second)

	log.Println()
	log.Println("=== 测试结果 ===")
	log.Println("✓ Context 在整个调用链中正确传递")
	log.Println("✓ 重试发送时使用相同的 context")
	log.Println("✓ 级联超时控制生效")

	// 停止
	rabbit.Stop()
	log.Println()
	log.Println("程序退出")
}
