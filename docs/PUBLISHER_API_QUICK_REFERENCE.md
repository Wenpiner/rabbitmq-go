# Publisher API 快速参考

## 基础发送

```go
// 简单发送
err := rabbit.Publish(ctx, "exchange", "route", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Hello"),
})

// 强制路由
err := rabbit.PublishMandatory(ctx, "exchange", "route", msg)
```

## 批量发送

```go
messages := []amqp.Publishing{
    {Body: []byte("Message 1")},
    {Body: []byte("Message 2")},
    {Body: []byte("Message 3")},
}

// 普通批量发送（高性能）
err := rabbit.PublishBatch(ctx, "exchange", "route", messages)

// 带确认的批量发送（可靠）
err := rabbit.PublishBatchWithConfirm(ctx, "exchange", "route", messages)

// 事务批量发送（原子性）
err := rabbit.PublishBatchTx(ctx, "exchange", "route", messages)
```

## Confirm 模式

```go
err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送多条消息
    for i := 0; i < 10; i++ {
        msg := amqp.Publishing{Body: []byte(fmt.Sprintf("Message %d", i))}
        if err := ch.PublishWithContext(ctx, "exchange", "route", false, false, msg); err != nil {
            return err
        }
    }
    return nil // 自动等待所有确认
})
```

## 事务模式

```go
err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送多条消息
    for i := 0; i < 3; i++ {
        msg := amqp.Publishing{Body: []byte(fmt.Sprintf("Tx Message %d", i))}
        if err := ch.PublishWithContext(ctx, "exchange", "route", false, false, msg); err != nil {
            return err
        }
    }
    return nil // 成功返回自动提交，错误返回自动回滚
})
```

## BatchPublisher

```go
// 创建批量发送器
publisher := rabbit.NewBatchPublisher("exchange", "route").
    SetBatchSize(100).      // 批次大小
    SetAutoFlush(true)      // 自动刷新

defer publisher.Close(ctx)  // 确保最后刷新

// 添加消息
for i := 0; i < 1000; i++ {
    msg := amqp.Publishing{Body: []byte(fmt.Sprintf("Message %d", i))}
    publisher.Add(ctx, msg) // 每 100 条自动刷新
}

// 手动刷新
publisher.Flush(ctx)              // 普通刷新
publisher.FlushWithConfirm(ctx)   // Confirm 刷新
publisher.FlushTx(ctx)            // 事务刷新

// 查询状态
count := publisher.Len()          // 待发送消息数量
```

## 模式对比

| 方法 | 性能 | 可靠性 | 原子性 | 适用场景 |
|------|------|--------|--------|---------|
| Publish | ⭐⭐⭐⭐⭐ | ⭐ | ❌ | 实时消息 |
| PublishBatch | ⭐⭐⭐⭐⭐ | ⭐ | ❌ | 日志收集 |
| PublishBatchWithConfirm | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ | 重要消息 |
| PublishBatchTx | ⭐ | ⭐⭐⭐⭐⭐ | ✅ | 订单系统 |

## 错误处理

```go
// 基础错误处理
err := rabbit.PublishBatch(ctx, exchange, route, messages)
if err != nil {
    log.Printf("Failed to publish: %v", err)
    // 重试或记录失败
}

// 事务错误处理
err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送消息
    if err := ch.PublishWithContext(ctx, exchange, route, false, false, msg); err != nil {
        return err // 自动回滚
    }
    
    // 业务逻辑错误
    if someCondition {
        return fmt.Errorf("business error") // 自动回滚
    }
    
    return nil // 自动提交
})
```

## 超时控制

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := rabbit.PublishBatch(ctx, exchange, route, messages)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Publish timeout")
    }
}
```

## 最佳实践

### 1. 日志收集系统

```go
logPublisher := rabbit.NewBatchPublisher("logs", "app.logs").
    SetBatchSize(100).
    SetAutoFlush(true)

defer logPublisher.Close(ctx)

for logEntry := range logChannel {
    logPublisher.Add(ctx, logEntry)
}
```

### 2. 订单系统（原子性）

```go
orderMessages := []amqp.Publishing{
    {Body: []byte(`{"type":"order_created"}`)},
    {Body: []byte(`{"type":"inventory_deducted"}`)},
    {Body: []byte(`{"type":"payment_processed"}`)},
}

err := rabbit.PublishBatchTx(ctx, "orders", "order.events", orderMessages)
```

### 3. 重要消息（可靠性）

```go
err := rabbit.PublishBatchWithConfirm(ctx, "notifications", "email", messages)
if err != nil {
    log.Printf("Failed to send notifications: %v", err)
    // 重试逻辑
}
```

## 性能建议

### 批次大小选择

| 场景 | 推荐批次大小 |
|------|------------|
| 日志收集 | 100-500 |
| 监控数据 | 50-100 |
| 数据导入 | 1000-5000 |
| 实时消息 | 10-50 |

### 模式选择

- **高吞吐量**: 使用 `PublishBatch`
- **可靠发送**: 使用 `PublishBatchWithConfirm`
- **原子操作**: 使用 `PublishBatchTx`
- **实时消息**: 使用 `Publish`

## 常见问题

### Q: 什么时候使用事务模式？
A: 仅在需要原子性保证时使用（如订单系统），因为性能开销大。

### Q: BatchPublisher 是否线程安全？
A: 不是，需要在单个 goroutine 中使用。

### Q: 如何确保消息不丢失？
A: 使用 `PublishBatchWithConfirm` 或 `PublishBatchTx`。

### Q: 批量发送失败后如何重试？
A: 捕获错误后重新调用发送方法，消息列表不会被修改。

## 示例代码

完整示例请参考：
- [examples/16-publisher-transaction/](../examples/16-publisher-transaction/) - 事务模式
- [examples/17-batch-publisher/](../examples/17-batch-publisher/) - BatchPublisher

## 相关文档

- [PUBLISHER_API_SUMMARY.md](../PUBLISHER_API_SUMMARY.md) - 实现总结
- [README.md](../README.md) - 主文档

