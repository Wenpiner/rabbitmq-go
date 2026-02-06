package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
	"github.com/wenpiner/rabbitmq-go/v2/test/stability/common"
)

// 高并发压力测试
// 目标: 测试系统在高并发场景下的性能和稳定性

func main() {
	log.Println("=== 高并发压力测试 ===")

	cfg := common.LoadConfig()
	log.Printf("配置: 测试时长=%v, 消息速率=%d/s, 消费者数量=%d, 批量大小=%d\n",
		cfg.TestDuration, cfg.MessageRate, cfg.ConsumerCount, cfg.BatchSize)

	metrics := common.NewMetrics()

	// 启动指标服务
	go func() {
		log.Printf("指标服务启动在 %s\n", cfg.MetricsAddr)
		if err := metrics.ServeMetrics(cfg.MetricsAddr); err != nil {
			log.Printf("指标服务错误: %v\n", err)
		}
	}()

	// 创建客户端
	client := rabbitmq.New(
		rabbitmq.WithConfig(cfg.RabbitMQ),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelWarn)),
		rabbitmq.WithAutoReconnect(true),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	log.Println("✅ 已连接到 RabbitMQ")

	// 注册大量消费者
	exchangeName := "stability-high-concurrency-exchange"
	queuePrefix := "stability-high-concurrency-queue"
	routingKey := "stability.high-concurrency.#"

	for i := 0; i < cfg.ConsumerCount; i++ {
		consumerName := fmt.Sprintf("consumer-%d", i)
		queueName := fmt.Sprintf("%s-%d", queuePrefix, i%10) // 10个队列

		handler := rabbitmq.NewFuncHandler(func(ctx context.Context, msg *rabbitmq.Message) error {
			metrics.RecordReceived()
			// 快速处理
			return nil
		})

		err := client.RegisterConsumer(consumerName,
			rabbitmq.WithQueue(conf.QueueConf{
				Name:       queueName,
				Durable:    false,
				AutoDelete: true,
			}),
			rabbitmq.WithExchange(conf.ExchangeConf{
				ExchangeName: exchangeName,
				Type:         "topic",
				Durable:      false,
				AutoDelete:   true,
			}),
			rabbitmq.WithRouteKey(routingKey),
			rabbitmq.WithAutoAck(true),
			rabbitmq.WithHandler(handler),
			rabbitmq.WithConcurrency(10), // 高并发
			rabbitmq.WithQos(conf.QosConf{
				Enable:       true,
				PrefetchCount: 100,
			}),
		)
		if err != nil {
			log.Fatalf("注册消费者失败: %v", err)
		}
	}

	log.Printf("✅ 已注册 %d 个消费者\n", cfg.ConsumerCount)

	// 启动多个发送协程
	stopChan := make(chan struct{})
	var wg sync.WaitGroup

	publisherCount := 10
	for p := 0; p < publisherCount; p++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ticker := time.NewTicker(time.Second / time.Duration(cfg.MessageRate/publisherCount))
			defer ticker.Stop()

			for {
				select {
				case <-stopChan:
					return
				case <-ticker.C:
					// 批量发送
					messages := make([]amqp.Publishing, cfg.BatchSize)
					for i := 0; i < cfg.BatchSize; i++ {
						messages[i] = amqp.Publishing{
							ContentType: "text/plain",
							Body:        []byte(fmt.Sprintf("msg-%d-%d", id, i)),
						}
					}

					routingKey := fmt.Sprintf("stability.high-concurrency.%d", id)
					err := client.PublishBatch(ctx, exchangeName, routingKey, messages)
					if err != nil {
						metrics.RecordFailed()
						metrics.RecordError(err)
					} else {
						for range messages {
							metrics.RecordSent()
						}
					}
				}
			}
		}(p)
	}

	// 定期打印统计
	go func() {
		ticker := time.NewTicker(10 * time.Second)
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

	log.Printf("🚀 高并发测试开始，将运行 %v\n", cfg.TestDuration)

	select {
	case <-testTimer.C:
		log.Println("⏰ 测试时间到")
	case sig := <-sigChan:
		log.Printf("🛑 收到信号: %v\n", sig)
	}

	close(stopChan)
	wg.Wait()

	time.Sleep(3 * time.Second)

	log.Println("\n=== 最终统计 ===")
	metrics.PrintStats()

	log.Println("✅ 测试完成")
}

