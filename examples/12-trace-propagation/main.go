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

var rabbit *rabbitmq.RabbitMQ

// ServiceAReceiver 服务 A 的消息接收器
type ServiceAReceiver struct{}

func (r *ServiceAReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Println(tracing.FormatTraceLog(ctx, "【服务 A】收到消息"))
	log.Printf("【服务 A】消息内容: %s", string(message.Body))

	// 模拟业务处理
	time.Sleep(50 * time.Millisecond)

	// 调用服务 B（通过发送消息）
	log.Println(tracing.FormatTraceLog(ctx, "【服务 A】调用服务 B"))

	_, err := rabbit.SendMessageWithTrace(ctx, "service-b-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("来自服务 A 的消息"),
	})

	if err != nil {
		log.Printf("【服务 A】调用服务 B 失败: %v", err)
		return err
	}

	log.Println(tracing.FormatTraceLog(ctx, "【服务 A】处理完成"))
	return nil
}

func (r *ServiceAReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Println(tracing.FormatTraceLog(ctx, fmt.Sprintf("【服务 A】异常: %v", err)))
}

// ServiceBReceiver 服务 B 的消息接收器
type ServiceBReceiver struct{}

func (r *ServiceBReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Println(tracing.FormatTraceLog(ctx, "【服务 B】收到消息"))
	log.Printf("【服务 B】消息内容: %s", string(message.Body))

	// 提取追踪信息
	traceInfo := tracing.ExtractFromContext(ctx)
	log.Printf("【服务 B】追踪链路 - Trace ID: %s, Span ID: %s, Parent Span ID: %s",
		traceInfo.TraceID, traceInfo.SpanID, traceInfo.ParentSpanID)

	// 模拟业务处理
	time.Sleep(50 * time.Millisecond)

	// 调用服务 C（通过发送消息）
	log.Println(tracing.FormatTraceLog(ctx, "【服务 B】调用服务 C"))

	_, err := rabbit.SendMessageWithTrace(ctx, "service-c-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("来自服务 B 的消息"),
	})

	if err != nil {
		log.Printf("【服务 B】调用服务 C 失败: %v", err)
		return err
	}

	log.Println(tracing.FormatTraceLog(ctx, "【服务 B】处理完成"))
	return nil
}

func (r *ServiceBReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Println(tracing.FormatTraceLog(ctx, fmt.Sprintf("【服务 B】异常: %v", err)))
}

// ServiceCReceiver 服务 C 的消息接收器
type ServiceCReceiver struct{}

func (r *ServiceCReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
	log.Println(tracing.FormatTraceLog(ctx, "【服务 C】收到消息"))
	log.Printf("【服务 C】消息内容: %s", string(message.Body))

	// 提取追踪信息
	traceInfo := tracing.ExtractFromContext(ctx)
	log.Printf("【服务 C】追踪链路 - Trace ID: %s, Span ID: %s, Parent Span ID: %s",
		traceInfo.TraceID, traceInfo.SpanID, traceInfo.ParentSpanID)

	// 模拟业务处理
	time.Sleep(50 * time.Millisecond)

	log.Println(tracing.FormatTraceLog(ctx, "【服务 C】处理完成"))
	log.Println()
	log.Println("========================================")
	log.Println("  完整调用链追踪完成！")
	log.Println("  客户端 -> 服务 A -> 服务 B -> 服务 C")
	log.Printf("  Trace ID: %s", traceInfo.TraceID)
	log.Println("========================================")
	log.Println()

	return nil
}

func (r *ServiceCReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
	log.Println(tracing.FormatTraceLog(ctx, fmt.Sprintf("【服务 C】异常: %v", err)))
}

func main() {
	log.Println("========================================")
	log.Println("  追踪链路传播示例")
	log.Println("  演示: 客户端 -> 服务 A -> 服务 B -> 服务 C")
	log.Println("========================================")
	log.Println()

	// 创建 RabbitMQ 实例
	rabbit = rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 注册服务 A 消费者
	err := rabbit.Register("service-a", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("service-a-exchange"),
		Queue:    conf.NewQueue("service-a-queue"),
		Name:     "service-a-consumer",
		AutoAck:  false,
	}, &ServiceAReceiver{})
	if err != nil {
		log.Fatalf("注册服务 A 失败: %v", err)
	}

	// 注册服务 B 消费者
	err = rabbit.Register("service-b", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("service-b-exchange"),
		Queue:    conf.NewQueue("service-b-queue"),
		Name:     "service-b-consumer",
		AutoAck:  false,
	}, &ServiceBReceiver{})
	if err != nil {
		log.Fatalf("注册服务 B 失败: %v", err)
	}

	// 注册服务 C 消费者
	err = rabbit.Register("service-c", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("service-c-exchange"),
		Queue:    conf.NewQueue("service-c-queue"),
		Name:     "service-c-consumer",
		AutoAck:  false,
	}, &ServiceCReceiver{})
	if err != nil {
		log.Fatalf("注册服务 C 失败: %v", err)
	}

	// 启动 RabbitMQ
	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = rabbit.StartWithContext(startCtx)
	if err != nil {
		log.Fatalf("启动 RabbitMQ 失败: %v", err)
	}

	log.Println("✅ RabbitMQ 已启动")
	log.Println("✅ 服务 A、B、C 已注册")
	log.Println()

	// 等待一秒确保消费者准备好
	time.Sleep(1 * time.Second)

	// 发送初始消息到服务 A
	log.Println("📤 客户端发送消息到服务 A，开始追踪链路...")
	log.Println()

	// 生成追踪信息
	traceInfo := tracing.TraceInfo{
		TraceID: tracing.GenerateTraceID(),
		SpanID:  tracing.GenerateSpanID(),
	}
	ctx := tracing.InjectToContext(context.Background(), traceInfo)

	log.Printf("【客户端】生成 Trace ID: %s", traceInfo.TraceID)
	log.Printf("【客户端】生成 Span ID: %s", traceInfo.SpanID)
	log.Println()

	_, err = rabbit.SendMessageWithTrace(ctx, "service-a-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("来自客户端的初始消息"),
	})

	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		log.Println("✅ 消息已发送到服务 A")
		log.Println()
	}

	log.Println("========================================")
	log.Println("  观察追踪链路传播...")
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
