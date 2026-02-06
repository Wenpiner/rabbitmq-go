# RabbitMQ-Go 集成测试指南

本文档说明如何运行 RabbitMQ-Go 的集成测试。

## 前置条件

### 1. 安装 Docker

确保你的系统已安装 Docker。

### 2. 启动 RabbitMQ 容器

使用以下命令启动 RabbitMQ Docker 容器：

```bash
# 启动 RabbitMQ（带管理界面）
docker run -d \
  --name rabbitmq-test \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management

# 查看容器状态
docker ps | grep rabbitmq-test

# 查看日志
docker logs rabbitmq-test
```

### 3. 验证 RabbitMQ 运行状态

访问管理界面：http://localhost:15672
- 用户名：guest
- 密码：guest

或使用命令行检查：

```bash
# 检查 RabbitMQ 是否就绪
docker exec rabbitmq-test rabbitmqctl status
```

## 运行集成测试

### 运行所有集成测试

```bash
# 运行所有集成测试
go test -v -run TestIntegration

# 运行所有测试（包括单元测试和集成测试）
go test -v ./...
```

### 运行特定的集成测试

```bash
# 测试基本连接
go test -v -run TestIntegration_BasicConnection

# 测试发布和消费
go test -v -run TestIntegration_PublishAndConsume

# 测试批量发送
go test -v -run TestIntegration_BatchPublish

# 测试 Confirm 模式
go test -v -run TestIntegration_PublishWithConfirm

# 测试追踪功能
go test -v -run TestIntegration_PublishWithTrace

# 测试优雅关闭
go test -v -run TestIntegration_GracefulShutdown

# 测试消费者管理
go test -v -run TestIntegration_ConsumerManagement

# 测试 QoS
go test -v -run TestIntegration_QoS
```

### 跳过集成测试

如果你想跳过集成测试（例如在 CI 环境中没有 RabbitMQ），可以使用 `-short` 标志：

```bash
go test -short ./...
```

## 测试覆盖的功能

集成测试覆盖以下功能：

1. **基本连接** - 测试客户端连接到 RabbitMQ
2. **发布和消费** - 测试消息的发送和接收
3. **批量发送** - 测试批量发送多条消息
4. **Confirm 模式** - 测试发布确认机制
5. **追踪功能** - 测试消息追踪和链路追踪
6. **优雅关闭** - 测试客户端的优雅关闭
7. **消费者管理** - 测试消费者的注册、注销和状态管理
8. **QoS 设置** - 测试服务质量控制

## 配置测试环境

如果你的 RabbitMQ 运行在不同的地址或端口，可以修改 `integration_test.go` 中的配置：

```go
var testConfig = conf.RabbitConf{
    Scheme:   "amqp",
    Host:     "localhost",  // 修改为你的 RabbitMQ 地址
    Port:     5672,         // 修改为你的 RabbitMQ 端口
    Username: "guest",      // 修改为你的用户名
    Password: "guest",      // 修改为你的密码
    VHost:    "/",
}
```

## 清理测试环境

测试完成后，可以停止并删除 RabbitMQ 容器：

```bash
# 停止容器
docker stop rabbitmq-test

# 删除容器
docker rm rabbitmq-test

# 或者一步完成
docker rm -f rabbitmq-test
```

## 故障排查

### 连接失败

如果测试失败并显示连接错误：

1. 确认 RabbitMQ 容器正在运行：
   ```bash
   docker ps | grep rabbitmq-test
   ```

2. 检查端口是否被占用：
   ```bash
   lsof -i :5672
   lsof -i :15672
   ```

3. 查看 RabbitMQ 日志：
   ```bash
   docker logs rabbitmq-test
   ```

### 测试超时

如果测试超时，可能是因为：
- RabbitMQ 启动较慢，等待几秒后重试
- 网络问题，检查 Docker 网络配置
- 资源不足，检查系统资源

## 持续集成

在 CI 环境中运行集成测试的示例：

```yaml
# GitHub Actions 示例
name: Integration Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      rabbitmq:
        image: rabbitmq:3-management
        ports:
          - 5672:5672
          - 15672:15672
        env:
          RABBITMQ_DEFAULT_USER: guest
          RABBITMQ_DEFAULT_PASS: guest
        options: >-
          --health-cmd "rabbitmqctl status"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run integration tests
        run: go test -v -run TestIntegration
```

## 贡献

如果你想添加新的集成测试，请：
1. 在 `integration_test.go` 中添加新的测试函数
2. 使用 `TestIntegration_` 前缀命名
3. 添加 `testing.Short()` 检查以支持跳过
4. 更新本文档的测试列表

