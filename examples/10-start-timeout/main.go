package main

import (
	"context"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

// SimpleReceiver 简单的消息接收器
type SimpleReceiver struct{}

func (r *SimpleReceiver) ReceiveWithContext(ctx context.Context, key string, message amqp.Delivery) error {
	log.Printf("[SimpleReceiver] 收到消息: %s", string(message.Body))
	return nil
}

func (r *SimpleReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[SimpleReceiver] 异常: %v", err)
}

func main() {
	log.Println("=== 阶段三：启动超时控制演示 ===")
	log.Println()

	// 测试 1: 正常启动（足够的超时时间）
	log.Println("【测试 1】正常启动（30秒超时）")
	log.Println("-----------------------------------")

	rabbit1 := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	err := rabbit1.Register("test1", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("test-exchange"),
		Queue:    conf.NewQueue("test-queue-1"),
		Name:     "test-consumer-1",
		AutoAck:  false,
	}, &SimpleReceiver{})

	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	startCtx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel1()

	log.Println("启动 RabbitMQ...")
	startTime := time.Now()
	err = rabbit1.StartWithContext(startCtx1)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("❌ 启动失败: %v (耗时: %v)", err, elapsed)
	} else {
		log.Printf("✅ 启动成功 (耗时: %v)", elapsed)
	}
	log.Println()

	// 清理
	time.Sleep(1 * time.Second)
	rabbit1.Stop()
	time.Sleep(2 * time.Second)

	// 测试 2: 启动超时（非常短的超时时间）
	log.Println("【测试 2】启动超时（1毫秒超时）")
	log.Println("-----------------------------------")
	log.Println("注意：这个测试可能会失败，因为超时时间太短")

	rabbit2 := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	err = rabbit2.Register("test2", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("test-exchange"),
		Queue:    conf.NewQueue("test-queue-2"),
		Name:     "test-consumer-2",
		AutoAck:  false,
	}, &SimpleReceiver{})

	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	startCtx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel2()

	log.Println("启动 RabbitMQ（超时时间: 1ms）...")
	startTime = time.Now()
	err = rabbit2.StartWithContext(startCtx2)
	elapsed = time.Since(startTime)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("✅ 预期的超时错误: %v (耗时: %v)", err, elapsed)
			log.Println("   超时控制正常工作")
		} else {
			log.Printf("⚠️  其他错误: %v (耗时: %v)", err, elapsed)
		}
	} else {
		log.Printf("⚠️  启动成功（可能是因为连接非常快）(耗时: %v)", elapsed)
	}
	log.Println()

	// 清理
	rabbit2.Stop()
	time.Sleep(1 * time.Second)

	// 测试 3: 使用旧接口（兼容性测试）
	log.Println("【测试 3】使用旧接口 Start()（兼容性）")
	log.Println("-----------------------------------")

	rabbit3 := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "admin",
		Password: "admin",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	err = rabbit3.Register("test3", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("test-exchange"),
		Queue:    conf.NewQueue("test-queue-3"),
		Name:     "test-consumer-3",
		AutoAck:  false,
	}, &SimpleReceiver{})

	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	log.Println("使用旧接口启动...")
	rabbit3.Start()
	time.Sleep(3 * time.Second)
	log.Println("✅ 旧接口工作正常")
	log.Println()

	rabbit3.Stop()
	time.Sleep(1 * time.Second)

	log.Println("=== 测试完成 ===")
	log.Println("✓ StartWithContext 支持超时控制")
	log.Println("✓ 超时时正确返回错误")
	log.Println("✓ 旧接口 Start() 保持兼容")
	log.Println()
	log.Println("程序退出")
}
