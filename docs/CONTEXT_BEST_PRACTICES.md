# Context 最佳实践

## 📋 概述

本文档提供在 RabbitMQ-Go 项目中使用 `context.Context` 的最佳实践和常见陷阱。

---

## ✅ 最佳实践

### 1. 始终传递 Context

**推荐做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 将 context 传递给所有下游调用
    return r.processMessage(ctx, message)
}

func (r *MyReceiver) processMessage(ctx context.Context, message amqp.Delivery) error {
    // 使用 context 进行数据库调用
    result, err := r.db.QueryContext(ctx, "SELECT ...")
    if err != nil {
        return err
    }
    
    // 使用 context 进行 HTTP 调用
    req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
    resp, err := r.client.Do(req)
    
    return nil
}
```

**避免做法**：
```go
// ❌ 不要忽略 context
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 错误：没有传递 context
    result, err := r.db.Query("SELECT ...")
    return err
}
```

---

### 2. 检查 Context 取消

**推荐做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 在长时间操作前检查 context
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // 执行操作
    for i := 0; i < 100; i++ {
        // 定期检查 context
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        // 处理逻辑
        processItem(i)
    }
    
    return nil
}
```

---

### 3. 合理设置超时时间

**推荐做法**：
```go
// 根据业务需求设置合理的超时时间
err := rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange:       conf.NewFanoutExchange("my-exchange"),
    Queue:          conf.NewQueue("my-queue"),
    Name:           "my-consumer",
    AutoAck:        false,
    HandlerTimeout: 30 * time.Second, // 根据实际处理时间设置
}, &MyReceiver{})
```

**超时时间建议**：
- **快速处理**（< 1 秒）：5-10 秒
- **普通处理**（1-5 秒）：15-30 秒
- **复杂处理**（5-30 秒）：60-120 秒
- **长时间处理**（> 30 秒）：考虑异步处理或拆分任务

---

### 4. 不要存储 Context

**推荐做法**：
```go
type MyReceiver struct {
    db *sql.DB
    // ✅ 不要在结构体中存储 context
}

func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ✅ context 作为参数传递
    return r.process(ctx, message)
}
```

**避免做法**：
```go
type MyReceiver struct {
    ctx context.Context // ❌ 不要存储 context
    db  *sql.DB
}
```

---

### 5. 使用 context.WithValue 传递请求范围的数据

**推荐做法**：
```go
// 定义自定义类型作为 key
type contextKey string

const (
    userIDKey contextKey = "user_id"
    requestIDKey contextKey = "request_id"
)

func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 从消息中提取用户 ID
    userID := extractUserID(message)
    
    // 将用户 ID 存入 context
    ctx = context.WithValue(ctx, userIDKey, userID)
    
    // 下游可以获取用户 ID
    return r.processWithUser(ctx, message)
}

func (r *MyReceiver) processWithUser(ctx context.Context, message amqp.Delivery) error {
    // 获取用户 ID
    userID, ok := ctx.Value(userIDKey).(string)
    if !ok {
        return errors.New("user ID not found in context")
    }
    
    log.Printf("Processing message for user: %s", userID)
    return nil
}
```

**注意事项**：
- 使用自定义类型作为 key，避免冲突
- 只存储请求范围的数据，不要存储可选参数
- 不要滥用 WithValue，优先使用函数参数

---

### 6. 优雅处理超时错误

**推荐做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    err := r.processMessage(ctx, message)
    
    if err != nil {
        // 区分超时错误和业务错误
        if errors.Is(err, context.DeadlineExceeded) {
            log.Printf("Message processing timeout, will retry")
            return err // 返回错误，触发重试
        }
        
        if errors.Is(err, context.Canceled) {
            log.Printf("Message processing cancelled")
            return err
        }
        
        // 其他业务错误
        log.Printf("Business error: %v", err)
        return err
    }
    
    return nil
}
```

---

### 7. 在 Exception 中使用 Context

**推荐做法**：
```go
func (r *MyReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
    // 检查 context 是否仍然有效
    select {
    case <-ctx.Done():
        log.Printf("Exception handler cancelled, skipping")
        return
    default:
    }
    
    // 使用 context 进行告警发送
    alertCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    r.sendAlert(alertCtx, err, message)
    
    // 使用 context 写入死信队列
    r.writeToDLQ(ctx, message)
}
```

---

## ❌ 常见陷阱

### 1. Context 泄漏

**错误示例**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ❌ 创建了 context 但没有 cancel
    newCtx, _ := context.WithTimeout(ctx, 10*time.Second)
    return r.process(newCtx, message)
}
```

**正确做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ✅ 始终调用 cancel
    newCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    return r.process(newCtx, message)
}
```

---

### 2. 使用 context.Background() 替代传入的 Context

**错误示例**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ❌ 忽略了传入的 context
    newCtx := context.Background()
    return r.process(newCtx, message)
}
```

**正确做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ✅ 基于传入的 context 创建新的 context
    newCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    return r.process(newCtx, message)
}
```

---

### 3. 在 Goroutine 中使用父 Context

**错误示例**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ❌ 在 goroutine 中使用父 context
    go func() {
        // 如果父 context 被取消，这里会立即失败
        r.asyncProcess(ctx, message)
    }()
    return nil
}
```

**正确做法**：
```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // ✅ 为异步操作创建独立的 context
    asyncCtx := context.Background()
    
    // 或者从父 context 复制值，但使用新的取消控制
    asyncCtx = context.WithValue(asyncCtx, tracing.TraceIDKey, tracing.GetTraceID(ctx))
    
    go func() {
        r.asyncProcess(asyncCtx, message)
    }()
    return nil
}
```

---

## 📊 性能考虑

### 1. Context 创建开销

- 每次创建 context 有约 48 字节的内存分配
- WithValue 会创建新的 context 节点
- 建议：避免在循环中频繁创建 context

### 2. Context 检查开销

- `ctx.Done()` 的 select 检查非常快（纳秒级）
- 建议：在长循环中定期检查，不需要每次迭代都检查

**示例**：
```go
for i := 0; i < 1000000; i++ {
    // 每 1000 次迭代检查一次
    if i%1000 == 0 {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }
    
    // 处理逻辑
    process(i)
}
```

---

## 🔍 调试技巧

### 1. 记录 Context 超时信息

```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 记录 deadline
    if deadline, ok := ctx.Deadline(); ok {
        remaining := time.Until(deadline)
        log.Printf("Processing with %v remaining", remaining)
    }
    
    err := r.process(ctx, message)
    
    if errors.Is(err, context.DeadlineExceeded) {
        log.Printf("Processing timeout after deadline")
    }
    
    return err
}
```

### 2. 使用追踪信息

```go
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    traceID := tracing.GetTraceID(ctx)
    log.Printf("[%s] Start processing", traceID)
    
    err := r.process(ctx, message)
    
    if err != nil {
        log.Printf("[%s] Processing failed: %v", traceID, err)
    } else {
        log.Printf("[%s] Processing completed", traceID)
    }
    
    return err
}
```

---

## 📚 参考资料

- [Go Context 官方文档](https://pkg.go.dev/context)
- [Go Context 最佳实践](https://go.dev/blog/context)
- [Effective Go - Context](https://go.dev/doc/effective_go#context)

**最后更新**：2025-12-26
