# Publisher 追踪示例

本示例演示如何使用 Publisher API 的追踪功能，实现分布式追踪。

## 功能特性

- ✅ 自动生成追踪信息
- ✅ 使用已有的追踪信息
- ✅ 批量发送带追踪
- ✅ 模拟分布式追踪链路
- ✅ 消费并验证追踪信息

## 运行示例

```bash
# 确保 RabbitMQ 正在运行
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# 运行示例
go run main.go
```

## 示例说明

### 示例 1: 自动生成追踪信息

```go
err := rabbit.PublishWithTrace(ctx, "", "trace-queue", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Message with auto-generated trace"),
})
```

- 自动生成 trace ID 和 span ID
- 追踪信息自动注入到消息 headers

### 示例 2: 使用已有的追踪信息

```go
traceInfo := tracing.TraceInfo{
    TraceID: "my-trace-id-12345",
    SpanID:  "my-span-id-67890",
}
ctxWithTrace := tracing.InjectToContext(ctx, traceInfo)

err = rabbit.PublishWithTrace(ctxWithTrace, "", "trace-queue", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Message with existing trace"),
})
```

- 使用已有的 trace ID
- 自动生成新的 span ID
- 保持追踪链路的连续性

### 示例 3: 批量发送带追踪

```go
messages := []amqp.Publishing{
    {ContentType: "text/plain", Body: []byte("Batch message 1")},
    {ContentType: "text/plain", Body: []byte("Batch message 2")},
    {ContentType: "text/plain", Body: []byte("Batch message 3")},
}

err = rabbit.PublishBatchWithTrace(ctx, "", "trace-queue", messages)
```

- 所有消息共享同一个 trace ID
- 每条消息有独立的 span ID
- 适合批量操作的追踪

### 示例 4: 模拟分布式追踪链路

```go
// 服务 A 创建追踪
serviceATraceInfo := tracing.TraceInfo{
    TraceID: "distributed-trace-001",
    SpanID:  "service-a-span",
}
serviceACtx := tracing.InjectToContext(ctx, serviceATraceInfo)
err = rabbit.PublishWithTrace(serviceACtx, "", "trace-queue", msg)

// 服务 B 继续追踪链路
serviceBTraceInfo := tracing.TraceInfo{
    TraceID:      serviceATraceInfo.TraceID,      // 保持相同的 trace ID
    SpanID:       "service-b-span",
    ParentSpanID: serviceATraceInfo.SpanID,       // 设置父 span
}
serviceBCtx := tracing.InjectToContext(ctx, serviceBTraceInfo)
err = rabbit.PublishWithTrace(serviceBCtx, "", "trace-queue", msg)
```

- 跨服务保持相同的 trace ID
- 通过 parent span ID 建立调用关系
- 完整的分布式追踪链路

### 示例 5: 消费并验证追踪信息

```go
err = rabbit.RegisterConsumer("trace-queue", func(msg amqp.Delivery) error {
    // 从消息 headers 中提取追踪信息
    traceInfo := tracing.ExtractFromHeaders(msg.Headers)
    
    fmt.Printf("Trace ID: %s\n", traceInfo.TraceID)
    fmt.Printf("Span ID: %s\n", traceInfo.SpanID)
    fmt.Printf("Parent Span ID: %s\n", traceInfo.ParentSpanID)
    
    return nil
})
```

- 从消息 headers 中提取追踪信息
- 验证追踪信息的正确性
- 可以继续传递追踪信息

## 追踪信息格式

追踪信息存储在消息的 Headers 中：

```go
Headers: {
    "trace_id":       "uuid-string",
    "span_id":        "uuid-string",
    "parent_span_id": "uuid-string",  // 可选
}
```

## 最佳实践

1. **使用统一的 trace ID**: 在整个请求链路中保持相同的 trace ID
2. **正确设置 parent span**: 建立清晰的调用关系
3. **记录追踪日志**: 使用 `tracing.FormatTraceLog()` 格式化日志
4. **传递追踪上下文**: 通过 context 传递追踪信息

## 与其他追踪系统集成

可以将 RabbitMQ 的追踪信息与其他追踪系统集成：

- Jaeger
- Zipkin
- OpenTelemetry
- AWS X-Ray

只需要在发送消息前将追踪信息注入到 context 中即可。

