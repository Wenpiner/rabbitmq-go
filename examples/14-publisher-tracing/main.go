package main

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

func main() {
	// 创建 RabbitMQ 实例
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Host:     "127.0.0.1",
		Port:     5672,
		Username: "guest",
		Password: "guest",
		VHost:    "/",
	})
	defer rabbit.Stop()

	ctx := context.Background()

	fmt.Println("=== 示例 1: 自动生成追踪信息 ===")
	err := rabbit.PublishWithTrace(ctx, "", "trace-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Message with auto-generated trace"),
	})
	if err != nil {
		log.Fatalf("Failed to publish with trace: %v", err)
	}
	fmt.Println("✓ 消息已发送（自动生成追踪信息）")

	fmt.Println("\n=== 示例 2: 使用已有的追踪信息 ===")
	// 创建追踪信息
	traceInfo := tracing.TraceInfo{
		TraceID: "my-trace-id-12345",
		SpanID:  "my-span-id-67890",
	}
	ctxWithTrace := tracing.InjectToContext(ctx, traceInfo)

	err = rabbit.PublishWithTrace(ctxWithTrace, "", "trace-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Message with existing trace"),
	})
	if err != nil {
		log.Fatalf("Failed to publish with existing trace: %v", err)
	}
	fmt.Println("✓ 消息已发送（使用已有追踪信息）")

	fmt.Println("\n=== 示例 3: 批量发送带追踪 ===")
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Batch message 1")},
		{ContentType: "text/plain", Body: []byte("Batch message 2")},
		{ContentType: "text/plain", Body: []byte("Batch message 3")},
	}

	err = rabbit.PublishBatchWithTrace(ctx, "", "trace-queue", messages)
	if err != nil {
		log.Fatalf("Failed to publish batch with trace: %v", err)
	}
	fmt.Println("✓ 批量消息已发送（共享追踪 ID）")

	fmt.Println("\n=== 示例 4: 模拟分布式追踪链路 ===")
	// 模拟服务 A 创建追踪
	serviceATraceInfo := tracing.TraceInfo{
		TraceID: "distributed-trace-001",
		SpanID:  "service-a-span",
	}
	serviceACtx := tracing.InjectToContext(ctx, serviceATraceInfo)

	// 服务 A 发送消息
	err = rabbit.PublishWithTrace(serviceACtx, "", "trace-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Message from Service A"),
	})
	if err != nil {
		log.Fatalf("Failed to publish from service A: %v", err)
	}
	fmt.Println("✓ 服务 A 发送消息")

	// 模拟服务 B 接收并继续追踪链路
	serviceBTraceInfo := tracing.TraceInfo{
		TraceID:      serviceATraceInfo.TraceID, // 保持相同的 trace ID
		SpanID:       "service-b-span",
		ParentSpanID: serviceATraceInfo.SpanID, // 设置父 span
	}
	serviceBCtx := tracing.InjectToContext(ctx, serviceBTraceInfo)

	// 服务 B 发送消息
	err = rabbit.PublishWithTrace(serviceBCtx, "", "trace-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Message from Service B"),
	})
	if err != nil {
		log.Fatalf("Failed to publish from service B: %v", err)
	}
	fmt.Println("✓ 服务 B 发送消息（继承追踪链路）")

	fmt.Println("\n=== 示例 5: 验证追踪信息的传递 ===")
	// 发送一条带追踪的消息
	finalTraceInfo := tracing.TraceInfo{
		TraceID: "final-trace-id",
		SpanID:  "final-span-id",
	}
	finalCtx := tracing.InjectToContext(ctx, finalTraceInfo)

	testMsg := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test message for trace verification"),
	}

	err = rabbit.PublishWithTrace(finalCtx, "", "trace-queue", testMsg)
	if err != nil {
		log.Fatalf("Failed to publish test message: %v", err)
	}

	fmt.Println("✓ 追踪信息已注入到消息 headers")
	fmt.Println("  消费者可以通过 tracing.ExtractFromHeaders(msg.Headers) 提取追踪信息")
	fmt.Println("  示例代码:")
	fmt.Println("    traceInfo := tracing.ExtractFromHeaders(msg.Headers)")
	fmt.Printf("    fmt.Printf(\"Trace ID: %%s\\n\", traceInfo.TraceID)\n")
	fmt.Printf("    fmt.Printf(\"Span ID: %%s\\n\", traceInfo.SpanID)\n")

	fmt.Println("\n✓ 所有示例完成！")
}
