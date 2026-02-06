# RabbitMQ-Go 稳定性测试指南

本文档介绍如何使用 Docker 容器化环境运行 RabbitMQ-Go 的稳定性测试。

## 📋 目录

- [测试概述](#测试概述)
- [快速开始](#快速开始)
- [测试类型](#测试类型)
- [监控和指标](#监控和指标)
- [测试结果分析](#测试结果分析)
- [故障排查](#故障排查)

## 🎯 测试概述

稳定性测试套件包含以下测试场景:

| 测试类型 | 目标 | 运行时长 | 资源需求 |
|---------|------|---------|---------|
| **长时间稳定性** | 检测内存泄漏、连接稳定性 | 24小时 | 低 |
| **高并发压力** | 测试高负载下的性能 | 1小时 | 高 |
| **网络故障恢复** | 验证自动重连机制 | 2小时 | 中 |
| **内存泄漏检测** | 检测资源泄漏 | 12小时 | 中 |

## 🚀 快速开始

### 前置条件

- Docker 20.10+
- Docker Compose 1.29+
- 至少 4GB 可用内存
- 至少 10GB 可用磁盘空间

### 运行测试

```bash
# 1. 运行长时间稳定性测试
./scripts/run-stability-tests.sh long-run

# 2. 运行高并发压力测试
./scripts/run-stability-tests.sh high-concurrency

# 3. 运行网络故障恢复测试
./scripts/run-stability-tests.sh network-chaos

# 4. 运行内存泄漏检测测试
./scripts/run-stability-tests.sh memory-leak

# 5. 运行所有测试
./scripts/run-stability-tests.sh all
```

### 查看测试状态

```bash
# 查看容器状态
./scripts/run-stability-tests.sh -s

# 查看测试日志
./scripts/run-stability-tests.sh -l long-run

# 打开监控面板
./scripts/run-stability-tests.sh -m
```

### 停止测试

```bash
# 停止并清理所有容器
./scripts/run-stability-tests.sh -d
```

## 📊 测试类型详解

### 1. 长时间稳定性测试 (Long Run)

**目标**: 验证系统在长时间运行下的稳定性

**测试场景**:
- 持续运行 24 小时
- 每秒发送 100 条消息
- 10 个并发消费者
- 监控内存、Goroutine 数量

**环境变量**:
```bash
TEST_DURATION=24h
MESSAGE_RATE=100
CONSUMER_COUNT=10
```

**验收标准**:
- ✅ 内存增长 < 1MB/hour
- ✅ Goroutine 数量稳定
- ✅ 消息丢失率 = 0%
- ✅ 无错误日志

### 2. 高并发压力测试 (High Concurrency)

**目标**: 测试系统在高并发场景下的性能

**测试场景**:
- 运行 1 小时
- 每秒发送 10,000 条消息
- 100 个并发消费者
- 批量发送 (每批 100 条)

**环境变量**:
```bash
TEST_DURATION=1h
MESSAGE_RATE=10000
CONSUMER_COUNT=100
BATCH_SIZE=100
```

**验收标准**:
- ✅ 吞吐量 > 10,000 msg/s
- ✅ P99 延迟 < 100ms
- ✅ 错误率 < 0.01%
- ✅ CPU 使用率 < 80%

### 3. 网络故障恢复测试 (Network Chaos)

**目标**: 验证系统在网络故障后的自动恢复能力

**测试场景**:
- 运行 2 小时
- 每 5 分钟模拟一次网络故障
- 每次故障持续 30 秒
- 验证自动重连和消息不丢失

**环境变量**:
```bash
TEST_DURATION=2h
CHAOS_INTERVAL=5m
CHAOS_DURATION=30s
```

**验收标准**:
- ✅ 重连成功率 > 99.9%
- ✅ 消息丢失率 = 0%
- ✅ 恢复时间 < 10s

### 4. 内存泄漏检测测试 (Memory Leak)

**目标**: 检测是否存在内存泄漏和 Goroutine 泄漏

**测试场景**:
- 运行 12 小时
- 反复创建和销毁消费者 (1000 次循环)
- 每个周期发送 10 条消息
- 启用 pprof 性能分析

**环境变量**:
```bash
TEST_DURATION=12h
CYCLE_COUNT=1000
PPROF_ENABLED=true
```

**验收标准**:
- ✅ Goroutine 泄漏 = 0
- ✅ 内存增长 < 5MB (GC 后)
- ✅ Channel 正确关闭

## 📈 监控和指标

### 访问监控面板

测试启动后，可以通过以下地址访问监控:

- **RabbitMQ 管理界面**: http://localhost:15672 (guest/guest)
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### 关键指标

#### 应用指标 (通过 /metrics 端点)

```
messages_sent_total         # 发送消息总数
messages_received_total     # 接收消息总数
messages_failed_total       # 失败消息总数
reconnect_count_total       # 重连次数
goroutines                  # 当前 Goroutine 数量
memory_alloc_mb            # 内存使用 (MB)
```

#### RabbitMQ 指标

```
rabbitmq_queue_messages                    # 队列消息数
rabbitmq_queue_messages_published_total    # 发布消息总数
rabbitmq_queue_messages_delivered_total    # 投递消息总数
rabbitmq_connections                       # 连接数
rabbitmq_process_resident_memory_bytes     # 内存使用
```

### pprof 性能分析

内存泄漏测试启用了 pprof，可以通过以下方式分析:

```bash
# 查看堆内存
go tool pprof http://localhost:6060/debug/pprof/heap

# 查看 Goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine

# 生成火焰图
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
```

## 📊 测试结果分析

### 查看实时日志

```bash
# 查看特定测试的日志
docker-compose logs -f stability-long-run

# 查看所有测试日志
docker-compose logs -f
```

### 导出测试报告

测试完成后，可以从容器中导出统计信息:

```bash
# 查看最终统计
docker-compose logs stability-long-run | grep "最终统计" -A 10
```

### 性能基准

参考性能指标 (基于 4 核 8GB 环境):

| 指标 | 目标值 | 优秀 | 良好 | 需优化 |
|------|--------|------|------|--------|
| 吞吐量 | >10k msg/s | >50k | >20k | <10k |
| P99 延迟 | <100ms | <50ms | <100ms | >100ms |
| 内存增长 | <1MB/h | <0.5MB/h | <1MB/h | >1MB/h |
| Goroutine 泄漏 | 0 | 0 | 0 | >0 |

## 🔧 故障排查

### 常见问题

#### 1. 容器启动失败

```bash
# 查看容器日志
docker-compose logs rabbitmq

# 检查端口占用
lsof -i :5672
lsof -i :15672
```

#### 2. RabbitMQ 连接失败

```bash
# 检查 RabbitMQ 健康状态
docker-compose exec rabbitmq rabbitmq-diagnostics ping

# 查看 RabbitMQ 日志
docker-compose logs rabbitmq
```

#### 3. 测试容器异常退出

```bash
# 查看退出状态
docker-compose ps -a

# 查看容器日志
docker-compose logs stability-long-run
```

#### 4. 内存不足

```bash
# 检查 Docker 资源限制
docker stats

# 增加 Docker 内存限制 (Docker Desktop)
# Preferences -> Resources -> Memory
```

### 清理和重置

```bash
# 停止所有容器
./scripts/run-stability-tests.sh -d

# 清理所有数据卷
docker-compose down -v

# 清理 Docker 缓存
docker system prune -a
```

## 📝 自定义测试

### 修改测试参数

编辑 `docker-compose.yml` 中的环境变量:

```yaml
environment:
  TEST_DURATION: 48h      # 修改测试时长
  MESSAGE_RATE: 1000      # 修改消息速率
  CONSUMER_COUNT: 20      # 修改消费者数量
```

### 添加新的测试场景

1. 在 `test/stability/` 下创建新目录
2. 编写测试代码
3. 创建对应的 Dockerfile
4. 在 `docker-compose.yml` 中添加服务定义

## 🎓 最佳实践

1. **逐步增加负载**: 先运行低负载测试，确认无问题后再运行高负载测试
2. **监控资源使用**: 实时监控 CPU、内存、网络使用情况
3. **保存测试结果**: 定期导出日志和指标数据
4. **对比基准**: 建立性能基准，对比不同版本的测试结果
5. **CI/CD 集成**: 将稳定性测试集成到 CI/CD 流程中

## 📚 相关文档

- [集成测试指南](../INTEGRATION_TEST.md)
- [快速开始](../QUICKSTART.md)
- [主 README](../README.md)

