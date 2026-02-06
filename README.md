# rabbitmq-go

A production-ready RabbitMQ client for Go built on a robust state machine architecture, featuring automatic reconnection, distributed tracing, and comprehensive message handling capabilities.

[![Go Reference](https://pkg.go.dev/badge/github.com/wenpiner/rabbitmq-go.svg)](https://pkg.go.dev/github.com/wenpiner/rabbitmq-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/wenpiner/rabbitmq-go)](https://goreportcard.com/report/github.com/wenpiner/rabbitmq-go)

> 🚀 **New to RabbitMQ-Go?** Check out the [Quick Start Guide](./QUICKSTART.md) to get up and running in 5 minutes!

## Features

- ✅ **State Machine Architecture** - Robust connection lifecycle management with clear state transitions
- ✅ **Automatic Reconnection** - Intelligent reconnection with configurable retry strategies
- ✅ **Context Support** - Full context.Context support for timeout control and cancellation
- ✅ **Distributed Tracing** - Built-in trace ID propagation for observability
- ✅ **Graceful Shutdown** - Ensures no message loss during shutdown
- ✅ **Flexible Retry Strategies** - Exponential backoff, linear backoff, or custom strategies
- ✅ **Middleware Support** - Chain handlers with logging, recovery, and custom middlewares
- ✅ **Publisher API** - Batch publishing, confirm mode, and transaction support
- ✅ **Concurrent Processing** - Configurable concurrency for message handlers
- ✅ **Type Safe** - Leverages Go's type system for safer code
- ✅ **Production Ready** - Battle-tested in production environments

## Key Features

### 🔄 State Machine Architecture
- **Clear State Transitions**: Disconnected → Connecting → Connected → Reconnecting → Closed
- **Thread-Safe**: Concurrent state access with proper locking
- **Event-Driven**: State changes trigger automatic actions (consumer restart, reconnection, etc.)
- **State Listeners**: Subscribe to state changes for monitoring and logging

### 🔌 Automatic Reconnection
- **Intelligent Retry**: Configurable reconnection intervals and max attempts
- **Consumer Recovery**: Automatically restarts all consumers after reconnection
- **Connection Monitoring**: Detects connection failures and triggers reconnection
- **Graceful Degradation**: Handles partial failures without crashing

### 🎯 Context Support
- **Full Context Integration**: All operations support context.Context
- **Timeout Control**: Configurable timeouts for connections, handlers, and shutdown
- **Cancellation**: Proper cancellation propagation throughout the system
- **Deadline Awareness**: Respects context deadlines in all async operations

### 🔍 Distributed Tracing
- **Built-in Tracing**: Automatic trace ID generation and propagation
- **Message-Level Tracing**: Each message carries trace and span IDs
- **Context Integration**: Trace information flows through context
- **Structured Logging**: Trace IDs included in all log messages

### 🎭 Middleware Support
- **Handler Chain**: Compose multiple middlewares for message processing
- **Built-in Middlewares**: Logging, panic recovery included
- **Custom Middlewares**: Easy to implement custom middleware logic
- **Execution Order**: Predictable middleware execution order

### 📤 Publisher API
- **Basic Publishing**: Simple message publishing with optional tracing
- **Batch Publishing**: High-performance batch message sending
- **Confirm Mode**: Reliable publishing with broker acknowledgments
- **Transaction Mode**: Atomic message publishing with commit/rollback

### 🔧 Flexible Configuration
- **Functional Options**: Clean, extensible configuration pattern
- **Consumer Options**: Per-consumer configuration (timeout, concurrency, retry)
- **Client Options**: Global settings (reconnection, heartbeat, logging)
- **Retry Strategies**: Pluggable retry strategies (exponential, linear, custom)

## Install

```bash
go get github.com/wenpiner/rabbitmq-go/v2@latest
```

## Quick Start

### Basic Usage

```go
package main

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go/v2"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

func main() {
	// Create client with options
	client := rabbitmq.New(
		rabbitmq.WithConfig(conf.RabbitConf{
			Scheme:   "amqp",
			Username: "guest",
			Password: "guest",
			Host:     "localhost",
			Port:     5672,
			VHost:    "/",
		}),
		rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
		rabbitmq.WithReconnectInterval(5*time.Second),
		rabbitmq.WithMaxReconnectAttempts(10),
	)

	// Connect to RabbitMQ
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer client.Close()

	// Create message handler
	handler := rabbitmq.NewFuncHandler(
		func(ctx context.Context, msg *rabbitmq.Message) error {
			log.Printf("Received: %s", string(msg.Body()))
			return nil
		},
		rabbitmq.WithErrorHandler(func(ctx context.Context, msg *rabbitmq.Message, err error) {
			log.Printf("Error processing message: %v", err)
		}),
	)

	// Register consumer
	err := client.RegisterConsumer("my-consumer",
		rabbitmq.WithQueue(conf.QueueConf{
			Name:    "my-queue",
			Durable: true,
		}),
		rabbitmq.WithExchange(conf.ExchangeConf{
			ExchangeName: "my-exchange",
			Type:         "fanout",
			Durable:      true,
		}),
		rabbitmq.WithAutoAck(false),
		rabbitmq.WithHandler(handler),
		rabbitmq.WithConcurrency(5),
		rabbitmq.WithHandlerTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to register consumer:", err)
	}

	// Publish a message
	err = client.Publish(ctx, "my-exchange", "", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Hello, RabbitMQ!"),
	})
	if err != nil {
		log.Printf("Failed to publish: %v", err)
	}

	// Wait for messages...
	select {}
}
```

> 💡 **More Examples**: Check out the [examples](./examples) directory for more comprehensive examples including batch publishing, tracing, retry strategies, concurrency, and graceful shutdown.

### Using Middleware

```go
// Create handler with middleware chain
handler := rabbitmq.NewChainHandler(
	rabbitmq.NewFuncHandler(func(ctx context.Context, msg *rabbitmq.Message) error {
		// Your business logic
		log.Printf("Processing: %s", string(msg.Body()))
		return nil
	}),
	rabbitmq.LoggingMiddleware(logger.NewDefaultLogger(logger.LevelInfo)),
	rabbitmq.RecoveryMiddleware(logger.NewDefaultLogger(logger.LevelError)),
)

// Register consumer with middleware
client.RegisterConsumer("my-consumer",
	rabbitmq.WithQueue(conf.QueueConf{
		Name:    "my-queue",
		Durable: true,
	}),
	rabbitmq.WithExchange(conf.ExchangeConf{
		ExchangeName: "my-exchange",
		Type:         "fanout",
		Durable:      true,
	}),
	rabbitmq.WithHandler(handler),
)
```

### Custom Middleware

```go
// Create custom middleware
func MetricsMiddleware(next rabbitmq.HandlerFunc) rabbitmq.HandlerFunc {
	return func(ctx context.Context, msg *rabbitmq.Message) error {
		start := time.Now()
		err := next(ctx, msg)
		duration := time.Since(start)

		// Record metrics
		log.Printf("Message processed in %v", duration)
		return err
	}
}

// Use custom middleware
handler := rabbitmq.NewChainHandler(
	baseHandler,
	MetricsMiddleware,
	rabbitmq.LoggingMiddleware(logger),
)
```



## State Machine

The client uses a state machine to manage connection lifecycle:

```
Disconnected ──Connect──> Connecting ──Success──> Connected
     ↑                         │                      │
     │                      Fail                  Disconnect
     │                         │                      │
     └─────────────────────────┴──> Reconnecting <────┘
                                           │
                                        Success
                                           │
                                           ↓
                                      Connected

Any State ──Close──> Closed (Terminal)
```

### State Monitoring

```go
// Check current state
state := client.State()
log.Printf("Current state: %s", state)

// Check if connected
if client.IsConnected() {
    log.Println("Client is connected")
}
```

## Publisher API

### Basic Publishing

```go
// Simple publish
err := client.Publish(ctx, "my-exchange", "routing.key", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Hello World"),
})

// Publish with tracing
err := client.PublishWithTrace(ctx, "my-exchange", "routing.key", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Traced message"),
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
err := client.PublishBatch(ctx, "my-exchange", "routing.key", messages)

// Batch publish with confirm (reliable)
err := client.PublishBatchWithConfirm(ctx, "my-exchange", "routing.key", messages)
```

### Publisher Confirm Mode

```go
// Use confirm mode for reliable publishing
err := client.PublishWithConfirm(ctx, "my-exchange", "routing.key", amqp.Publishing{
    ContentType: "text/plain",
    Body:        []byte("Important message"),
})
```

### Transaction Mode

```go
// Use transaction for atomic publishing
err := client.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
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

| Method | Performance | Reliability | Atomicity | Use Case |
|--------|-------------|-------------|-----------|----------|
| `Publish` | High | Low | No | Simple messages |
| `PublishBatch` | Very High | Low | No | High throughput |
| `PublishWithConfirm` | Medium | High | No | Reliable delivery |
| `PublishBatchWithConfirm` | Medium | High | No | Reliable batch |
| `WithPublisherTx` | Low | High | Yes | Atomic operations |

## Retry Strategies

### Exponential Backoff

```go
// Register consumer with exponential retry
client.RegisterConsumer("my-consumer",
    rabbitmq.WithQueue(conf.QueueConf{
        Name:    "my-queue",
        Durable: true,
    }),
    rabbitmq.WithExchange(conf.ExchangeConf{
        ExchangeName: "my-exchange",
        Type:         "fanout",
        Durable:      true,
    }),
    rabbitmq.WithHandler(handler),
    rabbitmq.WithRetryStrategy(&conf.ExponentialRetry{
        MaxRetries:   5,
        InitialDelay: time.Second,
        Multiplier:   2.0,
        MaxDelay:     time.Minute,
        Jitter:       true,
    }),
)
```

**Retry timeline:** ~1s → ~2s → ~4s → ~8s → ~16s (with jitter)

### Linear Backoff

```go
// Linear retry with fixed interval
rabbitmq.WithRetryStrategy(&conf.LinearRetry{
    MaxRetries:   3,
    InitialDelay: 2 * time.Second,
})
```

**Retry timeline:** 2s → 2s → 2s

### No Retry

```go
// Disable retry
rabbitmq.WithRetryStrategy(&conf.NoRetry{})
```

## Reconnection

### Automatic Reconnection

```go
// Configure reconnection behavior
client := rabbitmq.New(
    rabbitmq.WithConfig(conf.RabbitConf{...}),
    rabbitmq.WithAutoReconnect(true),
    rabbitmq.WithReconnectInterval(5*time.Second),
    rabbitmq.WithMaxReconnectAttempts(10), // 0 = infinite
)
```

When connection is lost:
1. Client transitions to `Reconnecting` state
2. Waits for `ReconnectInterval`
3. Attempts to reconnect
4. On success: transitions to `Connected` and restarts all consumers
5. On failure: retries up to `MaxReconnectAttempts`

### Connection Timeout

```go
// Set connection timeout
client := rabbitmq.New(
    rabbitmq.WithConfig(conf.RabbitConf{...}),
    rabbitmq.WithConnectTimeout(30*time.Second),
)
```

## Logging

### Using Default Logger

```go
import "github.com/wenpiner/rabbitmq-go/v2/logger"

// Create client with logger
client := rabbitmq.New(
    rabbitmq.WithConfig(conf.RabbitConf{...}),
    rabbitmq.WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
)
```

### Disable Logging

```go
// Use NoopLogger for zero-overhead logging
client := rabbitmq.New(
    rabbitmq.WithLogger(logger.NewNoopLogger()),
)
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

## Concurrency

### Concurrent Message Processing

```go
// Process messages concurrently
client.RegisterConsumer("my-consumer",
    rabbitmq.WithQueue(conf.QueueConf{
        Name:    "my-queue",
        Durable: true,
    }),
    rabbitmq.WithExchange(conf.ExchangeConf{
        ExchangeName: "my-exchange",
        Type:         "fanout",
        Durable:      true,
    }),
    rabbitmq.WithHandler(handler),
    rabbitmq.WithConcurrency(10), // Process 10 messages concurrently
)
```

### Handler Timeout

```go
// Set timeout for message handlers
client.RegisterConsumer("my-consumer",
    rabbitmq.WithQueue(conf.QueueConf{Name: "my-queue", Durable: true}),
    rabbitmq.WithExchange(conf.ExchangeConf{ExchangeName: "my-exchange", Type: "fanout", Durable: true}),
    rabbitmq.WithHandler(handler),
    rabbitmq.WithHandlerTimeout(30*time.Second), // Handler must complete within 30s
)
```

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                        Client                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ State Machine│  │  Connection  │  │   Publisher  │  │
│  │              │  │   Manager    │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Consumer Registry                      │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │  │
│  │  │Consumer 1│  │Consumer 2│  │Consumer N│  ...  │  │
│  │  └──────────┘  └──────────┘  └──────────┘       │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘

Each Consumer:
┌─────────────────────────────────────────┐
│           Consumer                      │
│  ┌──────────────┐  ┌──────────────┐    │
│  │    State     │  │   Channel    │    │
│  └──────────────┘  └──────────────┘    │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │      Handler Chain               │  │
│  │  Middleware 1 → Middleware 2 →   │  │
│  │  ... → Base Handler              │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

## Examples

The library includes comprehensive examples demonstrating various features:

| Example | Description | Key Features |
|---------|-------------|--------------|
| [01-basic](./examples/01-basic) | Basic publish and consume | Client setup, consumer registration, message publishing |
| [02-batch-publish](./examples/02-batch-publish) | Batch message publishing | High-performance batch sending, confirm mode |
| [03-tracing](./examples/03-tracing) | Distributed tracing | Trace ID generation, propagation, extraction |
| [04-retry-strategy](./examples/04-retry-strategy) | Retry strategies | Exponential backoff, error handling |
| [05-concurrency](./examples/05-concurrency) | Concurrent processing | Concurrency control, QoS, performance metrics |
| [06-graceful-shutdown](./examples/06-graceful-shutdown) | Graceful shutdown | Clean shutdown, message completion guarantee |

**Running Examples:**

```bash
# Run a specific example
cd examples/01-basic
go run main.go

# Or build and run
go build -o example
./example
```

See [examples/README.md](./examples/README.md) for detailed documentation.

## Testing

The library includes comprehensive unit and integration tests:

```bash
# Run all unit tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test
go test -v -run TestStateMachine

# Run integration tests (requires RabbitMQ)
go test -v -run TestIntegration
```

**Test Coverage:**
- State machine: 14 tests
- Message handlers: 14 tests
- Configuration options: 18 tests
- Integration tests: 8 tests
- **Total: 54+ tests**

See [INTEGRATION_TEST.md](./INTEGRATION_TEST.md) for integration testing guide.

## Documentation

### API Documentation
- See [GoDoc](https://pkg.go.dev/github.com/wenpiner/rabbitmq-go/v2) for complete API documentation

### Release Information
- [CHANGELOG](CHANGELOG.md) - Complete changelog and version history

## License

MIT
