# Deprecation Warnings Example

This example demonstrates the deprecation warning mechanism in the RabbitMQ-Go library.

## Features

### Deprecated APIs

The following APIs are marked as deprecated but still functional:

1. **Channel()** - Create channel
   - ⚠️ Reason: Returning channels can lead to resource leaks
   - ✅ Alternative: Use `WithPublisher()`

2. **ChannelByName()** - Create named channel
   - ⚠️ Reason: Returning channels can lead to resource leaks
   - ✅ Alternative: Use `WithPublisher()`

3. **SendMessage()** - Send message
   - ⚠️ Reason: Returns channel but rarely used
   - ✅ Alternative: Use `Publish()`

4. **SendMessageClose()** - Send message and close channel
   - ⚠️ Reason: Unclear API naming
   - ✅ Alternative: Use `Publish()`

5. **SendMessageWithTrace()** - Send message with tracing
   - ⚠️ Reason: Returns channel but rarely used
   - ✅ Alternative: Use `PublishWithTrace()`

### Warning Mechanism

- ✅ Each deprecated API warns only once
- ✅ Warning messages include alternatives and migration guide links
- ✅ Does not affect program functionality
- ✅ Allows gradual migration without breaking existing code

## Running the Example

### Prerequisites

Ensure RabbitMQ is running:

```bash
# Start RabbitMQ using Docker
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3-management
```

### Run the Program

```bash
cd examples/18-deprecation-warnings
go run main.go
```

### Expected Output

```
=== Deprecation Warnings Example ===

1. Using deprecated Channel() API
   First call will show warning...
DEPRECATION WARNING: Channel() is deprecated and will be removed in a future version.
Please use WithPublisher() instead.
Migration guide: https://github.com/wenpiner/rabbitmq-go/blob/main/docs/PUBLISHER_MIGRATION_GUIDE.md
   ✓ Channel created successfully (with deprecation warning)

2. Using Channel() API again
   Second call won't show warning (each API warns only once)...
   ✓ Channel created successfully (no warning)

...
```

## Migration Guide

### 1. Identify Deprecated APIs

Run your program and check logs for deprecation warnings.

### 2. Review Migration Guide

Visit [Publisher API Migration Guide](../../docs/PUBLISHER_MIGRATION_GUIDE.md) for detailed migration steps.

### 3. Gradual Migration

No need to migrate all code at once. Replace gradually:

#### Old Code

```go
// Deprecated API
channel, err := rabbit.Channel()
if err != nil {
    return err
}
defer channel.Close()

err = channel.PublishWithContext(ctx, exchange, route, false, false, msg)
```

#### New Code

```go
// Recommended API
err := rabbit.Publish(ctx, exchange, route, msg)
```

### 4. Verify Functionality

Run tests after migration to ensure everything works correctly.

## FAQ

### Q: When will deprecated APIs be removed?

A: Deprecated APIs will be retained long-term for backward compatibility, but are not recommended. We'll consider removal in future major versions (e.g., v5.0.0).

### Q: Must I migrate immediately?

A: Not required, but strongly recommended. Deprecated APIs will continue to work but will print warnings. New APIs are safer and more performant.

### Q: How to disable deprecation warnings?

A: Migrate to new APIs to eliminate warnings. Each deprecated API warns only once.

### Q: Are new APIs really more performant?

A: For batch sending, performance improvement is significant (10-100x). For single messages, performance is similar.

## Related Documentation

- [Publisher API Migration Guide](../../docs/PUBLISHER_MIGRATION_GUIDE.md)
- [Publisher API Quick Reference](../../docs/PUBLISHER_API_QUICK_REFERENCE.md)

## Technical Details

### Warning Mechanism Implementation

```go
// Each RabbitMQ instance maintains a set of warned APIs
type RabbitMQ struct {
    deprecationWarnings map[string]bool
    deprecationMu       sync.Mutex
    // ...
}

// Print warning only on first call
func (g *RabbitMQ) warnDeprecation(oldAPI, newAPI, migrationURL string) {
    g.deprecationMu.Lock()
    defer g.deprecationMu.Unlock()

    if g.deprecationWarnings[oldAPI] {
        return // Already warned
    }

    log.Printf("DEPRECATION WARNING: %s is deprecated...", oldAPI)
    g.deprecationWarnings[oldAPI] = true
}
```

### Backward Compatibility

All deprecated APIs internally call new APIs to ensure consistent functionality:

```go
// Deprecated API
func (g *RabbitMQ) SendMessage(...) (*amqp.Channel, error) {
    g.warnDeprecation("SendMessage()", "Publish()", "...")
    
    // Use new API internally
    err := g.Publish(ctx, exchange, route, msg)
    return nil, err
}
```

