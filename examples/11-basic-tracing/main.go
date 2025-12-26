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
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// TracingReceiver 实现带追踪的消息接收器
type TracingReceiver struct{}

// Receive 实现 ReceiveWithContext 接口
func (r *TracingReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	// 从 context 中获取 trace ID
	traceID := tracing.GetTraceID(ctx)
	spanID := tracing.GetSpanID(ctx)
	parentSpanID := tracing.GetParentSpanID(ctx)

	// 在日志中输出追踪信息
	log.Println(tracing.FormatTraceLog(ctx, "收到消息"))
	log.Printf("追踪信息 - Trace ID: %s, Span ID: %s, Parent Span ID: %s", traceID, spanID, parentSpanID)
	log.Printf("消息内容: %s", string(message.Body))

	// 模拟业务处理
	time.Sleep(100 * time.Millisecond)

	log.Println(tracing.FormatTraceLog(ctx, "消息处理完成"))
	return nil
}

// Exception 实现异常处理
func (r *TracingReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Println(tracing.FormatTraceLog(ctx, fmt.Sprintf("消息处理异常: %v", err)))
}

func main() {
	log.Println("========================================")
	log.Println("  基础追踪示例")
	log.Println("========================================")
	log.Println()

	// 创建 RabbitMQ 实例
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 注册消费者
	err := rabbit.Register("tracing-example", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("tracing-exchange"),
		Queue:    conf.NewQueue("tracing-queue"),
		Name:     "tracing-consumer",
		AutoAck:  false,
	}, &TracingReceiver{})

	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	// 启动 RabbitMQ
	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = rabbit.StartWithContext(startCtx)
	if err != nil {
		log.Fatalf("启动 RabbitMQ 失败: %v", err)
	}

	log.Println("✅ RabbitMQ 已启动")
	log.Println()

	// 等待一秒确保消费者准备好
	time.Sleep(1 * time.Second)

	// 发送带追踪信息的消息
	log.Println("📤 发送带追踪信息的消息...")
	log.Println()

	// 示例 1: 自动生成追踪信息
	log.Println("示例 1: 自动生成追踪信息")
	ctx1 := context.Background()
	_, err = rabbit.SendMessageWithTrace(ctx1, "tracing-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("消息 1: 自动生成追踪信息"),
	})
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Println("✅ 消息 1 已发送")
	}
	log.Println()

	// 示例 2: 手动指定追踪信息
	log.Println("示例 2: 手动指定追踪信息")
	traceInfo := tracing.TraceInfo{
		TraceID: tracing.GenerateTraceID(),
		SpanID:  tracing.GenerateSpanID(),
	}
	ctx2 := tracing.InjectToContext(context.Background(), traceInfo)
	log.Printf("生成的 Trace ID: %s", traceInfo.TraceID)
	log.Printf("生成的 Span ID: %s", traceInfo.SpanID)

	_, err = rabbit.SendMessageWithTrace(ctx2, "tracing-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("消息 2: 手动指定追踪信息"),
	})
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Println("✅ 消息 2 已发送")
	}
	log.Println()

	// 示例 3: 模拟调用链
	log.Println("示例 3: 模拟调用链（父子 span）")
	parentTraceInfo := tracing.TraceInfo{
		TraceID: tracing.GenerateTraceID(),
		SpanID:  tracing.GenerateSpanID(),
	}
	ctx3 := tracing.InjectToContext(context.Background(), parentTraceInfo)
	log.Printf("父 Trace ID: %s", parentTraceInfo.TraceID)
	log.Printf("父 Span ID: %s", parentTraceInfo.SpanID)

	_, err = rabbit.SendMessageWithTrace(ctx3, "tracing-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("消息 3: 模拟调用链"),
	})
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Println("✅ 消息 3 已发送（会生成新的 Span ID）")
	}
	log.Println()

	log.Println("========================================")
	log.Println("  等待消息处理...")
	log.Println("  按 Ctrl+C 优雅退出")
	log.Println("========================================")
	log.Println()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println()
	log.Println("收到退出信号，开始优雅关闭...")

	// 优雅关闭
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	err = rabbit.StopWithContext(stopCtx)
	if err != nil {
		log.Printf("关闭失败: %v", err)
	} else {
		log.Println("✅ 优雅关闭完成")
	}
}
