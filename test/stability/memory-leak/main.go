package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
	"github.com/wenpiner/rabbitmq-go/v2/test/stability/common"
)

// 内存泄漏检测测试
// 目标: 通过反复创建和销毁消费者，检测是否存在内存泄漏和 goroutine 泄漏

func main() {
	log.Println("=== 内存泄漏检测测试 ===")

	cfg := common.LoadConfig()
	cycleCount := getEnvInt("CYCLE_COUNT", 1000)
	pprofEnabled := os.Getenv("PPROF_ENABLED") == "true"

	log.Printf("配置: 测试时长=%v, 循环次数=%d, pprof=%v\n",
		cfg.TestDuration, cycleCount, pprofEnabled)

	metrics := common.NewMetrics()

	// 启动指标服务
	go func() {
		log.Printf("指标服务启动在 %s\n", cfg.MetricsAddr)
		if err := metrics.ServeMetrics(cfg.MetricsAddr); err != nil {
			log.Printf("指标服务错误: %v\n", err)
		}
	}()

	// 启动 pprof
	if pprofEnabled {
		go func() {
			log.Println("pprof 服务启动在 :6060")
			log.Println("访问 http://localhost:6060/debug/pprof/")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				log.Printf("pprof 服务错误: %v\n", err)
			}
		}()
	}

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

	exchangeName := "stability-memory-leak-exchange"
	queuePrefix := "stability-memory-leak-queue"
	routingKey := "stability.memory"

	stopChan := make(chan struct{})
	cycle := 0

	// 记录初始内存状态
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	initialGoroutines := runtime.NumGoroutine()

	log.Printf("初始状态: Goroutines=%d, Memory=%d MB\n",
		initialGoroutines, initialMem.Alloc/1024/1024)

	// 循环创建和销毁消费者
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				cycle++
				if cycle > cycleCount {
					continue
				}

				consumerName := fmt.Sprintf("consumer-%d", cycle)
				queueName := fmt.Sprintf("%s-%d", queuePrefix, cycle)

				// 创建消费者
				handler := rabbitmq.NewFuncHandler(func(ctx context.Context, msg *rabbitmq.Message) error {
					metrics.RecordReceived()
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
				)
				if err != nil {
					log.Printf("注册消费者失败: %v", err)
					continue
				}

				// 发送一些消息
				for i := 0; i < 10; i++ {
					err := client.Publish(ctx, exchangeName, routingKey, amqp.Publishing{
						ContentType: "text/plain",
						Body:        []byte(fmt.Sprintf("msg-%d-%d", cycle, i)),
					})
					if err == nil {
						metrics.RecordSent()
					}
				}

				// 等待处理
				time.Sleep(1 * time.Second)

				// 注销消费者
				if err := client.UnregisterConsumer(consumerName); err != nil {
					log.Printf("注销消费者失败: %v", err)
				}

				// 强制 GC
				if cycle%10 == 0 {
					runtime.GC()
					
					var mem runtime.MemStats
					runtime.ReadMemStats(&mem)
					currentGoroutines := runtime.NumGoroutine()
					
					log.Printf("循环 %d: Goroutines=%d (增长=%d), Memory=%d MB (增长=%d MB)\n",
						cycle,
						currentGoroutines,
						currentGoroutines-initialGoroutines,
						mem.Alloc/1024/1024,
						int64(mem.Alloc/1024/1024)-int64(initialMem.Alloc/1024/1024))
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

	log.Printf("🚀 内存泄漏测试开始，将运行 %v\n", cfg.TestDuration)

	select {
	case <-testTimer.C:
		log.Println("⏰ 测试时间到")
	case sig := <-sigChan:
		log.Printf("🛑 收到信号: %v\n", sig)
	}

	close(stopChan)
	time.Sleep(3 * time.Second)

	// 最终内存检查
	runtime.GC()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)
	finalGoroutines := runtime.NumGoroutine()

	log.Println("\n=== 内存泄漏分析 ===")
	log.Printf("初始: Goroutines=%d, Memory=%d MB\n",
		initialGoroutines, initialMem.Alloc/1024/1024)
	log.Printf("最终: Goroutines=%d, Memory=%d MB\n",
		finalGoroutines, finalMem.Alloc/1024/1024)
	log.Printf("增长: Goroutines=%d, Memory=%d MB\n",
		finalGoroutines-initialGoroutines,
		int64(finalMem.Alloc/1024/1024)-int64(initialMem.Alloc/1024/1024))
	log.Printf("完成循环: %d/%d\n", cycle, cycleCount)

	metrics.PrintStats()

	log.Println("✅ 测试完成")
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return defaultValue
}

