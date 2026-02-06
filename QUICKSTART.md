# RabbitMQ-Go 快速开始指南

本指南将帮助你在 5 分钟内开始使用 RabbitMQ-Go。

## 1. 安装 RabbitMQ

使用 Docker 快速启动 RabbitMQ：

```bash
docker run -d \
  --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management
```

验证 RabbitMQ 运行状态：
- 访问管理界面：http://localhost:15672 (guest/guest)
- 或运行：`docker ps | grep rabbitmq`

## 2. 安装 RabbitMQ-Go

```bash
go get github.com/wenpiner/rabbitmq-go/v2@latest
```

## 3. 创建你的第一个应用

创建 `main.go`：

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

func main() {
	// 1. 创建客户端
	client := rabbitmq.New(
		rabbitmq.WithConfig(conf.RabbitConf{
			Scheme:   "amqp",
			Host:     "localhost",
			Port:     5672,
			Username: "guest",
			Password: "guest",
			VHost:    "/",
		}),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	// 2. 连接
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatal("连接失败:", err)
	}
	defer client.Close()

	log.Println("✅ 已连接到 RabbitMQ")

	// 3. 创建消息处理器
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			log.Printf("📨 收到消息: %s", string(msg.Body()))
			return nil
		},
	)

	// 4. 注册消费者
	err := client.RegisterConsumer("my-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "hello-queue",
			Durable: false,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "hello-exchange",
			Type:         "fanout",
			Durable:      false,
		}),
		rabbitmq.WithAutoAck(true),
		rabbitmq.WithHandler(handler),
	)
	if err != nil {
		log.Fatal("注册消费者失败:", err)
	}

	log.Println("✅ 消费者已注册")

	// 5. 发送消息
	go func() {
		time.Sleep(1 * time.Second)
		for i := 1; i <= 5; i++ {
			err := client.Publish(ctx, "hello-exchange", "", amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte("Hello World!"),
			})
			if err != nil {
				log.Printf("❌ 发送失败: %v", err)
			} else {
				log.Printf("📤 已发送消息 #%d", i)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// 6. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	log.Println("🚀 应用运行中，按 Ctrl+C 退出...")
	<-sigChan
	
	log.Println("🛑 正在关闭...")
}
```

## 4. 运行应用

```bash
go run main.go
```

你应该看到类似的输出：

```
✅ 已连接到 RabbitMQ
✅ 消费者已注册
🚀 应用运行中，按 Ctrl+C 退出...
📤 已发送消息 #1
📨 收到消息: Hello World!
📤 已发送消息 #2
📨 收到消息: Hello World!
...
```

## 5. 下一步

恭喜！你已经成功运行了第一个 RabbitMQ-Go 应用。

### 探索更多功能

查看 [examples](./examples) 目录了解更多高级功能：

- **批量发送**: [02-batch-publish](./examples/02-batch-publish)
- **分布式追踪**: [03-tracing](./examples/03-tracing)
- **重试策略**: [04-retry-strategy](./examples/04-retry-strategy)
- **并发处理**: [05-concurrency](./examples/05-concurrency)
- **优雅关闭**: [06-graceful-shutdown](./examples/06-graceful-shutdown)

### 阅读文档

- [完整 README](./README.md) - 所有功能的详细文档
- [Examples README](./examples/README.md) - 示例详细说明
- [集成测试指南](./INTEGRATION_TEST.md) - 如何运行测试

## 常见问题

### 连接失败？

确保 RabbitMQ 正在运行：
```bash
docker ps | grep rabbitmq
```

### 消息未被消费？

检查队列绑定和 routing key 是否正确。访问 RabbitMQ 管理界面查看队列状态。

### 需要帮助？

- 查看 [完整文档](./README.md)
- 运行 [示例程序](./examples)
- 查看 [测试代码](./integration_test.go)

