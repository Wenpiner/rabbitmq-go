# 重试机制使用指南

## 概述

RabbitMQ-Go 支持灵活的消息重试机制，包括：
- **指数退避 (Exponential Backoff)** - 默认策略，适合大多数场景
- **线性退避 (Linear Backoff)** - 固定间隔重试
- **自定义策略** - 通过接口实现完全自定义的重试逻辑
- **向后兼容** - 未配置时自动使用旧版线性重试

## 快速开始

### 1. 使用默认指数退避

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry:    conf.NewRetryConf(), // 默认配置
}, receiver)
```

**默认参数:**
- 最大重试次数: 5次
- 初始延迟: 1秒
- 倍数: 2.0
- 最大延迟: 5分钟
- 抖动: 开启

**重试时间序列:**
```
第1次重试: ~1秒
第2次重试: ~2秒
第3次重试: ~4秒
第4次重试: ~8秒
第5次重试: ~16秒
```

### 2. 自定义指数退避参数

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry:    conf.NewExponentialRetryConf(10, 500*time.Millisecond, 1.5),
    // 参数: maxRetries=10, initialDelay=500ms, multiplier=1.5
}, receiver)
```

### 3. 使用线性退避

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry:    conf.NewLinearRetryConf(3, 2*time.Second),
    // 参数: maxRetries=3, initialDelay=2秒
}, receiver)
```

**重试时间序列:**
```
第1次重试: 2秒
第2次重试: 4秒
第3次重试: 6秒
```

### 4. 完全自定义配置

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry: conf.RetryConf{
        Enable:       true,
        MaxRetries:   7,
        Strategy:     "exponential",
        InitialDelay: time.Second,
        Multiplier:   2.5,
        MaxDelay:     10 * time.Minute,
        Jitter:       true,
    },
}, receiver)
```

### 5. 禁用重试

```go
// 方式1: 设置 Enable = false
Retry: conf.RetryConf{
    Enable: false,
}

// 方式2: 设置 MaxRetries = 0
Retry: conf.RetryConf{
    Enable:     true,
    MaxRetries: 0,
}

// 方式3: 不配置 Retry 字段 (使用旧版行为)
// 未配置时会使用旧版线性重试 (3次, 3秒间隔)
```

## 高级用法

### 自定义重试策略

实现 `ReceiveWithRetry` 接口来提供完全自定义的重试逻辑:

```go
type MyReceiver struct {
    // 你的字段
}

func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    // 处理消息
    return nil
}

func (r *MyReceiver) Exception(key string, err error, message amqp.Delivery) {
    // 异常处理
}

// 实现自定义重试策略
func (r *MyReceiver) GetRetryStrategy() conf.RetryStrategy {
    return conf.NewExponentialRetry(15, 100*time.Millisecond, 2.0, time.Hour, true)
}
```

**注意:** 当实现了 `GetRetryStrategy()` 方法时，`ConsumerConf.Retry` 配置会被忽略。

## 重试元数据

重试过程中，以下信息会存储在消息的 Headers 中:

| Header 字段 | 类型 | 说明 |
|------------|------|------|
| `retry_nums` | int32 | 当前重试次数 |
| `retry_delay` | int32 | 本次重试的延迟时间(毫秒) |
| `first_attempt_time` | int64 | 首次尝试的时间戳 |
| `last_error` | string | 最后一次失败的错误信息 |

### 在业务代码中访问重试信息

```go
func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    // 获取重试次数
    if retryNum, ok := message.Headers["retry_nums"].(int32); ok {
        log.Printf("当前是第 %d 次重试", retryNum)
    }
    
    // 获取首次尝试时间
    if firstTime, ok := message.Headers["first_attempt_time"].(int64); ok {
        elapsed := time.Now().Unix() - firstTime
        log.Printf("消息已处理 %d 秒", elapsed)
    }
    
    // 你的业务逻辑
    return nil
}
```

## 配置参数详解

### RetryConf 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Enable` | bool | true | 是否启用重试 |
| `MaxRetries` | int32 | 5 | 最大重试次数 |
| `Strategy` | string | "exponential" | 重试策略: "linear" 或 "exponential" |
| `InitialDelay` | time.Duration | 1s | 初始延迟时间 |
| `Multiplier` | float64 | 2.0 | 指数退避倍数(仅exponential) |
| `MaxDelay` | time.Duration | 5m | 最大延迟时间，防止无限增长 |
| `Jitter` | bool | true | 是否添加随机抖动(±25%) |

### 什么是 Jitter (抖动)?

Jitter 在计算出的延迟基础上添加 ±25% 的随机偏移，避免大量消息同时重试造成的"雷鸣群效应"。

**示例:**
```
计算延迟: 10秒
启用 Jitter: 7.5秒 ~ 12.5秒 (随机)
```

## 向后兼容性

为了保持向后兼容，未配置 `Retry` 字段的消费者会自动使用旧版重试行为:

```go
// 旧代码 - 无需修改
rabbit.Register("old-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("test"),
    Queue:    conf.NewQueue("test"),
    AutoAck:  false,
    // 没有 Retry 字段
}, receiver)

// 自动使用: LinearRetry{MaxRetries: 3, InitialDelay: 3000}
// 重试时间: 3秒, 6秒, 9秒
```

## 最佳实践

1. **大多数场景使用默认指数退避**
   ```go
   Retry: conf.NewRetryConf()
   ```

2. **短暂故障使用较少重试次数**
   ```go
   Retry: conf.NewExponentialRetryConf(3, 500*time.Millisecond, 2.0)
   ```

3. **长时间运行的任务使用更多重试和更长延迟**
   ```go
   Retry: conf.NewExponentialRetryConf(10, 5*time.Second, 1.5)
   ```

4. **生产环境建议开启 Jitter**
   ```go
   Retry: conf.RetryConf{
       // ...
       Jitter: true, // 避免雷鸣群效应
   }
   ```

5. **根据错误类型决定是否重试**
   ```go
   func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
       err := processMessage(message)
       if err != nil {
           // 业务错误不重试
           if errors.Is(err, ErrInvalidData) {
               log.Printf("数据无效，不重试: %v", err)
               return nil // 返回 nil 表示成功，不会重试
           }
           // 临时错误重试
           return err
       }
       return nil
   }
   ```

## 监控和调试

查看重试日志:

```
消息重试已调度 (key: my-consumer, retry: 1, delay: 1000ms)
消息重试已调度 (key: my-consumer, retry: 2, delay: 2000ms)
消息达到最大重试次数或重试已禁用 (key: my-consumer, retry: 5), 进入异常环节
```

## 常见问题

**Q: 如何知道消息重试了多少次?**  
A: 检查 `message.Headers["retry_nums"]`

**Q: 重试次数用完后会怎样?**  
A: 会调用 `Exception(key, err, message)` 方法，你可以在这里记录日志、入库等

**Q: 可以针对不同错误使用不同重试策略吗?**  
A: 可以在 `Receive` 方法中根据错误类型返回 nil (不重试) 或 error (重试)

**Q: 重试会影响其他消息的处理吗?**  
A: 不会，重试通过延迟队列实现，不会阻塞当前消费者

**Q: 如何升级现有代码使用新的重试机制?**  
A: 只需在 `ConsumerConf` 中添加 `Retry` 字段即可，不影响现有代码

