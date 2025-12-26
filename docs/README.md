# RabbitMQ-Go Documentation

Welcome to the RabbitMQ-Go documentation. This guide will help you get started with the library and make the most of its features.

## 📚 Documentation Index

### Getting Started
- [Main README](../README.md) - Project overview and quick start
- [Migration Guide](MIGRATION_GUIDE.md) - Upgrade from older versions
- [Examples](../examples/) - Working code examples

### Feature Guides
- [Distributed Tracing Guide](TRACING_GUIDE.md) - How to use distributed tracing
- [Graceful Shutdown Guide](GRACEFUL_SHUTDOWN_GUIDE.md) - How to implement graceful shutdown
- [Context Best Practices](CONTEXT_BEST_PRACTICES.md) - Best practices for using context

### Reference
- [Changelog](../CHANGELOG.md) - Version history and changes

---

## 🚀 Quick Start

### For New Projects (Recommended)

Use the new context-aware API with full tracing support:

```go
import (
    "context"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/conf"
    "github.com/wenpiner/rabbitmq-go/tracing"
)

type MyReceiver struct{}

func (r *MyReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
    log.Println(tracing.FormatTraceLog(ctx, "Processing message"))
    // Your business logic here
    return nil
}

func (r *MyReceiver) ExceptionWithContext(ctx context.Context, key string, err error, msg amqp.Delivery) {
    log.Println(tracing.FormatTraceLog(ctx, "Error: "+err.Error()))
}

func main() {
    rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
        Scheme:   "amqp",
        Username: "guest",
        Password: "guest",
        Host:     "localhost",
        Port:     5672,
        VHost:    "/",
    })

    rabbit.Register("my-key", conf.ConsumerConf{
        Exchange:       conf.NewFanoutExchange("my-exchange"),
        Queue:          conf.NewQueue("my-queue"),
        Name:           "my-consumer",
        HandlerTimeout: 30 * time.Second,
    }, &MyReceiver{})

    // Start with timeout control
    startCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    rabbit.StartWithContext(startCtx)

    // Send message with tracing
    ctx := context.Background()
    rabbit.SendMessageWithTrace(ctx, "my-exchange", "", true, amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("Hello World"),
    })

    // Graceful shutdown
    stopCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    rabbit.StopWithContext(stopCtx)
}
```

### For Existing Projects (Legacy API)

Continue using the old API without any changes:

```go
type MyReceiver struct{}

func (r *MyReceiver) Receive(key string, msg amqp.Delivery) error {
    log.Println("Processing message")
    return nil
}

func (r *MyReceiver) Exception(key string, err error, msg amqp.Delivery) {
    log.Println("Error:", err)
}

func main() {
    rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{...})
    rabbit.Register("my-key", conf.ConsumerConf{...}, &MyReceiver{})
    rabbit.Start()
    defer rabbit.Stop()
}
```

---

## ✨ Features

- ✅ **Context Support** - Full context.Context support for timeout control and cancellation
- ✅ **Distributed Tracing** - Built-in trace ID propagation for observability
- ✅ **Graceful Shutdown** - Ensures no message loss during shutdown
- ✅ **Automatic Reconnection** - Handles connection failures gracefully
- ✅ **Flexible Retry** - Multiple retry strategies (exponential, linear, custom)
- ✅ **QoS Support** - Fine-grained control over message prefetching
- ✅ **Backward Compatible** - Seamless upgrade from older versions
- ✅ **Type Safe** - Leverages Go's type system for safer code
- ✅ **Production Ready** - Battle-tested in production environments

---

## 📖 Learn More

### By Use Case

**I want to add timeout control to my handlers**
→ See [Context Best Practices](CONTEXT_BEST_PRACTICES.md)

**I want to track messages across services**
→ See [Distributed Tracing Guide](TRACING_GUIDE.md)

**I want to shutdown without losing messages**
→ See [Graceful Shutdown Guide](GRACEFUL_SHUTDOWN_GUIDE.md)

**I want to upgrade from an older version**
→ See [Migration Guide](MIGRATION_GUIDE.md)

**I want to see working examples**
→ See [Examples Directory](../examples/)

### By Topic

- **Context Support**: [Context Best Practices](CONTEXT_BEST_PRACTICES.md)
- **Tracing**: [Tracing Guide](TRACING_GUIDE.md)
- **Shutdown**: [Graceful Shutdown Guide](GRACEFUL_SHUTDOWN_GUIDE.md)
- **Migration**: [Migration Guide](MIGRATION_GUIDE.md)
- **Changes**: [Changelog](../CHANGELOG.md)

---

## 🔗 External Resources

- [Go Context Documentation](https://pkg.go.dev/context)
- [Go Context Best Practices](https://go.dev/blog/context)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

---

## 💡 Contributing

We welcome contributions! Please see the main [README](../README.md) for contribution guidelines.

---

## 📞 Support

- **GitHub Issues**: [Submit an Issue](https://github.com/wenpiner/rabbitmq-go/issues)
- **Pull Requests**: [Submit a PR](https://github.com/wenpiner/rabbitmq-go/pulls)

---

**Last Updated**: 2025-12-26
**Version**: 4.0.0
