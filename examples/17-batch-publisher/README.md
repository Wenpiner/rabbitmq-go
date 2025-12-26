# BatchPublisher 批量发送器示例

本示例演示如何使用 BatchPublisher 批量发送辅助器，优化批量发送的使用体验。

## 功能特性

- ✅ 批量消息收集
- ✅ 自动刷新
- ✅ 手动刷新
- ✅ 多种刷新模式（普通/Confirm/事务）
- ✅ 链式配置

## 运行示例

```bash
# 确保 RabbitMQ 正在运行
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# 运行示例
go run main.go
```

## 示例说明

### 示例 1: 基本使用

```go
publisher := rabbit.NewBatchPublisher("", "batch-queue")

// 添加消息
for i := 1; i <= 5; i++ {
    msg := amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte(fmt.Sprintf("Message %d", i)),
    }
    publisher.Add(ctx, msg)
}

// 手动刷新
publisher.Flush(ctx)
```

- 创建批量发送器
- 添加消息到队列
- 手动刷新发送

### 示例 2: 自动刷新

```go
autoPublisher := rabbit.NewBatchPublisher("", "auto-batch-queue").
    SetBatchSize(3).
    SetAutoFlush(true)

// 添加消息，每 3 条自动刷新
for i := 1; i <= 10; i++ {
    msg := amqp.Publishing{...}
    autoPublisher.Add(ctx, msg)
}

// 刷新剩余消息
autoPublisher.Close(ctx)
```

- 设置批次大小为 3
- 启用自动刷新
- 达到批次大小时自动发送

### 示例 3: 日志收集系统

```go
logPublisher := rabbit.NewBatchPublisher("logs-exchange", "app.logs").
    SetBatchSize(100).
    SetAutoFlush(true)

defer logPublisher.Close(ctx)

// 模拟日志收集
for i := 1; i <= 250; i++ {
    logEntry := amqp.Publishing{
        ContentType: "application/json",
        Body:        []byte(fmt.Sprintf(`{"level":"info","message":"Log entry %d"}`, i)),
    }
    logPublisher.Add(ctx, logEntry)
}
```

- 批量收集日志
- 每 100 条自动发送
- 使用 defer 确保最后刷新

### 示例 4: 使用不同的刷新模式

```go
mixedPublisher := rabbit.NewBatchPublisher("", "mixed-queue")

// 添加消息
for i := 1; i <= 5; i++ {
    mixedPublisher.Add(ctx, msg)
}

// 使用普通刷新
mixedPublisher.Flush(ctx)

// 使用带确认的刷新
mixedPublisher.FlushWithConfirm(ctx)

// 使用事务刷新
mixedPublisher.FlushTx(ctx)
```

- 支持三种刷新模式
- 根据需求选择合适的模式

## API 参考

### 创建批量发送器

```go
publisher := rabbit.NewBatchPublisher(exchange, routingKey)
```

### 配置方法

```go
// 设置批次大小（默认 100）
publisher.SetBatchSize(50)

// 设置自动刷新（默认 false）
publisher.SetAutoFlush(true)

// 链式调用
publisher.SetBatchSize(50).SetAutoFlush(true)
```

### 添加消息

```go
err := publisher.Add(ctx, msg)
```

- 添加消息到批次
- 如果启用自动刷新且达到批次大小，会自动刷新

### 刷新方法

```go
// 普通刷新
err := publisher.Flush(ctx)

// 带确认的刷新
err := publisher.FlushWithConfirm(ctx)

// 事务刷新
err := publisher.FlushTx(ctx)

// 关闭并刷新（等同于 Flush）
err := publisher.Close(ctx)
```

### 查询方法

```go
// 获取当前待发送的消息数量
count := publisher.Len()
```

## 使用场景

### 1. 日志收集系统

```go
logPublisher := rabbit.NewBatchPublisher("logs", "app.logs").
    SetBatchSize(100).
    SetAutoFlush(true)

defer logPublisher.Close(ctx)

// 持续收集日志
for logEntry := range logChannel {
    logPublisher.Add(ctx, logEntry)
}
```

### 2. 监控数据上报

```go
metricsPublisher := rabbit.NewBatchPublisher("metrics", "app.metrics").
    SetBatchSize(50).
    SetAutoFlush(true)

// 定期上报指标
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
    metric := collectMetrics()
    metricsPublisher.Add(ctx, metric)
}
```

### 3. 批量数据导入

```go
importPublisher := rabbit.NewBatchPublisher("import", "data.import").
    SetBatchSize(1000)

// 读取文件并批量导入
for record := range records {
    importPublisher.Add(ctx, record)
}

// 最后刷新剩余数据
importPublisher.Close(ctx)
```

## 性能优化

### 批次大小选择

| 场景 | 推荐批次大小 | 说明 |
|------|------------|------|
| 日志收集 | 100-500 | 平衡延迟和吞吐量 |
| 监控数据 | 50-100 | 较低延迟 |
| 数据导入 | 1000-5000 | 高吞吐量 |
| 实时消息 | 10-50 | 低延迟优先 |

### 自动刷新 vs 手动刷新

| 特性 | 自动刷新 | 手动刷新 |
|------|---------|---------|
| 使用简单 | ✅ | ❌ |
| 控制精确 | ❌ | ✅ |
| 适合场景 | 持续流式数据 | 批量处理 |

## 最佳实践

1. **使用 defer Close**: 确保最后刷新剩余消息
   ```go
   publisher := rabbit.NewBatchPublisher(exchange, route)
   defer publisher.Close(ctx)
   ```

2. **合理设置批次大小**: 根据消息大小和延迟要求调整
   ```go
   publisher.SetBatchSize(100) // 根据实际情况调整
   ```

3. **错误处理**: 正确处理 Add 和 Flush 的错误
   ```go
   if err := publisher.Add(ctx, msg); err != nil {
       log.Printf("Failed to add message: %v", err)
   }
   ```

4. **监控队列长度**: 定期检查待发送消息数量
   ```go
   if publisher.Len() > 1000 {
       log.Warn("Too many pending messages")
   }
   ```

5. **选择合适的刷新模式**:
   - 普通刷新: 高性能，无保证
   - Confirm 刷新: 可靠性高
   - 事务刷新: 原子性保证

## 注意事项

1. **内存使用**: 批次大小过大会占用更多内存
2. **延迟**: 批次大小越大，延迟越高
3. **错误处理**: Flush 失败时消息不会清空
4. **并发安全**: BatchPublisher 不是并发安全的

## 与直接调用的对比

### 使用 BatchPublisher

```go
publisher := rabbit.NewBatchPublisher(exchange, route).
    SetBatchSize(100).
    SetAutoFlush(true)

for msg := range messages {
    publisher.Add(ctx, msg)
}
```

**优点**:
- ✅ 代码简洁
- ✅ 自动管理批次
- ✅ 链式配置

### 直接调用 PublishBatch

```go
batch := make([]amqp.Publishing, 0, 100)

for msg := range messages {
    batch = append(batch, msg)
    if len(batch) >= 100 {
        rabbit.PublishBatch(ctx, exchange, route, batch)
        batch = batch[:0]
    }
}

// 刷新剩余消息
if len(batch) > 0 {
    rabbit.PublishBatch(ctx, exchange, route, batch)
}
```

**缺点**:
- ❌ 代码冗长
- ❌ 手动管理批次
- ❌ 容易遗漏最后的刷新

## 总结

- ✅ 简化批量发送的使用
- ✅ 支持自动和手动刷新
- ✅ 多种刷新模式
- ✅ 链式配置
- ⚠️ 注意内存和延迟

