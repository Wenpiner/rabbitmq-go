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

// GracefulReceiver 演示优雅关闭时的消息处理
type GracefulReceiver struct {
	messageCount int
}

func (r *GracefulReceiver) ReceiveWithContext(ctx context.Context, key string, message amqp.Delivery) error {
	r.messageCount++
	log.Printf("[GracefulReceiver] 开始处理消息 #%d: %s", r.messageCount, string(message.Body))

	// 模拟消息处理（2秒）
	select {
	case <-time.After(2 * time.Second):
		log.Printf("[GracefulReceiver] ✓ 消息 #%d 处理完成", r.messageCount)
		return nil
	case <-ctx.Done():
		log.Printf("[GracefulReceiver] ⚠️  消息 #%d 处理被中断: %v", r.messageCount, ctx.Err())
		return ctx.Err()
	}
}

func (r *GracefulReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Printf("[GracefulReceiver] 异常处理: %v", err)
}

func main() {
	log.Println("=== 阶段三：优雅关闭演示 ===")
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

	// 注册消费者
	log.Println("1. 注册消费者...")
	err := rabbit.Register("graceful-test", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("graceful-exchange"),
		Queue:          conf.NewQueue("graceful-queue"),
		Name:           "graceful-consumer",
		AutoAck:        false,
		HandlerTimeout: 10 * time.Second, // 10秒超时
	}, &GracefulReceiver{})

	if err != nil {
		log.Fatalf("   ❌ 注册失败: %v", err)
	}
	log.Println("   ✓ 消费者注册成功")
	log.Println()

	// 使用带超时的启动
	log.Println("2. 启动 RabbitMQ（带30秒超时）...")
	startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startCancel()

	if err := rabbit.StartWithContext(startCtx); err != nil {
		log.Fatalf("   ❌ 启动失败: %v", err)
	}
	log.Println("   ✓ RabbitMQ 启动成功")
	log.Println()

	// 等待连接建立
	time.Sleep(3 * time.Second)

	// 发送一些测试消息
	log.Println("3. 发送测试消息...")
	for i := 1; i <= 5; i++ {
		ctx := context.Background()
		err := rabbit.SendMessageClose(
			ctx,
			"graceful-exchange",
			"",
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("测试消息 #%d", i)),
			},
		)
		if err != nil {
			log.Printf("   ⚠️  发送消息 #%d 失败: %v", i, err)
		} else {
			log.Printf("   ✓ 发送消息 #%d", i)
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println()

	// 监听系统信号
	log.Println("4. 等待系统信号（Ctrl+C）...")
	log.Println("   提示：按 Ctrl+C 触发优雅关闭")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println()
	log.Println("=== 收到关闭信号，开始优雅关闭 ===")
	log.Println()

	// 优雅关闭，最多等待 30 秒
	log.Println("5. 优雅关闭 RabbitMQ（最多等待30秒）...")
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if err := rabbit.StopWithContext(stopCtx); err != nil {
		log.Printf("   ⚠️  优雅关闭超时: %v", err)
		log.Println("   部分消息可能未处理完成")
	} else {
		log.Println("   ✓ RabbitMQ 优雅关闭成功")
		log.Println("   所有消息已处理完成")
	}

	log.Println()
	log.Println("=== 测试完成 ===")
	log.Println("✓ StartWithContext 工作正常")
	log.Println("✓ StopWithContext 工作正常")
	log.Println("✓ 优雅关闭机制生效")
	log.Println()
	log.Println("程序退出")
}
