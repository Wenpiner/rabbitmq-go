# Logger Package

A pluggable structured logging system with support for custom logger implementations.

## Features

- ✅ **Pluggable Interface** - Implement the `Logger` interface to integrate any logging library
- ✅ **Log Levels** - Support for Debug/Info/Warn/Error levels
- ✅ **Structured Fields** - Type-safe structured logging fields
- ✅ **Default Implementation** - Built-in logger based on standard library `log` package
- ✅ **Zero Overhead** - `NoopLogger` provides zero-overhead logging for production
- ✅ **Easy Integration** - Simple integration with third-party loggers like zap, logrus, etc.

## Quick Start

### Using Default Logger

```go
import "github.com/wenpiner/rabbitmq-go/logger"

// Create logger with Info level
log := logger.NewDefaultLogger(logger.LevelInfo)

// Log messages
log.Debug("This won't be output")  // Below Info level
log.Info("Service started", logger.String("port", "8080"))
log.Warn("Slow connection", logger.Duration("latency", 500*time.Millisecond))
log.Error("Connection failed", logger.Error(err))
```

### Disable Logging

```go
// Use NoopLogger for zero performance overhead
log := logger.NewNoopLogger()
```

### Using with RabbitMQ

```go
import (
    rabbitmq "github.com/wenpiner/rabbitmq-go"
    "github.com/wenpiner/rabbitmq-go/logger"
)

rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{...})

// Set log level
rabbit.SetLogger(logger.NewDefaultLogger(logger.LevelInfo))

// Or disable logging
rabbit.SetLogger(logger.NewNoopLogger())
```

## Log Levels

```go
const (
    LevelDebug LogLevel = iota  // Debug information
    LevelInfo                   // General information
    LevelWarn                   // Warning information
    LevelError                  // Error information
)
```

## Structured Fields

Supported field types:

```go
// String field
logger.String("key", "value")

// Integer fields
logger.Int("count", 42)
logger.Int32("retry", 3)
logger.Int64("id", 123456)

// Boolean field
logger.Bool("success", true)

// Duration field
logger.Duration("elapsed", time.Second)

// Error field
logger.Error(err)

// Any type
logger.Any("data", complexObject)
```

## Custom Logger Implementation

### Integrating with Zap

```go
import (
    "go.uber.org/zap"
    "github.com/wenpiner/rabbitmq-go/logger"
)

type ZapLogger struct {
    zap *zap.Logger
}

func (l *ZapLogger) Debug(msg string, fields ...logger.Field) {
    l.zap.Debug(msg, convertFields(fields)...)
}

func (l *ZapLogger) Info(msg string, fields ...logger.Field) {
    l.zap.Info(msg, convertFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields ...logger.Field) {
    l.zap.Warn(msg, convertFields(fields)...)
}

func (l *ZapLogger) Error(msg string, fields ...logger.Field) {
    l.zap.Error(msg, convertFields(fields)...)
}

func convertFields(fields []logger.Field) []zap.Field {
    zapFields := make([]zap.Field, len(fields))
    for i, f := range fields {
        zapFields[i] = zap.Any(f.Key, f.Value)
    }
    return zapFields
}

// Usage
zapLogger, _ := zap.NewProduction()
rabbit.SetLogger(&ZapLogger{zap: zapLogger})
```

### Integrating with Logrus

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/wenpiner/rabbitmq-go/logger"
)

type LogrusLogger struct {
    logrus *logrus.Logger
}

func (l *LogrusLogger) Debug(msg string, fields ...logger.Field) {
    l.logrus.WithFields(convertFields(fields)).Debug(msg)
}

// ... implement other methods

func convertFields(fields []logger.Field) logrus.Fields {
    logrusFields := make(logrus.Fields)
    for _, f := range fields {
        logrusFields[f.Key] = f.Value
    }
    return logrusFields
}
```

## API Documentation

See godoc comments in each file:
- `logger.go` - Logger interface definition
- `default.go` - DefaultLogger implementation
- `noop.go` - NoopLogger implementation
- `field.go` - Structured field definitions
- `level.go` - Log level definitions

