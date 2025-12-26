# RabbitMQ-Go Examples

This directory contains various usage examples for RabbitMQ-Go. Each example is in its own subdirectory and can be run independently.

## Prerequisites

1. **Install RabbitMQ**
   ```bash
   # Using Docker (recommended)
   docker run -d --name rabbitmq \
     -p 5672:5672 \
     -p 15672:15672 \
     rabbitmq:3-management
   ```

2. **Access Management UI** (optional)
   - URL: http://localhost:15672
   - Username: guest
   - Password: guest

## Example List

### Context Support

| Example | Description | Difficulty |
|---------|-------------|------------|
| [01-context-basic](01-context-basic/) | Basic context usage with `ReceiveWithContext` | ⭐ |
| [02-legacy-compat](02-legacy-compat/) | Legacy compatibility demonstration | ⭐ |
| [03-timeout](03-timeout/) | Timeout control and handling | ⭐⭐ |
| [05-phase1-demo](05-phase1-demo/) | Phase 1 feature demonstration | ⭐⭐ |
| [06-simple-demo](06-simple-demo/) | Simple usage demo | ⭐ |
| [07-send-with-context](07-send-with-context/) | Sending messages with context | ⭐⭐ |
| [08-cascade-timeout](08-cascade-timeout/) | Cascading timeout control | ⭐⭐⭐ |

### Retry Mechanisms

| Example | Description | Difficulty |
|---------|-------------|------------|
| [04-retry](04-retry/) | Retry strategies and configuration | ⭐⭐ |

### Graceful Shutdown

| Example | Description | Difficulty |
|---------|-------------|------------|
| [09-graceful-shutdown](09-graceful-shutdown/) | Graceful shutdown demonstration | ⭐⭐ |
| [10-start-timeout](10-start-timeout/) | Starting with timeout control | ⭐⭐ |

### Distributed Tracing

| Example | Description | Difficulty |
|---------|-------------|------------|
| [11-basic-tracing](11-basic-tracing/) | Basic distributed tracing | ⭐⭐ |
| [12-trace-propagation](12-trace-propagation/) | Trace ID propagation across services | ⭐⭐⭐ |

### Publisher API

| Example | Description | Difficulty |
|---------|-------------|------------|
| [13-publisher-basic](13-publisher-basic/) | Basic publishing methods | ⭐ |
| [14-publisher-tracing](14-publisher-tracing/) | Publishing with tracing | ⭐⭐ |
| [15-publisher-confirm](15-publisher-confirm/) | Publisher confirm mode | ⭐⭐⭐ |
| [16-publisher-transaction](16-publisher-transaction/) | Publisher transaction mode | ⭐⭐⭐ |
| [17-batch-publisher](17-batch-publisher/) | Batch publishing with BatchPublisher helper | ⭐⭐ |

### Deprecation Warnings

| Example | Description | Difficulty |
|---------|-------------|------------|
| [18-deprecation-warnings](18-deprecation-warnings/) | Deprecation warning demonstration | ⭐ |

## Running Examples

Each example can be run independently:

```bash
# Navigate to example directory and run
cd examples/01-context-basic
go run main.go

# Or run directly
go run examples/01-context-basic/main.go
```

## Configuration

Most examples use the following default RabbitMQ configuration:

```go
conf.RabbitConf{
    Scheme:   "amqp",
    Username: "guest",
    Password: "guest",
    Host:     "127.0.0.1",
    Port:     5672,
    VHost:    "/",
}
```

Modify the configuration in each example's `main.go` file to match your RabbitMQ setup.

## Example Categories

### Beginner (⭐)
Start with these examples if you're new to RabbitMQ-Go:
- 01-context-basic
- 02-legacy-compat
- 06-simple-demo
- 13-publisher-basic

### Intermediate (⭐⭐)
These examples demonstrate more advanced features:
- 03-timeout
- 04-retry
- 05-phase1-demo
- 07-send-with-context
- 09-graceful-shutdown
- 10-start-timeout
- 11-basic-tracing
- 14-publisher-tracing
- 17-batch-publisher

### Advanced (⭐⭐⭐)
These examples show complex scenarios:
- 08-cascade-timeout
- 12-trace-propagation
- 15-publisher-confirm
- 16-publisher-transaction

## Key Features Demonstrated

### Context Support
- Context-aware message handlers
- Timeout control
- Cancellation support
- Backward compatibility with legacy API

### Graceful Shutdown
- Clean shutdown with `StopWithContext`
- Ensuring all messages are processed
- Signal handling

### Distributed Tracing
- Automatic trace ID generation
- Trace propagation across services
- Context and header-based tracing

### Publisher API
- Simple publishing
- Batch publishing (10-100x performance improvement)
- Confirm mode for reliability
- Transaction mode for atomicity
- BatchPublisher helper for convenience

### Retry Mechanisms
- Exponential backoff
- Linear backoff
- Custom retry strategies
- Per-consumer retry configuration

## Documentation

For more detailed information, see:
- [Main README](../README.md)
- [Documentation](../docs/)
- [CHANGELOG](../CHANGELOG.md)

## Support

If you encounter any issues or have questions:
1. Check the example's README.md file
2. Review the main documentation
3. Open an issue on GitHub

