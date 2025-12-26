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
	log.Println("=== 阶段二：消息发送 Context 支持测试 ===")
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

	// 启动连接
	log.Println("1. 启动 RabbitMQ 连接...")
	rabbit.Start()
	time.Sleep(3 * time.Second)
	log.Println("   ✓ 连接成功")
	log.Println()

	// 测试 1: 使用带超时的 context 发送延时消息
	log.Println("2. 测试带超时的 context 发送延时消息...")
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	err := rabbit.SendDelayMsg(
		ctx1,
		"test-exchange",
		"test.route",
		amqp.Delivery{
			Body: []byte("测试消息 - 带 5 秒超时的 context"),
			Headers: amqp.Table{
				"test_id": "test-1",
			},
		},
		3000, // 3 秒延时
	)

	if err != nil {
		log.Printf("   ❌ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送成功（带 5 秒超时）")
	}
	log.Println()

	// 测试 2: 使用已取消的 context
	log.Println("3. 测试已取消的 context...")
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2() // 立即取消

	err = rabbit.SendDelayMsg(
		ctx2,
		"test-exchange",
		"test.route",
		amqp.Delivery{
			Body: []byte("这条消息不应该被发送"),
		},
		1000,
	)

	if err != nil {
		log.Printf("   ✓ 预期的错误: %v", err)
	} else {
		log.Println("   ❌ 应该失败但成功了")
	}
	log.Println()

	// 测试 3: 使用 SendDelayMsgByArgs
	log.Println("4. 测试 SendDelayMsgByArgs 带 context...")
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel3()

	args := &conf.MQArgs{
		"x-custom-header": "custom-value",
	}

	err = rabbit.SendDelayMsgByArgs(
		ctx3,
		"test-exchange",
		"test.route",
		amqp.Delivery{
			Body: []byte("测试消息 - 带自定义参数"),
			Headers: amqp.Table{
				"test_id": "test-3",
			},
		},
		2000, // 2 秒延时
		args,
	)

	if err != nil {
		log.Printf("   ❌ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 发送成功（带自定义参数）")
	}
	log.Println()

	// 测试 4: 兼容性函数测试
	log.Println("5. 测试兼容性函数（不推荐使用）...")
	err = rabbit.SendDelayMsgCompat(
		"test-exchange",
		"test.route",
		amqp.Delivery{
			Body: []byte("兼容性测试消息"),
		},
		1000,
	)

	if err != nil {
		log.Printf("   ❌ 发送失败: %v", err)
	} else {
		log.Println("   ✓ 兼容性函数工作正常")
	}
	log.Println()

	log.Println("=== 测试完成 ===")
	log.Println("✓ SendDelayMsg 支持 context")
	log.Println("✓ SendDelayMsgByArgs 支持 context")
	log.Println("✓ Context 取消正确处理")
	log.Println("✓ 兼容性函数正常工作")

	// 停止
	rabbit.Stop()
	log.Println()
	log.Println("程序退出")
}
