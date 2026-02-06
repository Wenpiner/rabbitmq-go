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
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
	"github.com/wenpiner/rabbitmq-go/v2/test/stability/common"
)

// 长时间稳定性测试
// 目标: 验证系统在长时间运行下的稳定性，检测内存泄漏、连接稳定性等问题

func main() {
	log.Println("=== 长时间稳定性测试 ===")

	// 加载配置
	cfg := common.LoadConfig()
	log.Printf("配置: 测试时长=%v, 消息速率=%d/s, 消费者数量=%d\n",
		cfg.TestDuration, cfg.MessageRate, cfg.ConsumerCount)

	// 创建指标收集器
	metrics := common.NewMetrics()

	// 启动指标服务
	go func() {
		log.Printf("指标服务启动在 %s\n", cfg.MetricsAddr)
		if err := metrics.ServeMetrics(cfg.MetricsAddr); err != nil {
			log.Printf("指标服务错误: %v\n", err)
		}
	}()

	// 创建 RabbitMQ 客户端
	client := rabbitmq.New(
		rabbitmq.WithConfig(cfg.RabbitMQ),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
		rabbitmq.WithAutoReconnect(true),
		rabbitmq.WithReconnectInterval(5*time.Second),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Println("✅ 已连接到 RabbitMQ")

	// 注册消费者
	exchangeName := "stability-long-run-exchange"
	queueName := "stability-long-run-queue"
	routingKey := "stability.long-run"

	for i := 0; i < cfg.ConsumerCount; i++ {
		consumerName := fmt.Sprintf("consumer-%d", i)
		handler := rabbitmq.NewFuncHandler(func(ctx context.Context, msg *rabbitmq.Message) error {
			metrics.RecordReceived()
			// 模拟处理时间
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		err := client.RegisterConsumer(consumerName,
			rabbitmq.WithQueue(conf.QueueConf{
				Name:       queueName,
				Durable:    true,
				AutoDelete: false,
			}),
			rabbitmq.WithExchange(conf.ExchangeConf{
				ExchangeName: exchangeName,
				Type:         "topic",
				Durable:      true,
				AutoDelete:   false,
			}),
			rabbitmq.WithRouteKey(routingKey),
			rabbitmq.WithAutoAck(false),
			rabbitmq.WithHandler(handler),
			rabbitmq.WithConcurrency(5),
		)
		if err != nil {
			log.Fatalf("注册消费者失败: %v", err)
		}
	}

	log.Printf("✅ 已注册 %d 个消费者\n", cfg.ConsumerCount)

	// 启动发送协程
	stopChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(cfg.MessageRate))
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				err := client.Publish(ctx, exchangeName, routingKey, amqp.Publishing{
					ContentType:  "text/plain",
					Body:         []byte(fmt.Sprintf("message-%d", time.Now().Unix())),
					DeliveryMode: amqp.Persistent,
				})
				if err != nil {
					metrics.RecordFailed()
					metrics.RecordError(err)
				} else {
					metrics.RecordSent()
				}
			}
		}
	}()

	// 定期打印统计
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				metrics.PrintStats()
			}
		}
	}()

	// 等待测试时长或中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	testTimer := time.NewTimer(cfg.TestDuration)
	defer testTimer.Stop()

	log.Printf("🚀 测试开始，将运行 %v\n", cfg.TestDuration)

	select {
	case <-testTimer.C:
		log.Println("⏰ 测试时间到")
	case sig := <-sigChan:
		log.Printf("🛑 收到信号: %v\n", sig)
	}

	// 停止发送
	close(stopChan)

	// 等待消息处理完成
	time.Sleep(5 * time.Second)

	// 打印最终统计
	log.Println("\n=== 最终统计 ===")
	metrics.PrintStats()

	log.Println("✅ 测试完成")
}

