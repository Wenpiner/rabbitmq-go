# 稳定性测试实战示例

本文档通过实际示例演示如何使用稳定性测试套件。

## 🎯 场景 1: 快速验证 (5分钟)

**目标**: 快速验证测试环境是否正常工作

```bash
# 1. 启动测试 (修改为 5 分钟)
docker-compose up -d rabbitmq prometheus grafana

# 等待 RabbitMQ 就绪
sleep 20

# 2. 运行 5 分钟快速测试
docker-compose run -e TEST_DURATION=5m stability-long-run

# 3. 查看结果
docker-compose logs stability-long-run | grep "最终统计" -A 10

# 4. 清理
docker-compose down
```

**预期结果**:
```
=== 最终统计 ===
运行时间: 300.00 秒
发送消息: 30000 (100.00 msg/s)
接收消息: 30000 (100.00 msg/s)
失败消息: 0
重连次数: 0
错误次数: 0
Goroutine: 25
内存使用: 15 MB
```

## 🎯 场景 2: 长时间稳定性测试 (24小时)

**目标**: 验证系统在长时间运行下的稳定性

```bash
# 1. 启动测试
make stability-long-run

# 2. 打开监控面板
make stability-monitor
# 访问 http://localhost:3000

# 3. 定期查看状态 (每小时)
watch -n 3600 'make stability-status'

# 4. 查看实时日志
make stability-logs

# 5. 24 小时后查看结果
docker-compose logs stability-long-run | tail -50

# 6. 导出测试报告
docker-compose logs stability-long-run > stability-test-24h-$(date +%Y%m%d).log
```

**验收检查**:
- [ ] 运行 24 小时无崩溃
- [ ] 内存增长 < 24 MB
- [ ] Goroutine 数量稳定 (±5)
- [ ] 消息丢失率 = 0%
- [ ] 无 ERROR 日志

## 🎯 场景 3: 高并发压力测试

**目标**: 测试系统能否承受 10,000 msg/s 的负载

```bash
# 1. 启动测试
make stability-high-concurrency

# 2. 实时监控性能
# 在 Grafana 中查看:
# - Message Rate 图表
# - Queue Depth 图表
# - Memory Usage 图表

# 3. 查看实时指标
watch -n 5 'curl -s http://localhost:8080/metrics | grep messages'

# 4. 测试完成后分析
docker-compose logs stability-high-concurrency | grep "最终统计" -A 10
```

**性能指标**:
```bash
# 计算平均吞吐量
sent=$(docker-compose logs stability-high-concurrency | grep "发送消息:" | awk '{print $2}')
duration=$(docker-compose logs stability-high-concurrency | grep "运行时间:" | awk '{print $2}')
echo "平均吞吐量: $(echo "$sent / $duration" | bc) msg/s"
```

## 🎯 场景 4: 网络故障恢复测试

**目标**: 验证系统在网络故障后能自动恢复

```bash
# 1. 启动测试
make stability-network-chaos

# 2. 观察重连过程
docker-compose logs -f stability-network-chaos | grep -E "网络故障|恢复|重连"

# 3. 查看重连统计
curl http://localhost:8080/metrics | grep reconnect_count_total

# 4. 验证消息不丢失
# 对比发送和接收数量
curl http://localhost:8080/metrics | grep -E "messages_sent_total|messages_received_total"
```

**预期行为**:
```
⚠️  模拟网络故障 (持续 30s)
[WARN] 连接断开
[INFO] 尝试重连 (attempt: 1)
[INFO] 尝试重连 (attempt: 2)
✅ 网络故障恢复
[INFO] 状态变化: Disconnected -> Connected
```

## 🎯 场景 5: 内存泄漏检测

**目标**: 检测是否存在内存和 Goroutine 泄漏

```bash
# 1. 启动测试
make stability-memory-leak

# 2. 使用 pprof 分析内存
# 初始快照
curl http://localhost:6060/debug/pprof/heap > heap-initial.pprof

# 等待 1 小时
sleep 3600

# 最终快照
curl http://localhost:6060/debug/pprof/heap > heap-final.pprof

# 对比分析
go tool pprof -base heap-initial.pprof heap-final.pprof

# 3. 分析 Goroutine
curl http://localhost:6060/debug/pprof/goroutine > goroutine.pprof
go tool pprof goroutine.pprof

# 4. 生成火焰图
go tool pprof -http=:8081 heap-final.pprof
# 访问 http://localhost:8081
```

**检查清单**:
```bash
# 查看内存增长趋势
docker-compose logs stability-memory-leak | grep "循环" | tail -20

# 预期输出:
# 循环 10: Goroutines=25 (增长=0), Memory=15 MB (增长=0 MB)
# 循环 20: Goroutines=25 (增长=0), Memory=15 MB (增长=0 MB)
# ...
```

## 🎯 场景 6: 对比测试 (版本回归)

**目标**: 对比不同版本的性能

```bash
# 1. 测试当前版本
make stability-long-run
docker-compose logs stability-long-run > results-v2.0.0.log

# 2. 切换到旧版本
git checkout v1.0.0

# 3. 测试旧版本
make stability-long-run
docker-compose logs stability-long-run > results-v1.0.0.log

# 4. 对比结果
diff results-v1.0.0.log results-v2.0.0.log

# 5. 提取关键指标对比
for version in v1.0.0 v2.0.0; do
  echo "=== $version ==="
  grep "最终统计" -A 10 results-$version.log
done
```

## 🎯 场景 7: CI/CD 集成

**目标**: 在 CI 环境中运行稳定性测试

```bash
# 在 GitHub Actions 中手动触发
gh workflow run stability-test.yml \
  -f test_type=long-run \
  -f duration=1h

# 查看运行状态
gh run list --workflow=stability-test.yml

# 下载测试结果
gh run download <run-id>
```

## 📊 结果分析模板

### 测试报告模板

```markdown
# 稳定性测试报告

**测试日期**: 2026-02-06
**测试版本**: v2.0.0
**测试类型**: 长时间稳定性测试
**测试时长**: 24 小时

## 测试环境
- OS: Ubuntu 22.04
- CPU: 4 核
- Memory: 8 GB
- RabbitMQ: 3.12

## 测试结果

| 指标 | 结果 | 目标 | 状态 |
|------|------|------|------|
| 运行时长 | 24h | 24h | ✅ |
| 发送消息 | 8,640,000 | - | ✅ |
| 接收消息 | 8,640,000 | - | ✅ |
| 消息丢失 | 0 | 0 | ✅ |
| 平均吞吐量 | 100 msg/s | >50 msg/s | ✅ |
| 内存增长 | 5 MB | <24 MB | ✅ |
| Goroutine 增长 | 0 | 0 | ✅ |
| 重连次数 | 0 | - | ✅ |
| 错误次数 | 0 | 0 | ✅ |

## 结论
✅ 测试通过，系统稳定性良好

## 建议
- 可以投入生产使用
- 建议定期运行稳定性测试
```

## 💡 最佳实践

1. **逐步增加负载**
   ```bash
   # 先 5 分钟
   docker-compose run -e TEST_DURATION=5m stability-long-run
   
   # 再 1 小时
   docker-compose run -e TEST_DURATION=1h stability-long-run
   
   # 最后 24 小时
   make stability-long-run
   ```

2. **保存测试结果**
   ```bash
   # 创建结果目录
   mkdir -p test-results/$(date +%Y%m%d)
   
   # 导出日志
   docker-compose logs > test-results/$(date +%Y%m%d)/logs.txt
   
   # 导出指标
   curl http://localhost:8080/metrics > test-results/$(date +%Y%m%d)/metrics.txt
   ```

3. **建立性能基准**
   ```bash
   # 记录基准数据
   echo "v2.0.0,100,8640000,8640000,0,5" >> performance-baseline.csv
   # 格式: version,msg_rate,sent,received,failed,memory_mb
   ```

## 🆘 故障排查示例

### 问题: 测试容器启动失败

```bash
# 1. 查看容器状态
docker-compose ps

# 2. 查看详细日志
docker-compose logs stability-long-run

# 3. 检查 RabbitMQ
docker-compose exec rabbitmq rabbitmq-diagnostics ping

# 4. 重启
docker-compose restart stability-long-run
```

### 问题: 内存持续增长

```bash
# 1. 使用 pprof 分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 2. 查看 top 内存使用
(pprof) top

# 3. 查看调用栈
(pprof) list <function_name>
```

---

**更多示例请参考**: [test/STABILITY_TEST_README.md](test/STABILITY_TEST_README.md)

