package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
	"github.com/wenpiner/rabbitmq-go/v2/tracing"
)

// 示例 3: 分布式追踪
// 演示如何使用内置的追踪功能进行链路追踪

func main() {
	log.Println("=== RabbitMQ 分布式追踪示例 ===")

	// 创建客户端
	client := rabbitmq.New(
		rabbitmq.WithConfig(conf.RabbitConf{
			Scheme:   "amqp",
			Host:     "localhost",
			Port:     5672,
			Username: "guest",
			Password: "guest",
			VHost:    "/",
		}),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 创建带追踪的消息处理器
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			// 从消息中提取追踪信息
			traceInfo := msg.TraceInfo
			log.Printf("📨 收到消息:")
			log.Printf("   内容: %s", string(msg.Body()))
			log.Printf("   TraceID: %s", traceInfo.TraceID)
			log.Printf("   SpanID: %s", traceInfo.SpanID)
			log.Printf("   ParentSpanID: %s", traceInfo.ParentSpanID)

			// 模拟业务处理
			time.Sleep(100 * time.Millisecond)

			// 如果需要继续传播追踪信息到下游服务
			// 可以从 context 中提取追踪信息
			ctxTraceInfo := tracing.ExtractFromContext(ctx)
			log.Printf("   Context TraceID: %s", ctxTraceInfo.TraceID)

			return nil
		},
	)

	err := client.RegisterConsumer("trace-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "trace-queue",
			Durable: false,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "trace-exchange",
			Type:         "topic",
			Durable:      false,
		}),
		rabbitmq.WithRouteKey("trace.*"),
		rabbitmq.WithAutoAck(true),
		rabbitmq.WithHandler(handler),
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 发送带追踪的消息
	log.Println("\n📤 发送带追踪的消息...")

	// 方式 1: 使用 PublishWithTrace（自动生成追踪信息）
	for i := 1; i <= 3; i++ {
		err := client.PublishWithTrace(ctx, "trace-exchange", "trace.auto", amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Auto-traced message"),
		})
		if err != nil {
			log.Printf("❌ 发送失败: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 方式 2: 手动创建追踪上下文
	log.Println("\n📤 发送手动追踪的消息...")
	traceInfo := tracing.TraceInfo{
		TraceID: tracing.GenerateTraceID(),
		SpanID:  tracing.GenerateSpanID(),
	}
	traceCtx := tracing.InjectToContext(ctx, traceInfo)

	err = client.PublishWithTrace(traceCtx, "trace-exchange", "trace.manual", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Manual-traced message"),
	})
	if err != nil {
		log.Printf("❌ 发送失败: %v", err)
	}

	log.Printf("✅ 已发送追踪消息，TraceID: %s", traceInfo.TraceID)

	time.Sleep(2 * time.Second)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("\n🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan

	log.Println("🛑 正在优雅关闭...")
}

