# Publisher Confirm 模式示例

本示例演示如何使用 Publisher API 的 Confirm 模式，实现可靠的消息发送。

## 功能特性

- ✅ 手动处理消息确认
- ✅ 批量发送并等待所有确认
- ✅ 处理超时场景
- ✅ 跟踪每条消息的确认状态
- ✅ 高性能批量确认

## 运行示例

```bash
# 确保 RabbitMQ 正在运行
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# 运行示例
go run main.go
```

## 示例说明

### 示例 1: 使用 WithPublisherConfirm 手动处理确认

```go
err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 创建确认通道
    confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
    
    // 发送消息
    msg := amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("Message with manual confirmation"),
    }
    if err := ch.PublishWithContext(ctx, "", "confirm-queue", false, false, msg); err != nil {
        return err
    }
    
    // 等待确认
    select {
    case confirm := <-confirms:
        if confirm.Ack {
            fmt.Println("✓ 消息已确认")
        } else {
            return fmt.Errorf("message not confirmed")
        }
    case <-time.After(5 * time.Second):
        return fmt.Errorf("confirmation timeout")
    }
    
    return nil
})
```

- 完全控制确认流程
- 可以自定义超时时间
- 适合需要精细控制的场景

### 示例 2: 批量发送并等待所有确认

```go
messages := []amqp.Publishing{
    {ContentType: "text/plain", Body: []byte("Confirmed message 1")},
    {ContentType: "text/plain", Body: []byte("Confirmed message 2")},
    {ContentType: "text/plain", Body: []byte("Confirmed message 3")},
}

err = rabbit.PublishBatchWithConfirm(ctx, "", "confirm-queue", messages)
```

- 自动等待所有消息的确认
- 如果任何消息未确认，返回错误
- 简单易用，适合大多数场景

### 示例 3: 批量发送大量消息

```go
largeMessages := make([]amqp.Publishing, 100)
for i := 0; i < 100; i++ {
    largeMessages[i] = amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte(fmt.Sprintf("Large batch message %d", i+1)),
    }
}

start := time.Now()
err = rabbit.PublishBatchWithConfirm(ctx, "", "confirm-queue", largeMessages)
elapsed := time.Since(start)
```

- 高性能批量发送
- 自动管理确认通道
- 适合大批量消息发送

### 示例 4: 处理超时场景

```go
timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
defer cancel()

err = rabbit.PublishBatchWithConfirm(timeoutCtx, "", "confirm-queue", messages)
if err != nil {
    fmt.Printf("预期的超时错误: %v\n", err)
}
```

- 使用 context 控制超时
- 优雅处理超时错误
- 避免无限等待

### 示例 5: 高级用法 - 跟踪每条消息的确认

```go
err = rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(messages)))
    
    // 发送所有消息
    for i, msg := range messages {
        if err := ch.PublishWithContext(ctx, "", "confirm-queue", false, false, msg); err != nil {
            return err
        }
        fmt.Printf("发送消息 %d/%d\n", i+1, len(messages))
    }
    
    // 等待所有确认
    for i := 0; i < len(messages); i++ {
        select {
        case confirm := <-confirms:
            if confirm.Ack {
                fmt.Printf("确认消息 %d/%d (DeliveryTag: %d)\n", 
                    i+1, len(messages), confirm.DeliveryTag)
            } else {
                return fmt.Errorf("message %d not confirmed", i+1)
            }
        case <-time.After(5 * time.Second):
            return fmt.Errorf("confirmation timeout at message %d", i+1)
        }
    }
    
    return nil
})
```

- 跟踪每条消息的确认状态
- 显示 DeliveryTag
- 适合需要详细日志的场景

## Confirm 模式原理

1. **启用 Confirm 模式**: 调用 `ch.Confirm(false)` 启用
2. **发送消息**: 使用 `ch.PublishWithContext()` 发送
3. **接收确认**: 通过 `ch.NotifyPublish()` 接收确认
4. **检查 Ack**: 确认消息的 `Ack` 字段为 `true` 表示成功

## 性能对比

| 方法 | 性能 | 可靠性 | 适用场景 |
|------|------|--------|----------|
| Publish | 最快 | 低 | 非关键消息 |
| PublishBatchWithConfirm | 快 | 高 | 批量关键消息 |
| WithPublisherConfirm | 中 | 高 | 需要精细控制 |

## 最佳实践

1. **批量发送**: 使用 `PublishBatchWithConfirm` 提高性能
2. **设置超时**: 使用 context 控制超时时间
3. **错误处理**: 正确处理确认失败的情况
4. **缓冲区大小**: 确认通道的缓冲区大小应 >= 消息数量
5. **重试机制**: 对于确认失败的消息，实现重试逻辑

## 注意事项

1. **性能开销**: Confirm 模式会增加一定的性能开销
2. **内存使用**: 大批量发送时注意确认通道的缓冲区大小
3. **超时设置**: 根据网络情况合理设置超时时间
4. **错误处理**: 确认失败时需要决定是否重试

## 与事务模式对比

| 特性 | Confirm 模式 | 事务模式 |
|------|-------------|----------|
| 性能 | 高 | 低 |
| 吞吐量 | 高 | 低 |
| 复杂度 | 中 | 低 |
| 推荐使用 | ✅ | ❌ |

**推荐使用 Confirm 模式**，性能更好，吞吐量更高。

