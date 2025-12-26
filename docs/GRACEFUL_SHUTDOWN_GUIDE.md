# Graceful Shutdown Guide

This guide explains how to use the graceful shutdown features in RabbitMQ-Go to ensure no messages are lost during application shutdown.

## Overview

Graceful shutdown ensures that:
1. No new messages are accepted after shutdown is initiated
2. All in-flight messages are processed to completion
3. All connections are properly closed
4. No messages are lost during the shutdown process

## Basic Usage

### Starting with Context

```go
import (
    "context"
    "time"
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/conf"
)

func main() {
    rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
        Scheme:   "amqp",
        Username: "guest",
        Password: "guest",
        Host:     "localhost",
        Port:     5672,
        VHost:    "/",
    })
    
    // Register consumers
    rabbit.Register("my-key", conf.ConsumerConf{
        Exchange: conf.NewFanoutExchange("my-exchange"),
        Queue:    conf.NewQueue("my-queue"),
        Name:     "my-consumer",
    }, &MyReceiver{})
    
    // Start with timeout control
    startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := rabbit.StartWithContext(startCtx); err != nil {
        log.Fatal("Failed to start:", err)
    }
    
    // Your application logic here
    
    // Graceful shutdown with timeout
    stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    rabbit.StopWithContext(stopCtx)
}
```

## Shutdown Process

### What Happens During Shutdown

1. **Cancel Global Context**: Signals all goroutines to stop
2. **Stop Accepting New Messages**: Consumers stop receiving new messages
3. **Wait for In-Flight Messages**: Waits for all currently processing messages to complete
4. **Close Connections**: Properly closes all RabbitMQ connections
5. **Return**: Returns when all cleanup is complete or timeout is reached

### Shutdown Flow Diagram

```
StopWithContext called
  ↓
Cancel global context
  ↓
Stop receiving new messages
  ↓
Wait for all handlers to complete (with timeout)
  ↓
Close all connections
  ↓
Return
```

## Timeout Control

### Startup Timeout

Control how long to wait for the connection to be established:

```go
// Wait up to 30 seconds for startup
startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := rabbit.StartWithContext(startCtx); err != nil {
    if err == context.DeadlineExceeded {
        log.Fatal("Startup timeout exceeded")
    }
    log.Fatal("Failed to start:", err)
}
```

### Shutdown Timeout

Control how long to wait for graceful shutdown:

```go
// Wait up to 30 seconds for shutdown
stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

rabbit.StopWithContext(stopCtx)
// If timeout is exceeded, shutdown will be forced
```

### Handler Timeout

Control how long each message handler can run:

```go
rabbit.Register("my-key", conf.ConsumerConf{
    Exchange:       conf.NewFanoutExchange("my-exchange"),
    Queue:          conf.NewQueue("my-queue"),
    Name:           "my-consumer",
    HandlerTimeout: 10 * time.Second, // Each message must complete within 10 seconds
}, &MyReceiver{})
```

## Signal Handling

### Graceful Shutdown on OS Signals

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/conf"
)

func main() {
    rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{...})
    rabbit.Register("my-key", conf.ConsumerConf{...}, &MyReceiver{})
    
    startCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    if err := rabbit.StartWithContext(startCtx); err != nil {
        log.Fatal("Failed to start:", err)
    }
    
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Wait for signal
    <-sigChan
    log.Println("Shutdown signal received, starting graceful shutdown...")
    
    // Graceful shutdown
    stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    rabbit.StopWithContext(stopCtx)
    log.Println("Shutdown complete")
}
```

## Advanced Usage

### Custom Shutdown Logic

```go
type MyReceiver struct {
    shutdownChan chan struct{}
}

func (r *MyReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
    select {
    case <-ctx.Done():
        // Context cancelled, shutdown in progress
        log.Println("Shutdown in progress, rejecting message")
        return ctx.Err()
    case <-r.shutdownChan:
        // Custom shutdown signal
        log.Println("Custom shutdown signal received")
        return nil
    default:
        // Normal processing
        return r.processMessage(msg)
    }
}
```

### Monitoring Shutdown Progress

```go
func gracefulShutdown(rabbit *rabbitmq.RabbitMQ) {
    stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    done := make(chan struct{})
    go func() {
        rabbit.StopWithContext(stopCtx)
        close(done)
    }()
    
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-done:
            log.Println("Shutdown complete")
            return
        case <-ticker.C:
            log.Println("Still shutting down...")
        case <-stopCtx.Done():
            log.Println("Shutdown timeout exceeded")
            return
        }
    }
}
```

## Best Practices

1. **Always use StartWithContext and StopWithContext** for new applications
2. **Set reasonable timeouts** based on your message processing time
3. **Handle OS signals** (SIGINT, SIGTERM) for graceful shutdown
4. **Monitor shutdown progress** in production environments
5. **Test shutdown behavior** under load to ensure messages aren't lost
6. **Use handler timeouts** to prevent individual messages from blocking shutdown

## Complete Example

See [examples/09-graceful-shutdown/](../examples/09-graceful-shutdown/) for a complete working example.

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
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/conf"
)

type GracefulReceiver struct{}

func (r *GracefulReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
    log.Printf("Processing message: %s", string(msg.Body))
    
    // Simulate work
    select {
    case <-time.After(2 * time.Second):
        log.Println("Message processed successfully")
        return nil
    case <-ctx.Done():
        log.Println("Context cancelled, stopping processing")
        return ctx.Err()
    }
}

func (r *GracefulReceiver) ExceptionWithContext(ctx context.Context, key string, err error, msg amqp.Delivery) {
    log.Printf("Error processing message: %v", err)
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
    
    rabbit.Register("graceful-demo", conf.ConsumerConf{
        Exchange:       conf.NewFanoutExchange("graceful-exchange"),
        Queue:          conf.NewQueue("graceful-queue"),
        Name:           "graceful-consumer",
        HandlerTimeout: 10 * time.Second,
    }, &GracefulReceiver{})
    
    // Start with timeout
    startCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    if err := rabbit.StartWithContext(startCtx); err != nil {
        log.Fatal("Failed to start:", err)
    }
    log.Println("RabbitMQ started successfully")
    
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    log.Println("Waiting for shutdown signal...")
    <-sigChan
    
    log.Println("Shutdown signal received, starting graceful shutdown...")
    stopCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    rabbit.StopWithContext(stopCtx)
    
    log.Println("Shutdown complete")
}
```

## Troubleshooting

### Shutdown Takes Too Long

1. Check if handlers are respecting context cancellation
2. Increase shutdown timeout if needed
3. Review handler timeout settings
4. Check for blocking operations in handlers

### Messages Lost During Shutdown

1. Ensure you're using `StopWithContext` instead of `Stop`
2. Verify that shutdown timeout is sufficient
3. Check that handlers complete successfully
4. Review error handling in exception handlers

## See Also

- [Migration Guide](MIGRATION_GUIDE.md) - How to migrate to graceful shutdown
- [Tracing Guide](TRACING_GUIDE.md) - How to use distributed tracing
- [Examples](../examples/) - Working examples for all features

