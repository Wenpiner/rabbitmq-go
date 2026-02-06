# RabbitMQ-Go Examples

本目录包含 RabbitMQ-Go 客户端库的各种使用示例。

## 前置条件

确保你已经安装并运行了 RabbitMQ：

```bash
# 使用 Docker 启动 RabbitMQ
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management
```

访问管理界面：http://localhost:15672 (guest/guest)

## 示例列表

### 1. 基础示例 (01-basic)

演示最基本的发布和消费功能。

**功能点:**
- 创建客户端并连接
- 注册消费者
- 发布消息
- 优雅关闭

**运行:**
```bash
cd examples/01-basic
go run main.go
```

**预期输出:**
```
=== RabbitMQ 基础示例 ===
✅ 已连接到 RabbitMQ
✅ 消费者已注册
📤 已发送消息 #1
📨 收到消息: Hello RabbitMQ #1
...
```

---

### 2. 批量发布 (02-batch-publish)

演示如何高效地批量发送消息。

**功能点:**
- 普通批量发送（高性能）
- 带确认的批量发送（可靠）
- 性能对比

**运行:**
```bash
cd examples/02-batch-publish
go run main.go
```

**预期输出:**
```
📤 方式 1: 普通批量发送
✅ 发送 100 条消息，耗时: 50ms
📤 方式 2: 带确认的批量发送
✅ 发送并确认 10 条消息，耗时: 200ms
```

---

### 3. 分布式追踪 (03-tracing)

演示如何使用内置的追踪功能进行链路追踪。

**功能点:**
- 自动生成追踪 ID
- 追踪信息传播
- 从消息中提取追踪信息
- 手动创建追踪上下文

**运行:**
```bash
cd examples/03-tracing
go run main.go
```

**预期输出:**
```
📨 收到消息:
   内容: Auto-traced message
   TraceID: 1a2b3c4d5e6f7g8h
   SpanID: 9i0j1k2l3m4n5o6p
   ParentSpanID: 
```

---

### 4. 重试策略 (04-retry-strategy)

演示不同的重试策略：指数退避、线性退避。

**功能点:**
- 指数退避重试
- 重试次数控制
- 错误处理
- 延迟队列

**运行:**
```bash
cd examples/04-retry-strategy
go run main.go
```

**预期输出:**
```
📨 处理消息 (第 1 次尝试，重试次数: 0): Test message
❌ 错误处理器: 模拟处理失败 (重试次数: 0)
📨 处理消息 (第 2 次尝试，重试次数: 1): Test message
...
✅ 消息处理成功！
```

---

### 5. 并发处理 (05-concurrency)

演示如何配置并发处理消息以提高吞吐量。

**功能点:**
- 配置并发数
- QoS 设置
- 并发统计
- 性能测试

**运行:**
```bash
cd examples/05-concurrency
go run main.go
```

**预期输出:**
```
✅ 消费者已注册 (并发数: 10)
📤 发送 50 条消息...
🔄 [Worker 1] 开始处理: Message #1 (当前并发: 1)
🔄 [Worker 2] 开始处理: Message #2 (当前并发: 2)
...
📊 处理完成统计:
   总消息数: 50
   总耗时: 3.2s
   最大并发数: 10
   吞吐量: 15.63 条/秒
```

---

### 6. 优雅关闭 (06-graceful-shutdown)

演示如何在关闭时确保所有消息都被处理完成。

**功能点:**
- 优雅关闭机制
- 等待消息处理完成
- 关闭超时控制
- 消息统计

**运行:**
```bash
cd examples/06-graceful-shutdown
go run main.go
# 在消息处理过程中按 Ctrl+C
```

**预期输出:**
```
🛑 收到关闭信号，开始优雅关闭...
   当前正在处理: 3 条消息
   已处理: 7/10 条消息
✅ 优雅关闭完成
   关闭耗时: 2.1s
   最终处理: 10/10 条消息
   🎉 所有消息都已处理完成！
```

---

## 通用运行方式

所有示例都可以通过以下方式运行：

```bash
# 进入示例目录
cd examples/<example-name>

# 运行示例
go run main.go

# 或者先编译再运行
go build -o example
./example
```

## 配置说明

所有示例默认使用以下 RabbitMQ 配置：

```go
conf.RabbitConf{
    Scheme:   "amqp",
    Host:     "localhost",
    Port:     5672,
    Username: "guest",
    Password: "guest",
    VHost:    "/",
}
```

如果你的 RabbitMQ 配置不同，请修改示例代码中的配置。

## 学习路径建议

1. **初学者**: 从 `01-basic` 开始，了解基本用法
2. **性能优化**: 学习 `02-batch-publish` 和 `05-concurrency`
3. **可靠性**: 学习 `04-retry-strategy` 和 `06-graceful-shutdown`
4. **可观测性**: 学习 `03-tracing`

## 故障排查

### 连接失败

如果示例运行时提示连接失败：

1. 确认 RabbitMQ 正在运行：
   ```bash
   docker ps | grep rabbitmq
   ```

2. 检查端口是否开放：
   ```bash
   telnet localhost 5672
   ```

3. 查看 RabbitMQ 日志：
   ```bash
   docker logs rabbitmq
   ```

### 消息未被消费

如果消息发送成功但未被消费：

1. 检查队列绑定是否正确
2. 查看 RabbitMQ 管理界面的队列状态
3. 确认 routing key 匹配

## 更多资源

- [主 README](../README.md) - 完整的功能文档
- [集成测试指南](../INTEGRATION_TEST.md) - 如何运行集成测试
- [API 文档](https://pkg.go.dev/github.com/wenpiner/rabbitmq-go/v2) - 完整的 API 参考

