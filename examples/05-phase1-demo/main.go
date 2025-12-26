package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

// ContextReceiver - 使用新的 ReceiveWithContext 接口
type ContextReceiver struct{}

func (r *ContextReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[ContextReceiver] 开始处理消息: %s", string(message.Body))

	// 检查 trace_id
	if traceID := ctx.Value("trace_id"); traceID != nil {
		log.Printf("[ContextReceiver] Trace ID: %v", traceID)
	}

	// 模拟耗时操作，支持 context 取消
	select {
	case <-time.After(2 * time.Second):
		log.Printf("[ContextReceiver] 消息处理成功: %s", string(message.Body))
		return nil
	case <-ctx.Done():
		// Context 被取消或超时
		log.Printf("[ContextReceiver] 处理被取消: %v", ctx.Err())
		return fmt.Errorf("processing cancelled: %w", ctx.Err())
	}
}

func (r *ContextReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[ContextReceiver] 异常处理: %v, 消息: %s", err, string(message.Body))
}

// TimeoutReceiver - 测试超时场景
type TimeoutReceiver struct{}

func (r *TimeoutReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[TimeoutReceiver] 开始处理消息: %s (将超时)", string(message.Body))

	// 模拟长时间处理（超过超时时间）
	select {
	case <-time.After(10 * time.Second):
		log.Printf("[TimeoutReceiver] 处理完成")
		return nil
	case <-ctx.Done():
		log.Printf("[TimeoutReceiver] 检测到超时: %v", ctx.Err())
		return ctx.Err()
	}
}

func (r *TimeoutReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[TimeoutReceiver] 异常处理 - 超时错误: %v", err)
}

// LegacyReceiver - 使用旧的 Receive 接口（测试向后兼容性）
type LegacyReceiver struct{}

func (r *LegacyReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[LegacyReceiver] 处理消息（旧接口）: %s", string(message.Body))
	time.Sleep(1 * time.Second)
	log.Printf("[LegacyReceiver] 消息处理成功")
	return nil
}

func (r *LegacyReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[LegacyReceiver] 异常处理（旧接口）: %v", err)
}

func main() {
	log.Println("=== 阶段一测试：Handler 层和 Receive/Exception 接口改造 ===")

	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 测试 1: 新接口 - 正常处理（5秒超时）
	err := rabbit.Register("context-normal", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("phase1-test"),
		Queue:          conf.NewQueue("context-normal-queue"),
		RouteKey:       "",
		Name:           "context-normal-consumer",
		AutoAck:        false,
		HandlerTimeout: 5 * time.Second,
		Retry:          conf.NewRetryConf(),
	}, &ContextReceiver{})
	if err != nil {
		log.Fatalf("注册 context-normal 失败: %v", err)
	}
	log.Println("✓ 注册 context-normal 消费者成功（5秒超时）")

	// 测试 2: 新接口 - 超时场景（3秒超时）
	err = rabbit.Register("context-timeout", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("phase1-test"),
		Queue:          conf.NewQueue("context-timeout-queue"),
		RouteKey:       "",
		Name:           "context-timeout-consumer",
		AutoAck:        false,
		HandlerTimeout: 3 * time.Second,
		Retry:          conf.NewRetryConf(),
	}, &TimeoutReceiver{})
	if err != nil {
		log.Fatalf("注册 context-timeout 失败: %v", err)
	}
	log.Println("✓ 注册 context-timeout 消费者成功（3秒超时）")

	// 测试 3: 旧接口 - 向后兼容性测试
	err = rabbit.Register("legacy", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("phase1-test"),
		Queue:    conf.NewQueue("legacy-queue"),
		RouteKey: "",
		Name:     "legacy-consumer",
		AutoAck:  false,
		Retry:    conf.NewRetryConf(),
	}, &LegacyReceiver{})
	if err != nil {
		log.Fatalf("注册 legacy 失败: %v", err)
	}
	log.Println("✓ 注册 legacy 消费者成功（旧接口，使用默认30秒超时）")

	rabbit.Start()
	log.Println("✓ RabbitMQ 连接成功，开始监听消息...")

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 发送测试消息
	log.Println("\n=== 发送测试消息 ===")
	sendTestMessages(rabbit)

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n按 Ctrl+C 停止测试...")
	<-sigChan

	log.Println("\n停止 RabbitMQ...")
	rabbit.Stop()
	time.Sleep(1 * time.Second)
	log.Println("测试完成")
}

func sendTestMessages(rabbit *rabbitmq.RabbitMQ) {
	ctx := context.Background()

	// 消息 1: 正常处理
	msg1 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("测试消息 1 - 正常处理"),
		Headers: amqp.Table{
			"trace_id": "trace-001",
		},
	}
	err := rabbit.SendMessageClose(ctx, "phase1-test", "", true, msg1)
	if err != nil {
		log.Printf("发送消息 1 失败: %v", err)
	} else {
		log.Println("✓ 发送消息 1: 正常处理")
	}

	time.Sleep(500 * time.Millisecond)

	// 消息 2: 超时测试
	msg2 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("测试消息 2 - 超时测试"),
		Headers: amqp.Table{
			"trace_id": "trace-002",
		},
	}
	err = rabbit.SendMessageClose(ctx, "phase1-test", "", true, msg2)
	if err != nil {
		log.Printf("发送消息 2 失败: %v", err)
	} else {
		log.Println("✓ 发送消息 2: 超时测试")
	}

	log.Println("\n=== 观察消息处理情况 ===")
}
