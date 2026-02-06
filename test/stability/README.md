# 稳定性测试代码说明

本目录包含所有稳定性测试的源代码。

## 目录结构

```
stability/
├── common/              # 通用工具库
│   ├── metrics.go      # 指标收集
│   └── config.go       # 配置加载
├── long-run/           # 长时间稳定性测试
│   └── main.go
├── high-concurrency/   # 高并发压力测试
│   └── main.go
├── network-chaos/      # 网络故障恢复测试
│   └── main.go
└── memory-leak/        # 内存泄漏检测测试
    └── main.go
```

## 通用库说明

### metrics.go

提供统一的指标收集功能:

- `NewMetrics()` - 创建指标收集器
- `RecordSent()` - 记录发送消息
- `RecordReceived()` - 记录接收消息
- `RecordFailed()` - 记录失败消息
- `RecordError()` - 记录错误
- `RecordReconnect()` - 记录重连
- `GetStats()` - 获取统计信息
- `PrintStats()` - 打印统计信息
- `ServeMetrics()` - 启动 HTTP 指标服务

### config.go

从环境变量加载配置:

- `LoadConfig()` - 加载测试配置
- 支持的环境变量:
  - `RABBITMQ_HOST` - RabbitMQ 主机
  - `RABBITMQ_PORT` - RabbitMQ 端口
  - `RABBITMQ_USER` - 用户名
  - `RABBITMQ_PASS` - 密码
  - `TEST_DURATION` - 测试时长
  - `MESSAGE_RATE` - 消息速率
  - `CONSUMER_COUNT` - 消费者数量
  - `BATCH_SIZE` - 批量大小

## 测试程序说明

### 1. long-run (长时间稳定性测试)

**目标**: 验证系统在长时间运行下的稳定性

**特点**:
- 持续发送和消费消息
- 监控内存和 Goroutine 数量
- 每 30 秒打印统计信息
- 暴露 Prometheus 指标

**运行**:
```bash
cd long-run
go run main.go
```

### 2. high-concurrency (高并发压力测试)

**目标**: 测试系统在高并发场景下的性能

**特点**:
- 多个发送协程并发发送
- 大量并发消费者
- 批量发送优化
- 高 QoS 预取设置

**运行**:
```bash
cd high-concurrency
go run main.go
```

### 3. network-chaos (网络故障恢复测试)

**目标**: 验证系统在网络故障后的自动恢复能力

**特点**:
- 定期模拟网络故障
- 验证自动重连机制
- 监控重连次数
- 验证消息不丢失

**运行**:
```bash
cd network-chaos
go run main.go
```

### 4. memory-leak (内存泄漏检测测试)

**目标**: 检测是否存在内存泄漏和 Goroutine 泄漏

**特点**:
- 反复创建和销毁消费者
- 监控内存和 Goroutine 变化
- 启用 pprof 性能分析
- 强制 GC 验证内存释放

**运行**:
```bash
cd memory-leak
PPROF_ENABLED=true go run main.go
```

## 本地开发

### 编译测试程序

```bash
# 编译所有测试程序
go build ./long-run
go build ./high-concurrency
go build ./network-chaos
go build ./memory-leak
```

### 运行单个测试

```bash
# 设置环境变量
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=5672
export TEST_DURATION=5m
export MESSAGE_RATE=100

# 运行测试
cd long-run
go run main.go
```

### 查看指标

```bash
# 应用指标
curl http://localhost:8080/metrics

# pprof (仅内存泄漏测试)
curl http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/heap
```

## 添加新测试

1. 在 `stability/` 下创建新目录
2. 创建 `main.go` 实现测试逻辑
3. 使用 `common` 包的工具函数
4. 在 `test/docker/stability/` 创建 Dockerfile
5. 在 `docker-compose.yml` 添加服务定义
6. 更新文档

## 注意事项

1. 所有测试都应该:
   - 使用 `common.Metrics` 收集指标
   - 暴露 HTTP 指标端点
   - 支持优雅关闭 (SIGINT/SIGTERM)
   - 打印最终统计信息

2. 环境变量命名规范:
   - 使用大写字母和下划线
   - 提供合理的默认值
   - 在文档中说明

3. 日志输出:
   - 使用结构化日志
   - 包含时间戳
   - 区分不同级别 (INFO/WARN/ERROR)

## 相关文档

- [快速开始](../STABILITY_QUICKSTART.md)
- [完整指南](../STABILITY_TEST_README.md)
- [总结文档](../../STABILITY_TEST_SUMMARY.md)

