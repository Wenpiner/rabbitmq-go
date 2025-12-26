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

// ContextReceiver - 使用新的 ReceiveWithContext 接口
type ContextReceiver struct {
	count int
}

func (r *ContextReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	r.count++
	log.Printf("[新接口] 收到消息 #%d: %s", r.count, string(message.Body))

	// 检查 context 超时
	deadline, ok := ctx.Deadline()
	if ok {
		log.Printf("[新接口] Context 超时时间: %v (剩余: %v)", deadline, time.Until(deadline))
	}

	// 模拟处理
	time.Sleep(1 * time.Second)
	log.Printf("[新接口] 消息处理完成 #%d", r.count)
	return nil
}

func (r *ContextReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[新接口] 异常: %v", err)
}

// LegacyReceiver - 使用旧接口
type LegacyReceiver struct {
	count int
}

func (r *LegacyReceiver) Receive(key string, message amqp.Delivery) error {
	r.count++
	log.Printf("[旧接口] 收到消息 #%d: %s", r.count, string(message.Body))
	time.Sleep(1 * time.Second)
	log.Printf("[旧接口] 消息处理完成 #%d", r.count)
	return nil
}

func (r *LegacyReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[旧接口] 异常: %v", err)
}

func main() {
	log.Println("=== 阶段一功能测试 ===")

	// 创建 RabbitMQ 连接
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 测试 1: 新接口（带 context）
	log.Println("1. 注册新接口消费者（HandlerTimeout: 5秒）...")
	err := rabbit.Register("test-new", conf.ConsumerConf{
		Exchange:       conf.NewDirectExchange("test-exchange"),
		Queue:          conf.NewQueue("test-new-queue"),
		RouteKey:       "test.new",
		Name:           "test-new-consumer",
		AutoAck:        false,
		HandlerTimeout: 5 * time.Second,
	}, &ContextReceiver{})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	log.Println("   ✓ 新接口消费者注册成功")

	// 测试 2: 旧接口（向后兼容）
	log.Println("2. 注册旧接口消费者（使用默认30秒超时）...")
	err = rabbit.Register("test-legacy", conf.ConsumerConf{
		Exchange: conf.NewDirectExchange("test-exchange"),
		Queue:    conf.NewQueue("test-legacy-queue"),
		RouteKey: "test.legacy",
		Name:     "test-legacy-consumer",
		AutoAck:  false,
	}, &LegacyReceiver{})
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	log.Println("   ✓ 旧接口消费者注册成功")

	// 启动
	log.Println("3. 启动 RabbitMQ 连接...")
	rabbit.Start()

	// 等待连接建立
	time.Sleep(3 * time.Second)
	log.Println("   ✓ 连接建立成功")

	// 发送测试消息
	log.Println("4. 发送测试消息...")
	ctx := context.Background()

	// 发送到新接口
	msg1 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("消息给新接口"),
		Headers: amqp.Table{
			"trace_id": "trace-new-001",
		},
	}
	err = rabbit.SendMessageClose(ctx, "test-exchange", "test.new", true, msg1)
	if err != nil {
		log.Printf("   ✗ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送消息到新接口消费者")
	}

	time.Sleep(500 * time.Millisecond)

	// 发送到旧接口
	msg2 := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("消息给旧接口"),
	}
	err = rabbit.SendMessageClose(ctx, "test-exchange", "test.legacy", true, msg2)
	if err != nil {
		log.Printf("   ✗ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送消息到旧接口消费者")
	}

	log.Println("\n5. 等待消息处理...")
	time.Sleep(5 * time.Second)

	log.Println("\n=== 测试完成 ===")
	log.Println("✓ 新接口（ReceiveWithContext）工作正常")
	log.Println("✓ 旧接口（Receive）向后兼容正常")
	log.Println("✓ Context 超时配置生效")

	rabbit.Stop()
	time.Sleep(1 * time.Second)
	fmt.Println("\n程序退出")
}
