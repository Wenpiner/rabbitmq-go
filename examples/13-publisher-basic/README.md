# Publisher Basic - 基础 Publisher API 示例

本示例演示如何使用新的 Publisher API 发送消息，包括单条发送、批量发送和性能对比。

## 功能特性

### 1. WithPublisher - 核心方法
自动管理 channel 生命周期，防止 channel 泄漏。

```go
err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // channel 会在函数返回后自动关闭
    return ch.PublishWithContext(ctx, "", "test-queue", false, false, msg)
})
```

### 2. Publish - 单条消息发送
最简单的发送方式，适合发送单条消息。

```go
err := rabbit.Publish(ctx, "", "test-queue", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Hello World"),
})
```

### 3. PublishBatch - 批量发送
高性能批量发送，所有消息复用同一个 channel。

```go
messages := []amqp.Publishing{
    {Body: []byte("Message 1")},
    {Body: []byte("Message 2")},
    {Body: []byte("Message 3")},
}
err := rabbit.PublishBatch(ctx, "", "test-queue", messages)
```

## 性能优势

### 旧方式 vs 新方式

**旧方式（每次创建新 channel）：**
```go
for i := 0; i < 100; i++ {
    rabbit.SendMessageClose(ctx, "", "queue", true, msg)
}
// 创建 100 次 channel，性能差
```

**新方式（复用 channel）：**
```go
rabbit.PublishBatch(ctx, "", "queue", messages)
// 只创建 1 次 channel，性能提升 10-100 倍
```

## 运行示例

### 前置条件
确保 RabbitMQ 服务正在运行：
```bash
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

### 运行
```bash
cd examples/13-publisher-basic
go run main.go
```

### 预期输出
```
=== 示例 1: WithPublisher ===
✓ Message sent successfully with WithPublisher

=== 示例 2: Publish ===
✓ Message sent successfully with Publish

=== 示例 3: Publish with Timeout ===
✓ Message sent successfully with timeout

=== 示例 4: PublishBatch ===
✓ 5 messages sent successfully with PublishBatch

=== 示例 5: Performance Comparison ===
Old way (SendMessageClose): 100 messages in 523ms
New way (PublishBatch): 100 messages in 45ms
✓ PublishBatch is 11.62x faster!

=== 示例 6: WithPublisher for Batch ===
✓ 10 messages sent successfully with WithPublisher

=== All examples completed ===
```

## 核心优势

### 1. 自动资源管理
- ✅ 自动创建和关闭 channel
- ✅ 防止 channel 泄漏
- ✅ 无需手动调用 `defer channel.Close()`

### 2. 性能优化
- ✅ 批量发送复用 channel
- ✅ 减少创建/销毁开销
- ✅ 性能提升 10-100 倍

### 3. 简洁的 API
- ✅ 统一的接口设计
- ✅ 支持 context 超时控制
- ✅ 清晰的错误处理

## 最佳实践

### 单条消息
使用 `Publish` 方法：
```go
err := rabbit.Publish(ctx, exchange, routingKey, msg)
```

### 批量消息
使用 `PublishBatch` 方法：
```go
err := rabbit.PublishBatch(ctx, exchange, routingKey, messages)
```

### 自定义逻辑
使用 `WithPublisher` 方法：
```go
err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 自定义发送逻辑
    return nil
})
```

## 迁移指南

### 从旧 API 迁移

**旧代码：**
```go
channel, err := rabbit.Channel()
if err != nil {
    return err
}
defer channel.Close()
err = channel.PublishWithContext(ctx, exchange, route, false, false, msg)
```

**新代码：**
```go
err := rabbit.Publish(ctx, exchange, route, msg)
```

## 相关文档

- [API 参考文档](../../docs/tmpFeature/05_API_REFERENCE.md)
- [迁移指南](../../docs/tmpFeature/06_MIGRATION_GUIDE.md)
- [性能优化指南](../../docs/tmpFeature/07_PERFORMANCE_GUIDE.md)

