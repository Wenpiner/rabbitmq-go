# rabbitmq-go

A flexible and feature-rich RabbitMQ client for Go with built-in retry mechanisms, context support, and distributed tracing.

## Features

- ✅ **Context Support** - Full context.Context support for timeout control and cancellation
- ✅ **Distributed Tracing** - Built-in trace ID propagation for observability
- ✅ **Graceful Shutdown** - Ensures no message loss during shutdown
- ✅ **Automatic Reconnection** - Handles connection failures gracefully
- ✅ **Flexible Retry Strategies** - Exponential backoff, linear backoff, or custom strategies
- ✅ **QoS Support** - Fine-grained control over message prefetching
- ✅ **Publisher API** - Batch publishing, confirm mode, and transaction support
- ✅ **Backward Compatible** - Seamless upgrade from older versions
- ✅ **Type Safe** - Leverages Go's type system for safer code
- ✅ **Production Ready** - Battle-tested in production environments

## Key Features

### 🎯 Context Support
- `ReceiveWithContext` interface for context-aware message handlers
- Context-aware message sending methods
- Timeout control and cancellation support
- Backward compatible with existing `Receive` interface

### 🛡️ Graceful Shutdown
- `StartWithContext` and `StopWithContext` methods
- Ensures all in-flight messages are processed
- Configurable shutdown timeout
- WaitGroup tracking for all goroutines

### 🔍 Distributed Tracing
- Built-in `tracing` package
- Automatic trace ID generation and propagation
- Context and header-based trace information
- Structured logging with trace IDs
- `SendMessageWithTrace` for automatic trace propagation

### 📤 Publisher API
- **Batch Publishing**: High-performance batch message sending
- **Confirm Mode**: Reliable publishing with broker acknowledgments
- **Transaction Mode**: Atomic message publishing with commit/rollback
- **BatchPublisher Helper**: Simplified batch publishing with auto-flush
- Multiple flush modes: normal, confirm, and transaction

### ⚠️ Deprecation Warnings
- **Graceful Deprecation**: Old APIs still work but show warnings
- **One-time Warnings**: Each deprecated API warns only once per instance
- **Migration Guide**: Detailed guide for upgrading to new APIs
- **Backward Compatible**: No breaking changes, migrate at your own pace

### 📝 Structured Logging
- **Custom Logger Interface**: Pluggable logging system
- **Default Logger**: Built-in logger with level filtering
- **Noop Logger**: Zero-overhead logging for production
- **Structured Fields**: Type-safe logging with structured fields
- **Third-party Integration**: Easy integration with zap, logrus, etc.

## Install

```bash
go get github.com/wenpiner/rabbitmq-go@latest
```

## Quick Start

### New Context-Aware API (Recommended)

```go
package main

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// Implement ReceiveWithContext interface
type MyReceiver struct{}

func (r *MyReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
	// Use tracing for observability
	log.Println(tracing.FormatTraceLog(ctx, "Processing message"))

	// Your business logic here
	return nil
}

func (r *MyReceiver) ExceptionWithContext(ctx context.Context, key string, err error, msg amqp.Delivery) {
	log.Println(tracing.FormatTraceLog(ctx, "Error: "+err.Error()))
}

func main() {
	// Create RabbitMQ client
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// Register consumer with timeout
	rabbit.Register("my-key", conf.ConsumerConf{
		Exchange:       conf.NewFanoutExchange("my-exchange"),
		Queue:          conf.NewQueue("my-queue"),
		RouteKey:       "",
		Name:           "my-consumer",
		HandlerTimeout: 30 * time.Second, // Handler timeout
	}, &MyReceiver{})

	// Start with context (with startup timeout)
	startCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	if err := rabbit.StartWithContext(startCtx); err != nil {
		log.Fatal("Failed to start:", err)
	}

	// Send message with tracing
	ctx := context.Background()
	rabbit.SendMessageWithTrace(ctx, "my-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Hello World!"),
	})

	// Graceful shutdown
	stopCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	rabbit.StopWithContext(stopCtx)
}
```

### Legacy API (Still Supported)

```go
package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

// Implement Receive interface
type MyReceiver struct{}

func (r *MyReceiver) Receive(key string, msg amqp.Delivery) error {
	log.Println("Processing message")
	return nil
}

func (r *MyReceiver) Exception(key string, err error, msg amqp.Delivery) {
	log.Println("Error:", err)
}

func main() {
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	rabbit.Register("my-key", conf.ConsumerConf{
		Exchange:  conf.NewFanoutExchange("my-exchange"),
		Queue:     conf.NewQueue("my-queue"),
		RouteKey:  "",
		Name:      "my-consumer",
	}, &MyReceiver{})

	rabbit.Start()
	defer rabbit.Stop()

	// Send message
	rabbit.SendMessage(context.Background(), "my-exchange", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Hello World!"),
	})
}
```



## Publisher API

### Basic Publishing

```go
// Simple publish
err := rabbit.Publish(ctx, "my-exchange", "routing.key", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Hello World"),
})

// Publish with mandatory flag
err := rabbit.PublishMandatory(ctx, "my-exchange", "routing.key", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Important message"),
})
```

### Batch Publishing

```go
// Prepare messages
messages := []amqp.Publishing{
    {ContentType: "text/plain", Body: []byte("Message 1")},
    {ContentType: "text/plain", Body: []byte("Message 2")},
    {ContentType: "text/plain", Body: []byte("Message 3")},
}

// Batch publish (high performance)
err := rabbit.PublishBatch(ctx, "my-exchange", "routing.key", messages)

// Batch publish with confirm (reliable)
err := rabbit.PublishBatchWithConfirm(ctx, "my-exchange", "routing.key", messages)

// Batch publish with transaction (atomic)
err := rabbit.PublishBatchTx(ctx, "my-exchange", "routing.key", messages)
```

### BatchPublisher Helper

```go
// Create batch publisher with auto-flush
publisher := rabbit.NewBatchPublisher("my-exchange", "routing.key").
    SetBatchSize(100).
    SetAutoFlush(true)

defer publisher.Close(ctx)

// Add messages (auto-flushes every 100 messages)
for i := 0; i < 1000; i++ {
    msg := amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte(fmt.Sprintf("Message %d", i)),
    }
    publisher.Add(ctx, msg)
}
```

### Publisher Confirm Mode

```go
// Use confirm mode for reliable publishing
err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // Send multiple messages
    for i := 0; i < 10; i++ {
        msg := amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(fmt.Sprintf("Message %d", i)),
        }
        if err := ch.PublishWithContext(ctx, "my-exchange", "routing.key", false, false, msg); err != nil {
            return err
        }
    }
    return nil // Auto-wait for confirms
})
```

### Transaction Mode

```go
// Use transaction for atomic publishing
err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
    // Send multiple messages atomically
    for i := 0; i < 3; i++ {
        msg := amqp.Publishing{
            ContentType: "text/plain",
            Body:        []byte(fmt.Sprintf("Tx Message %d", i)),
        }
        if err := ch.PublishWithContext(ctx, "my-exchange", "routing.key", false, false, msg); err != nil {
            return err
        }
    }
    return nil // Auto-commit (or auto-rollback on error)
})
```

### Publisher API Comparison

| Feature | Publish | PublishBatch | PublishBatchWithConfirm | PublishBatchTx |
|---------|---------|--------------|------------------------|----------------|
| Performance | High | Very High | Medium | Low |
| Reliability | Low | Low | High | High |
| Atomicity | No | No | No | Yes |
| Use Case | Simple | High throughput | Reliable delivery | Atomic operations |

📖 **For detailed publisher documentation, see:**
- [examples/16-publisher-transaction/](./examples/16-publisher-transaction/) - Transaction mode examples
- [examples/17-batch-publisher/](./examples/17-batch-publisher/) - BatchPublisher examples

## Retry Mechanism

### Default Exponential Backoff

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry:    conf.NewRetryConf(), // Default: 5 retries, exponential backoff
}, receiver)
```

**Retry timeline:** ~1s → ~2s → ~4s → ~8s → ~16s

### Custom Retry Configuration

```go
// Exponential backoff with custom parameters
Retry: conf.NewExponentialRetryConf(10, 500*time.Millisecond, 1.5)
// 10 retries, 500ms initial delay, 1.5x multiplier

// Linear backoff
Retry: conf.NewLinearRetryConf(3, 2*time.Second)
// 3 retries, 2s interval

// Disable retry
Retry: conf.RetryConf{Enable: false}
```

### Advanced: Custom Retry Strategy

```go
type MyReceiver struct {}

func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    // Your business logic
    return nil
}

func (r *MyReceiver) Exception(key string, err error, message amqp.Delivery) {
    // Handle max retries exceeded
}

func (r *MyReceiver) GetRetryStrategy() conf.RetryStrategy {
    return conf.NewExponentialRetry(15, 100*time.Millisecond, 2.0, time.Hour, true)
}
```

## Logging

### Using Default Logger

```go
import "github.com/wenpiner/rabbitmq-go/logger"

// Create RabbitMQ with custom logger
rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{...})

// Set log level (Debug, Info, Warn, Error)
rabbit.SetLogger(logger.NewDefaultLogger(logger.LevelInfo))
```

### Disable Logging

```go
// Use NoopLogger for zero-overhead logging
rabbit.SetLogger(logger.NewNoopLogger())
```

### Custom Logger Integration

```go
// Implement the Logger interface
type MyLogger struct {
    zapLogger *zap.Logger
}

func (l *MyLogger) Debug(msg string, fields ...logger.Field) {
    l.zapLogger.Debug(msg, convertFields(fields)...)
}

func (l *MyLogger) Info(msg string, fields ...logger.Field) {
    l.zapLogger.Info(msg, convertFields(fields)...)
}

// ... implement Warn and Error

// Use your custom logger
rabbit.SetLogger(&MyLogger{zapLogger: zap.NewProduction()})
```

### Structured Logging

```go
// Logger supports structured fields
logger.Info("Message processed",
    logger.String("queue", "my-queue"),
    logger.Int("retry", 3),
    logger.Duration("elapsed", time.Second),
    logger.Error(err))
```

## QoS Configuration

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    Qos:      conf.NewQos(10), // Prefetch 10 messages
    AutoAck:  false,
}, receiver)
```

## Examples

See the [examples](examples/) directory for more usage examples:
- [01-basic-context](examples/01-basic-context/) - Basic context usage
- [02-timeout-control](examples/02-timeout-control/) - Timeout control
- [03-graceful-shutdown](examples/03-graceful-shutdown/) - Graceful shutdown
- [04-distributed-tracing](examples/04-distributed-tracing/) - Distributed tracing
- [05-retry-mechanisms](examples/05-retry-mechanisms/) - Retry mechanisms
- [06-basic-publisher](examples/06-basic-publisher/) - Basic publisher
- [07-batch-publisher](examples/07-batch-publisher/) - Batch publishing
- [08-publisher-confirm](examples/08-publisher-confirm/) - Publisher confirm
- [09-publisher-transaction](examples/09-publisher-transaction/) - Publisher transaction
- [10-batch-publisher-helper](examples/10-batch-publisher-helper/) - BatchPublisher helper
- [18-deprecation-warnings](examples/18-deprecation-warnings/) - Deprecation warnings
- [retry_example.go](examples/retry_example.go) - Comprehensive retry examples

## Documentation

### User Guides
- [Migration Guide](docs/MIGRATION_GUIDE.md) - Complete guide for upgrading to v4.0.0
- [Publisher Migration Guide](docs/PUBLISHER_MIGRATION_GUIDE.md) - Guide for migrating from old to new Publisher APIs
- [Publisher API Quick Reference](docs/PUBLISHER_API_QUICK_REFERENCE.md) - Quick reference for Publisher APIs
- [Tracing Guide](docs/TRACING_GUIDE.md) - Distributed tracing usage guide
- [Graceful Shutdown Guide](docs/GRACEFUL_SHUTDOWN_GUIDE.md) - Best practices for graceful shutdown
- [Context Best Practices](docs/CONTEXT_BEST_PRACTICES.md) - Context usage patterns

### Release Information
- [CHANGELOG](CHANGELOG.md) - Complete changelog
- [Release Notes v4.0.0](RELEASE_NOTES_v4.0.0.md) - Detailed release notes

## License

MIT
