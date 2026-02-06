# RabbitMQ-Go 稳定性测试方案总结

## 📋 概述

本文档总结了为 RabbitMQ-Go 库创建的完整 Docker 化稳定性测试方案。

## 🎯 测试目标

通过容器化的方式，提供一套完整的稳定性测试环境，用于验证:

1. **长时间运行稳定性** - 检测内存泄漏、连接稳定性
2. **高并发性能** - 验证高负载下的吞吐量和延迟
3. **故障恢复能力** - 测试网络故障后的自动恢复
4. **资源泄漏检测** - 检测 Goroutine 和内存泄漏

## 📁 项目结构

```
rabbitmq-go/
├── docker-compose.yml                          # Docker Compose 配置
├── Makefile                                    # Make 命令定义
├── .github/workflows/stability-test.yml        # CI/CD 配置
├── scripts/
│   └── run-stability-tests.sh                 # 测试运行脚本
├── test/
│   ├── STABILITY_TEST_README.md               # 完整测试指南
│   ├── STABILITY_QUICKSTART.md                # 快速开始指南
│   ├── docker/
│   │   ├── rabbitmq/
│   │   │   ├── rabbitmq.conf                  # RabbitMQ 配置
│   │   │   └── enabled_plugins                # 启用的插件
│   │   ├── prometheus/
│   │   │   └── prometheus.yml                 # Prometheus 配置
│   │   ├── grafana/
│   │   │   ├── provisioning/                  # Grafana 数据源配置
│   │   │   └── dashboards/                    # 仪表板定义
│   │   └── stability/
│   │       ├── Dockerfile.long-run            # 长时间测试镜像
│   │       ├── Dockerfile.high-concurrency    # 高并发测试镜像
│   │       ├── Dockerfile.network-chaos       # 网络故障测试镜像
│   │       └── Dockerfile.memory-leak         # 内存泄漏测试镜像
│   └── stability/
│       ├── common/
│       │   ├── metrics.go                     # 指标收集
│       │   └── config.go                      # 配置加载
│       ├── long-run/main.go                   # 长时间测试程序
│       ├── high-concurrency/main.go           # 高并发测试程序
│       ├── network-chaos/main.go              # 网络故障测试程序
│       └── memory-leak/main.go                # 内存泄漏测试程序
└── STABILITY_TEST_SUMMARY.md                  # 本文档
```

## 🐳 Docker 服务

### 基础设施服务

| 服务 | 端口 | 说明 |
|------|------|------|
| **rabbitmq** | 5672, 15672, 15692 | RabbitMQ 服务器 + 管理界面 + Prometheus 指标 |
| **prometheus** | 9090 | 指标收集和存储 |
| **grafana** | 3000 | 可视化监控面板 |

### 测试服务

| 服务 | 测试类型 | 默认时长 | Profile |
|------|---------|---------|---------|
| **stability-long-run** | 长时间稳定性 | 24h | 默认 |
| **stability-high-concurrency** | 高并发压力 | 1h | high-concurrency |
| **stability-network-chaos** | 网络故障恢复 | 2h | chaos |
| **stability-memory-leak** | 内存泄漏检测 | 12h | memory-leak |

## 🚀 使用方式

### 方式 1: 使用 Make 命令 (推荐)

```bash
# 运行长时间测试
make stability-long-run

# 运行高并发测试
make stability-high-concurrency

# 查看状态
make stability-status

# 查看日志
make stability-logs

# 打开监控
make stability-monitor

# 停止测试
make stability-down
```

### 方式 2: 使用脚本

```bash
# 运行测试
./scripts/run-stability-tests.sh long-run

# 查看帮助
./scripts/run-stability-tests.sh --help
```

### 方式 3: 直接使用 Docker Compose

```bash
# 启动基础设施
docker-compose up -d rabbitmq prometheus grafana

# 运行特定测试
docker-compose up -d stability-long-run

# 查看日志
docker-compose logs -f stability-long-run

# 停止
docker-compose down
```

## 📊 监控和指标

### 访问地址

- **RabbitMQ 管理**: http://localhost:15672 (guest/guest)
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **应用指标**: http://localhost:8080/metrics
- **pprof** (内存泄漏测试): http://localhost:6060/debug/pprof/

### 关键指标

#### 应用级指标
- `messages_sent_total` - 发送消息总数
- `messages_received_total` - 接收消息总数
- `messages_failed_total` - 失败消息总数
- `reconnect_count_total` - 重连次数
- `goroutines` - Goroutine 数量
- `memory_alloc_mb` - 内存使用 (MB)

#### RabbitMQ 指标
- `rabbitmq_queue_messages` - 队列消息数
- `rabbitmq_connections` - 连接数
- `rabbitmq_process_resident_memory_bytes` - 内存使用

## ✅ 验收标准

### 长时间稳定性测试
- ✅ 运行 24 小时无崩溃
- ✅ 内存增长 < 1MB/hour
- ✅ Goroutine 数量稳定 (±5)
- ✅ 消息丢失率 = 0%
- ✅ 无 ERROR 级别日志

### 高并发压力测试
- ✅ 吞吐量 > 10,000 msg/s
- ✅ P99 延迟 < 100ms
- ✅ 错误率 < 0.01%
- ✅ CPU 使用率 < 80%

### 网络故障恢复测试
- ✅ 重连成功率 > 99.9%
- ✅ 消息不丢失
- ✅ 恢复时间 < 10s
- ✅ 自动重连次数 > 0

### 内存泄漏检测
- ✅ Goroutine 泄漏 = 0
- ✅ 内存增长 < 5MB (GC 后)
- ✅ Channel 正确关闭
- ✅ 完成 1000 次创建/销毁循环

## 🔧 配置说明

### 环境变量

所有测试都支持以下环境变量:

```bash
RABBITMQ_HOST=rabbitmq        # RabbitMQ 主机
RABBITMQ_PORT=5672            # RabbitMQ 端口
RABBITMQ_USER=guest           # 用户名
RABBITMQ_PASS=guest           # 密码
TEST_DURATION=24h             # 测试时长
MESSAGE_RATE=100              # 消息速率 (msg/s)
CONSUMER_COUNT=10             # 消费者数量
BATCH_SIZE=10                 # 批量大小
METRICS_ADDR=:8080            # 指标服务地址
```

### 自定义配置

编辑 `docker-compose.yml` 修改测试参数:

```yaml
environment:
  TEST_DURATION: 48h      # 修改测试时长
  MESSAGE_RATE: 1000      # 修改消息速率
  CONSUMER_COUNT: 20      # 修改消费者数量
```

## 📈 CI/CD 集成

### GitHub Actions

已提供 `.github/workflows/stability-test.yml`:

- **手动触发**: 可选择测试类型和时长
- **定期运行**: 每周日自动运行
- **PR 检查**: PR 时运行快速测试 (5分钟)

触发方式:
```bash
# 在 GitHub Actions 页面手动触发
# 或通过 API
gh workflow run stability-test.yml -f test_type=long-run -f duration=2h
```

## 📚 文档

- **快速开始**: [test/STABILITY_QUICKSTART.md](test/STABILITY_QUICKSTART.md)
- **完整指南**: [test/STABILITY_TEST_README.md](test/STABILITY_TEST_README.md)
- **集成测试**: [INTEGRATION_TEST.md](INTEGRATION_TEST.md)

## 🎓 最佳实践

1. **逐步增加负载**: 先运行短时间测试验证环境
2. **监控资源**: 确保主机有足够资源 (建议 4 核 8GB+)
3. **保存结果**: 定期导出日志和指标建立基准
4. **对比分析**: 对比不同版本的测试结果
5. **CI 集成**: 在发布前运行完整稳定性测试

## 🔍 故障排查

常见问题和解决方案请参考 [test/STABILITY_TEST_README.md](test/STABILITY_TEST_README.md#故障排查)

## 📝 下一步

1. ✅ 运行快速测试验证环境
2. ✅ 运行完整的 24 小时稳定性测试
3. ✅ 分析测试结果并建立性能基准
4. ✅ 集成到 CI/CD 流程
5. ✅ 定期运行并对比结果

---

**创建时间**: 2026-02-06  
**版本**: 1.0.0  
**维护者**: RabbitMQ-Go Team

