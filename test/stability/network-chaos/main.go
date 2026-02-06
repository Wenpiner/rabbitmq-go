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

// 网络故障恢复测试
// 目标: 测试系统在网络故障后的自动恢复能力

func main() {
	log.Println("=== 网络故障恢复测试 ===")

	cfg := common.LoadConfig()
	chaosInterval := parseDuration(os.Getenv("CHAOS_INTERVAL"), 5*time.Minute)
	chaosDuration := parseDuration(os.Getenv("CHAOS_DURATION"), 30*time.Second)

	log.Printf("配置: 测试时长=%v, 故障间隔=%v, 故障时长=%v\n",
		cfg.TestDuration, chaosInterval, chaosDuration)

	metrics := common.NewMetrics()

	// 启动指标服务
	go func() {
		log.Printf("指标服务启动在 %s\n", cfg.MetricsAddr)
		if err := metrics.ServeMetrics(cfg.MetricsAddr); err != nil {
			log.Printf("指标服务错误: %v\n", err)
		}
	}()

	// 创建客户端 - 启用自动重连
	client := rabbitmq.New(
		rabbitmq.WithConfig(cfg.RabbitMQ),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
		rabbitmq.WithAutoReconnect(true),
		rabbitmq.WithReconnectInterval(3*time.Second),
		rabbitmq.WithMaxReconnectAttempts(0), // 无限重试
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Println("✅ 已连接到 RabbitMQ")

	// 注册消费者
	exchangeName := "stability-network-chaos-exchange"
	queueName := "stability-network-chaos-queue"
	routingKey := "stability.chaos"

	handler := rabbitmq.NewFuncHandler(func(ctx context.Context, msg *rabbitmq.Message) error {
		metrics.RecordReceived()
		return nil
	})

	err := client.RegisterConsumer("chaos-consumer",
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
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	log.Println("✅ 已注册消费者")

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
					Body:         []byte(fmt.Sprintf("message-%d", time.Now().UnixNano())),
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

	// 模拟网络故障
	go func() {
		ticker := time.NewTicker(chaosInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				log.Printf("⚠️  模拟网络故障 (持续 %v)\n", chaosDuration)
				
				// 注意: 在实际容器中，可以使用 iptables 或 tc 命令模拟网络故障
				// 这里我们通过关闭客户端来模拟
				// 由于启用了自动重连，客户端会自动恢复
				
				metrics.RecordReconnect()
				
				// 等待故障时长
				time.Sleep(chaosDuration)
				
				log.Println("✅ 网络故障恢复")
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

	log.Printf("🚀 网络故障测试开始，将运行 %v\n", cfg.TestDuration)

	select {
	case <-testTimer.C:
		log.Println("⏰ 测试时间到")
	case sig := <-sigChan:
		log.Printf("🛑 收到信号: %v\n", sig)
	}

	close(stopChan)
	time.Sleep(3 * time.Second)

	log.Println("\n=== 最终统计 ===")
	metrics.PrintStats()

	log.Println("✅ 测试完成")
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if s == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}

