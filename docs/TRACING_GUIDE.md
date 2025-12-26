# Distributed Tracing Guide

This guide explains how to use the distributed tracing features in RabbitMQ-Go.

## Overview

The `tracing` package provides built-in support for distributed tracing, allowing you to track messages across multiple services and understand the complete request flow.

## Key Concepts

### TraceInfo Structure

```go
type TraceInfo struct {
    TraceID      string  // Unique identifier for the entire request chain
    SpanID       string  // Unique identifier for this operation
    ParentSpanID string  // Identifier of the parent operation
}
```

### Trace Flow

```
Client (generates Trace ID: abc, Span ID: 001)
  ↓
Service A (extracts Trace ID: abc, generates Span ID: 002, Parent: 001)
  ↓
Service B (extracts Trace ID: abc, generates Span ID: 003, Parent: 002)
  ↓
Service C (extracts Trace ID: abc, generates Span ID: 004, Parent: 003)
```

## Basic Usage

### 1. Sending Messages with Tracing

```go
import (
    "context"
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/tracing"
)

func sendMessage(rabbit *rabbitmq.RabbitMQ) {
    ctx := context.Background()
    
    // SendMessageWithTrace automatically injects trace information
    rabbit.SendMessageWithTrace(ctx, "my-exchange", "my-route", true, amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("Hello World"),
    })
}
```

### 2. Receiving Messages with Tracing

```go
type MyReceiver struct{}

func (r *MyReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
    // Trace information is automatically extracted and injected into ctx
    
    // Get trace ID
    traceID := tracing.GetTraceID(ctx)
    
    // Format log with trace information
    log.Println(tracing.FormatTraceLog(ctx, "Processing message"))
    
    // Your business logic here
    
    return nil
}

func (r *MyReceiver) ExceptionWithContext(ctx context.Context, key string, err error, msg amqp.Delivery) {
    log.Println(tracing.FormatTraceLog(ctx, "Error: "+err.Error()))
}
```

## Advanced Usage

### Manual Trace Generation

```go
import "github.com/wenpiner/rabbitmq-go/tracing"

// Generate a new trace ID
traceID := tracing.GenerateTraceID()

// Generate a new span ID
spanID := tracing.GenerateSpanID()

// Create trace info
trace := &tracing.TraceInfo{
    TraceID:      traceID,
    SpanID:       spanID,
    ParentSpanID: "", // Empty for root span
}
```

### Inject Trace into Context

```go
// Inject trace information into context
ctx := tracing.InjectToContext(context.Background(), trace)

// Later, extract trace information
extractedTrace := tracing.ExtractFromContext(ctx)
```

### Inject Trace into Headers

```go
headers := amqp.Table{}

// Inject trace information into message headers
tracing.InjectToHeaders(trace, headers)

// Send message with headers
rabbit.SendMessage(ctx, exchange, route, true, amqp.Publishing{
    Headers: headers,
    Body:    []byte("message"),
})
```

### Extract Trace from Headers

```go
// Extract trace information from message headers
trace := tracing.ExtractFromHeaders(msg.Headers)

if trace != nil {
    log.Printf("Trace ID: %s, Span ID: %s, Parent: %s",
        trace.TraceID, trace.SpanID, trace.ParentSpanID)
}
```

## Logging with Trace Information

### Format Trace Logs

```go
// FormatTraceLog creates a structured log message with trace information
log.Println(tracing.FormatTraceLog(ctx, "User login successful"))
// Output: [trace_id=abc123 span_id=def456] User login successful
```

### Get Individual Trace Fields

```go
traceID := tracing.GetTraceID(ctx)
spanID := tracing.GetSpanID(ctx)
parentSpanID := tracing.GetParentSpanID(ctx)

log.Printf("Processing request [trace=%s, span=%s, parent=%s]",
    traceID, spanID, parentSpanID)
```

## Complete Example

See [examples/11-basic-tracing/](../examples/11-basic-tracing/) and [examples/12-trace-propagation/](../examples/12-trace-propagation/) for complete working examples.

### Basic Tracing Example

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

type TracingReceiver struct{}

func (r *TracingReceiver) ReceiveWithContext(ctx context.Context, key string, msg amqp.Delivery) error {
    log.Println(tracing.FormatTraceLog(ctx, "Received message: "+string(msg.Body)))
    return nil
}

func (r *TracingReceiver) ExceptionWithContext(ctx context.Context, key string, err error, msg amqp.Delivery) {
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
    
    rabbit.Register("tracing-demo", conf.ConsumerConf{
        Exchange: conf.NewFanoutExchange("tracing-exchange"),
        Queue:    conf.NewQueue("tracing-queue"),
        Name:     "tracing-consumer",
    }, &TracingReceiver{})
    
    startCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    rabbit.StartWithContext(startCtx)
    
    // Send message with tracing
    ctx := context.Background()
    rabbit.SendMessageWithTrace(ctx, "tracing-exchange", "", true, amqp.Publishing{
        ContentType: "text/plain",
        Body:        []byte("Hello with tracing!"),
    })
    
    time.Sleep(2 * time.Second)
    
    stopCtx, _ := context.WithTimeout(context.Background(), 30*time.Second)
    rabbit.StopWithContext(stopCtx)
}
```

## Best Practices

1. **Always use SendMessageWithTrace** for new code to enable automatic trace propagation
2. **Use FormatTraceLog** for consistent log formatting across your application
3. **Integrate with log aggregation tools** (e.g., ELK, Splunk) to analyze trace data
4. **Use trace IDs for debugging** to track messages across multiple services
5. **Consider sampling** for high-volume applications to reduce overhead

## Integration with External Systems

### OpenTelemetry Integration (Future)

The tracing package is designed to be compatible with OpenTelemetry. Future versions may include direct integration.

### Custom Trace ID Format

You can generate custom trace IDs:

```go
// Use your own trace ID format
customTraceID := "your-custom-format"
trace := &tracing.TraceInfo{
    TraceID: customTraceID,
    SpanID:  tracing.GenerateSpanID(),
}
ctx := tracing.InjectToContext(context.Background(), trace)
```

## Performance Considerations

- Trace ID generation uses UUID v4, which is fast and collision-resistant
- Trace information is stored in context, with minimal memory overhead
- Header injection/extraction is lightweight and has negligible performance impact
- For high-throughput applications, consider implementing sampling strategies

## Troubleshooting

### Trace Information Not Propagating

1. Ensure you're using `SendMessageWithTrace` instead of `SendMessage`
2. Verify that your receiver implements `ReceiveWithContext`
3. Check that context is being passed through your application correctly

### Missing Trace IDs in Logs

1. Ensure you're using `FormatTraceLog` or `GetTraceID`
2. Verify that the context contains trace information
3. Check that trace extraction is working correctly

## See Also

- [Migration Guide](MIGRATION_GUIDE.md) - How to migrate to the new tracing features
- [Graceful Shutdown Guide](GRACEFUL_SHUTDOWN_GUIDE.md) - How to use graceful shutdown
- [Examples](../examples/) - Working examples for all features

