# Publisher 事务示例

本示例演示如何使用 Publisher API 的事务功能，实现原子性消息发送。

## 功能特性

- ✅ 事务模式（WithPublisherTx）
- ✅ 自动提交和回滚
- ✅ 批量事务发送（PublishBatchTx）
- ✅ 原子性保证
- ✅ 订单系统场景演示

## 运行示例

```bash
# 确保 RabbitMQ 正在运行
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management

# 运行示例
go run main.go
```

## 示例说明

### 示例 1: 事务模式 - 成功提交

```go
err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送多条消息
    for i := 1; i <= 3; i++ {
        msg := amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(fmt.Sprintf("Transaction Message %d", i)),
        }
        if err := ch.PublishWithContext(ctx, "", "tx-queue", false, false, msg); err != nil {
            return err
        }
    }
    return nil // 自动提交
})
```

- 用户函数成功返回时自动提交事务
- 所有消息原子性发送

### 示例 2: 事务模式 - 自动回滚

```go
err = rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送第一条消息
    msg1 := amqp.Publishing{...}
    if err := ch.PublishWithContext(ctx, "", "tx-queue", false, false, msg1); err != nil {
        return err
    }
    
    // 模拟错误
    return fmt.Errorf("simulated error")
})
```

- 用户函数返回错误时自动回滚
- 所有消息都不会发送

### 示例 3: 批量事务发送

```go
messages := []amqp.Publishing{
    {ContentType: "text/plain", Body: []byte("Batch Tx Message 1")},
    {ContentType: "text/plain", Body: []byte("Batch Tx Message 2")},
    {ContentType: "text/plain", Body: []byte("Batch Tx Message 3")},
}

err = rabbit.PublishBatchTx(ctx, "", "tx-queue", messages)
```

- 简化的批量事务发送接口
- 自动管理事务生命周期

### 示例 4: 订单系统场景

```go
// 订单系统：需要同时发送订单创建和库存扣减消息
// 要求：要么全部成功，要么全部失败
orderMessages := []amqp.Publishing{
    {Body: []byte(`{"type":"order_created","order_id":"ORD-12345"}`)},
    {Body: []byte(`{"type":"inventory_deducted","product_id":"PROD-456"}`)},
    {Body: []byte(`{"type":"payment_processed","order_id":"ORD-12345"}`)},
}

err = rabbit.PublishBatchTx(ctx, "order-exchange", "order.events", orderMessages)
```

- 保证订单相关事件的原子性
- 避免数据不一致

### 示例 5: 事务 vs 非事务性能对比

```go
// 非事务批量发送
err = rabbit.PublishBatch(ctx, "", "perf-queue", testMessages)

// 事务批量发送
err = rabbit.PublishBatchTx(ctx, "", "perf-queue", testMessages)
```

- 对比两种模式的性能差异
- 根据需求选择合适的模式

## 事务原理

### 工作流程

1. **开启事务**: 调用 `ch.Tx()` 启用事务模式
2. **发送消息**: 在事务中发送消息
3. **提交/回滚**:
   - 成功: 调用 `ch.TxCommit()` 提交
   - 失败: 调用 `ch.TxRollback()` 回滚

### 原子性保证

- **全部成功**: 所有消息都发送到 broker
- **全部失败**: 所有消息都不发送

## 性能考虑

| 特性 | 事务模式 | 非事务模式 | Confirm 模式 |
|------|---------|-----------|-------------|
| 性能 | 低 | 高 | 中 |
| 原子性 | ✅ | ❌ | ❌ |
| 可靠性 | 高 | 低 | 高 |
| 适用场景 | 强一致性 | 高吞吐量 | 可靠发送 |

**性能影响**:
- 事务模式比非事务模式慢 **10-20 倍**
- 仅在需要原子性保证时使用

## 使用场景

### 适合使用事务

1. **订单系统**: 订单创建 + 库存扣减 + 支付处理
2. **金融系统**: 转账操作（扣款 + 入账）
3. **工作流**: 多步骤操作需要原子性

### 不适合使用事务

1. **日志收集**: 不需要原子性
2. **监控数据**: 允许部分失败
3. **高吞吐量场景**: 性能要求高

## 最佳实践

1. **仅在必要时使用**: 事务会降低性能
2. **批量操作**: 使用 `PublishBatchTx` 而不是循环调用
3. **错误处理**: 正确处理回滚错误
4. **超时控制**: 使用 context 控制超时
5. **监控**: 监控事务成功率和性能

## 注意事项

1. **性能开销**: 事务模式会显著降低性能
2. **不支持跨连接**: 事务仅在单个 channel 内有效
3. **与 Confirm 互斥**: 不能同时使用事务和 Confirm 模式
4. **回滚日志**: 回滚失败会记录日志但不会返回错误

## 错误处理

```go
err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // 发送消息
    if err := ch.PublishWithContext(ctx, exchange, route, false, false, msg); err != nil {
        return err // 自动回滚
    }
    
    // 业务逻辑错误
    if someCondition {
        return fmt.Errorf("business logic error") // 自动回滚
    }
    
    return nil // 自动提交
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
    // 处理失败情况
}
```

## 总结

- ✅ 提供原子性保证
- ✅ 自动提交和回滚
- ✅ 简单易用的 API
- ⚠️ 性能开销较大
- ⚠️ 仅在需要强一致性时使用

