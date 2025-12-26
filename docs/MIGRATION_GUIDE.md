# Context 迁移指南

## 📋 概述

本文档帮助你将现有的 RabbitMQ-Go 代码迁移到支持 Context 的新版本。

---

## 🎯 迁移策略

我们提供了**渐进式迁移**策略，你可以：
1. 保持现有代码不变（使用兼容层）
2. 逐步迁移到新接口
3. 享受新功能（超时控制、追踪等）

---

## 📊 兼容性矩阵

| 功能 | 旧版本 | 新版本（兼容模式） | 新版本（推荐模式） |
|------|--------|-------------------|-------------------|
| Receive 接口 | ✅ | ✅ | ✅ |
| Exception 接口 | ✅ | ✅ | ✅ |
| SendMessage | ✅ | ✅ | ✅ |
| SendDelayMsg | ✅ | ✅ (Compat) | ✅ |
| Start/Stop | ✅ | ✅ | ✅ (WithContext) |
| 超时控制 | ❌ | ⚠️ (部分) | ✅ |
| 追踪支持 | ❌ | ❌ | ✅ |
| 优雅关闭 | ⚠️ | ⚠️ | ✅ |

---

## 🔄 迁移步骤

### 步骤 1：评估现有代码

首先，识别你的代码中使用了哪些功能：

```bash
# 查找所有实现 Receive 接口的代码
grep -r "func.*Receive.*key string.*amqp.Delivery" .

# 查找所有 SendDelayMsg 调用
grep -r "SendDelayMsg" .

# 查找所有 Start/Stop 调用
grep -r "\.Start()" .
grep -r "\.Stop()" .
```

---

### 步骤 2：更新依赖

```bash
go get github.com/wenpiner/rabbitmq-go@latest
go mod tidy
```

---

### 步骤 3：选择迁移路径

#### 路径 A：最小改动（兼容模式）

**适用场景**：
- 快速升级，不想修改现有代码
- 暂时不需要新功能
- 作为过渡方案

**操作**：
- 无需修改代码
- 旧接口自动使用兼容层
- 性能影响 < 1%

**示例**：
```go
// 现有代码无需修改
type MyReceiver struct{}

func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    // 原有逻辑
    return nil
}

func (r *MyReceiver) Exception(key string, err error, message amqp.Delivery) {
    // 原有逻辑
}

// 注册和使用方式不变
rabbit.Register("my-consumer", consumerConf, &MyReceiver{})
rabbit.Start()
```

---

#### 路径 B：渐进式迁移（推荐）

**适用场景**：
- 希望逐步享受新功能
- 有时间进行代码改造
- 追求更好的可靠性

**操作步骤**：

##### 3.1 迁移 Receive 接口

**旧代码**：
```go
type MyReceiver struct{}

func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    log.Printf("Processing: %s", string(message.Body))
    return r.process(message)
}

func (r *MyReceiver) Exception(key string, err error, message amqp.Delivery) {
    log.Printf("Error: %v", err)
}
```

**新代码**：
```go
type MyReceiver struct{}

// 修改签名，增加 ctx 参数
func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    log.Printf("Processing: %s", string(message.Body))
    
    // 将 context 传递给下游
    return r.process(ctx, message)
}

// 修改签名，增加 ctx 参数
func (r *MyReceiver) Exception(ctx context.Context, key string, err error, message amqp.Delivery) {
    log.Printf("Error: %v", err)
    
    // 可以使用 context 进行超时控制
    r.sendAlert(ctx, err)
}

// 更新内部方法签名
func (r *MyReceiver) process(ctx context.Context, message amqp.Delivery) error {
    // 使用 context 进行数据库调用
    result, err := r.db.QueryContext(ctx, "SELECT ...")
    return err
}

func (r *MyReceiver) sendAlert(ctx context.Context, err error) {
    // 使用 context 进行 HTTP 调用
    req, _ := http.NewRequestWithContext(ctx, "POST", alertURL, body)
    r.client.Do(req)
}
```

##### 3.2 配置超时时间

**新增配置**：
```go
err := rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange:       conf.NewFanoutExchange("my-exchange"),
    Queue:          conf.NewQueue("my-queue"),
    Name:           "my-consumer",
    AutoAck:        false,
    HandlerTimeout: 30 * time.Second, // 新增：设置超时时间
}, &MyReceiver{})
```

##### 3.3 迁移消息发送

**旧代码**：
```go
err := rabbit.SendDelayMsg("exchange", "routing-key", delivery, 5000)
```

**新代码**：
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := rabbit.SendDelayMsg(ctx, "exchange", "routing-key", delivery, 5000)
```

##### 3.4 迁移启动和停止

**旧代码**：
```go
rabbit.Start()
// ...
rabbit.Stop()
```

**新代码**：
```go
// 启动时设置超时
startCtx, startCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer startCancel()

if err := rabbit.StartWithContext(startCtx); err != nil {
    log.Fatalf("Failed to start: %v", err)
}

// 优雅关闭
stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
defer stopCancel()

if err := rabbit.StopWithContext(stopCtx); err != nil {
    log.Printf("Graceful shutdown failed: %v", err)
}
```

---

#### 路径 C：完全迁移（最佳实践）

**适用场景**：
- 新项目
- 重构现有项目
- 需要完整的追踪和监控

**额外步骤**：

##### 3.5 集成追踪

```go
import "github.com/wenpiner/rabbitmq-go/tracing"

type MyReceiver struct{}

func (r *MyReceiver) Receive(ctx context.Context, key string, message amqp.Delivery) error {
    // 获取 trace ID
    traceID := tracing.GetTraceID(ctx)
    
    // 在日志中使用
    log.Println(tracing.FormatTraceLog(ctx, "Processing message"))
    
    return r.process(ctx, message)
}

// 发送消息时注入追踪信息
ctx := context.Background()
traceInfo := tracing.TraceInfo{
    TraceID: tracing.GenerateTraceID(),
    SpanID:  tracing.GenerateSpanID(),
}
ctx = tracing.InjectToContext(ctx, traceInfo)

_, err := rabbit.SendMessageWithTrace(ctx, "exchange", "route", true, publishing)
```

---

## 📝 迁移检查清单

### 阶段一：接口迁移
- [ ] 所有 Receive 方法增加 `ctx context.Context` 参数
- [ ] 所有 Exception 方法增加 `ctx context.Context` 参数
- [ ] 更新内部方法签名，传递 context
- [ ] 配置 HandlerTimeout
- [ ] 测试超时功能

### 阶段二：消息发送迁移
- [ ] 所有 SendDelayMsg 调用增加 context 参数
- [ ] 所有 SendDelayMsgByKey 调用增加 context 参数
- [ ] 所有 SendDelayMsgByArgs 调用增加 context 参数
- [ ] 测试发送超时

### 阶段三：启动停止迁移
- [ ] 使用 StartWithContext 替代 Start
- [ ] 使用 StopWithContext 替代 Stop
- [ ] 实现信号处理和优雅关闭
- [ ] 测试优雅关闭

### 阶段四：追踪集成（可选）
- [ ] 集成 tracing 包
- [ ] 在日志中输出 trace ID
- [ ] 配置 OpenTelemetry（可选）
- [ ] 验证追踪链路

---

## 🧪 测试建议

### 1. 单元测试

```go
func TestMyReceiverWithContext(t *testing.T) {
    receiver := &MyReceiver{}
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    delivery := amqp.Delivery{
        Body: []byte("test message"),
    }
    
    err := receiver.Receive(ctx, "test-key", delivery)
    if err != nil {
        t.Errorf("Receive failed: %v", err)
    }
}

func TestMyReceiverTimeout(t *testing.T) {
    receiver := &MyReceiver{}
    
    // 设置很短的超时
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()
    
    time.Sleep(10 * time.Millisecond) // 确保超时
    
    delivery := amqp.Delivery{
        Body: []byte("test message"),
    }
    
    err := receiver.Receive(ctx, "test-key", delivery)
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("Expected timeout error, got: %v", err)
    }
}
```

### 2. 集成测试

```go
func TestGracefulShutdown(t *testing.T) {
    // 1. 启动 RabbitMQ
    // 2. 发送消息
    // 3. 触发优雅关闭
    // 4. 验证所有消息都被处理
}
```

---

## ⚠️ 常见问题

### Q1: 升级后现有代码是否需要修改？

**A**: 不需要。旧接口通过兼容层继续工作，但建议逐步迁移到新接口以享受新功能。

### Q2: 性能会受到影响吗？

**A**: 兼容模式下性能影响 < 1%。新接口模式下，context 传递开销约 < 0.1%。

### Q3: 如何设置合适的超时时间？

**A**: 建议：
- 测量现有消息的平均处理时间
- 设置为平均时间的 2-3 倍
- 根据实际情况调整

### Q4: 旧版本的 Start() 和 Stop() 还能用吗？

**A**: 可以。它们会继续工作，但不支持超时控制和优雅关闭。

### Q5: 如何处理长时间运行的任务？

**A**: 
- 考虑拆分为多个小任务
- 使用异步处理
- 增加超时时间
- 定期检查 context 状态

---

## 📞 获取帮助

如果在迁移过程中遇到问题：

1. 查看 [最佳实践文档](./CONTEXT_BEST_PRACTICES.md)
2. 查看 [示例代码](../examples/)
3. 提交 Issue
4. 联系维护者

---

## 📅 迁移时间表建议

| 阶段 | 时间 | 说明 |
|------|------|------|
| 评估 | 1 天 | 评估现有代码，制定迁移计划 |
| 阶段一 | 1-2 周 | 迁移接口 |
| 阶段二 | 1 周 | 迁移消息发送 |
| 阶段三 | 1 周 | 迁移启动停止 |
| 测试 | 1 周 | 完整测试 |
| 上线 | 1 天 | 灰度发布 |

**总计**：4-6 周（根据项目规模调整）

---

**最后更新**：2025-12-26
