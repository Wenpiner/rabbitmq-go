# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Context Support
- New `ReceiveWithContext` interface for handlers with context support
- New `ExceptionWithContext` interface for exception handling with context
- `HandlerTimeout` configuration for handler timeout control
- Context-aware message sending methods:
  - `SendDelayMsg(ctx, ...)`
  - `SendDelayMsgByArgs(ctx, ...)`
  - `SendDelayMsgByKey(ctx, ...)`
- Automatic timeout detection and retry mechanism
- Cascading timeout control across the message chain

#### Graceful Shutdown
- `StartWithContext(ctx)` method for controlled startup with timeout
- `StopWithContext(ctx)` method for graceful shutdown
- WaitGroup tracking for all goroutines
- Ensures all in-flight messages are processed before shutdown
- Configurable shutdown timeout

#### Distributed Tracing
- New `tracing` package for distributed tracing support
- `TraceInfo` structure with TraceID, SpanID, and ParentSpanID
- Automatic trace ID generation and propagation
- `SendMessageWithTrace` method for automatic trace injection
- Context and header-based trace information propagation
- `FormatTraceLog` function for structured logging with trace IDs
- Automatic trace extraction in message handlers

#### Publisher API
- **Basic Publishing**:
  - `Publish(ctx, exchange, route, msg)` - Simple publish
  - `PublishMandatory(ctx, exchange, route, msg)` - Mandatory routing
  - `PublishImmediate(ctx, exchange, route, msg)` - Immediate delivery
  - `PublishWithTrace(ctx, exchange, route, msg)` - Publish with tracing

- **Batch Publishing** (10-100x performance improvement):
  - `PublishBatch(ctx, exchange, route, messages)` - High-performance batch send
  - `PublishBatchWithConfirm(ctx, exchange, route, messages)` - Reliable batch send
  - `PublishBatchTx(ctx, exchange, route, messages)` - Atomic batch send

- **Advanced Publishing**:
  - `WithPublisher(ctx, fn)` - Custom publisher with auto-managed channel
  - `WithPublisherConfirm(ctx, fn)` - Publisher with confirm mode
  - `WithPublisherTx(ctx, fn)` - Publisher with transaction mode

- **BatchPublisher Helper**:
  - `NewBatchPublisher(exchange, route)` - Create batch publisher
  - `SetBatchSize(size)` - Configure batch size
  - `SetAutoFlush(enabled)` - Enable auto-flush
  - `Add(ctx, msg)` - Add message to batch
  - `Flush(ctx)` - Manual flush
  - `FlushWithConfirm(ctx)` - Flush with confirm
  - `FlushTx(ctx)` - Flush with transaction
  - `Close(ctx)` - Close and flush remaining messages

#### Deprecation Warnings
- Graceful deprecation mechanism for old APIs
- One-time warning per deprecated API per instance
- Clear migration guidance with documentation links
- Deprecated APIs:
  - `Channel()` → Use `WithPublisher()`
  - `ChannelByName()` → Use `WithPublisher()`
  - `SendMessage()` → Use `Publish()`
  - `SendMessageClose()` → Use `Publish()`
  - `SendMessageWithTrace()` → Use `PublishWithTrace()`

#### Structured Logging
- New `logger` package with pluggable logging interface
- `Logger` interface with Debug/Info/Warn/Error methods
- `DefaultLogger` implementation with level filtering
- `NoopLogger` for zero-overhead production logging
- Structured logging fields (String, Int, Bool, Duration, Error, etc.)
- `SetLogger()` method for custom logger integration
- Easy integration with third-party loggers (zap, logrus, etc.)
- All internal logging migrated to structured logger
- Chinese log messages for better localization

### Changed
- Handler interface now supports both `Receive` and `ReceiveWithContext`
- Message sending methods now accept optional context parameter
- Internal goroutines now respect context cancellation
- Deprecated APIs now internally call new APIs for consistency

### Deprecated
- `Receive` interface (still supported, but `ReceiveWithContext` is recommended)
- `Exception` interface (still supported, but `ExceptionWithContext` is recommended)
- `Start()` method (still supported, but `StartWithContext` is recommended)
- `Stop()` method (still supported, but `StopWithContext` is recommended)
- `Channel()` method (still supported, but `WithPublisher()` is recommended)
- `ChannelByName()` method (still supported, but `WithPublisher()` is recommended)
- `SendMessage()` method (still supported, but `Publish()` is recommended)
- `SendMessageClose()` method (still supported, but `Publish()` is recommended)
- `SendMessageWithTrace()` method (still supported, but `PublishWithTrace()` is recommended)

### Performance Improvements
- Batch publishing: 10-100x faster than loop-based sending
- Optimized channel management reduces resource overhead
- Improved connection retry mechanism

### Documentation
- Added comprehensive usage guides for all new features:
  - [Context Best Practices](docs/CONTEXT_BEST_PRACTICES.md)
  - [Graceful Shutdown Guide](docs/GRACEFUL_SHUTDOWN_GUIDE.md)
  - [Distributed Tracing Guide](docs/TRACING_GUIDE.md)
  - [Publisher API Quick Reference](docs/PUBLISHER_API_QUICK_REFERENCE.md)
  - [Publisher Migration Guide](docs/PUBLISHER_MIGRATION_GUIDE.md)
  - [v4.0.0 Migration Guide](docs/MIGRATION_GUIDE.md)
- Added 18 working examples covering all core functionality
- Added [Documentation Index](DOCUMENTATION_INDEX.md) for easy navigation
- Added [Release Notes](RELEASE_NOTES_v4.0.0.md) for v4.0.0
- Updated README with new features and quick start guide

### Testing
- Added unit tests for deprecation warnings (100% pass rate)
- Added unit tests for tracing package (100% pass rate)
- All 18 examples compile and run successfully
- Comprehensive verification with automated scripts

### Bug Fixes
- Fixed channel resource leak issues
- Improved connection retry mechanism
- Enhanced error handling throughout the codebase

## [3.x.x] - Previous Versions

### Features
- Automatic reconnection on connection failures
- Flexible retry strategies (exponential backoff, linear backoff, custom)
- QoS support for message prefetching control
- Type-safe API leveraging Go's type system
- Production-ready with battle-tested reliability

---

---

## Migration Guide

See [docs/MIGRATION_GUIDE.md](docs/MIGRATION_GUIDE.md) for detailed migration instructions from v3.x to v4.0.0.

For Publisher API migration, see [docs/PUBLISHER_MIGRATION_GUIDE.md](docs/PUBLISHER_MIGRATION_GUIDE.md).

## Documentation

### Quick Start
- [README.md](README.md) - Project overview and quick start
- [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) - Complete documentation index
- [RELEASE_NOTES_v4.0.0.md](RELEASE_NOTES_v4.0.0.md) - v4.0.0 release notes

### Feature Guides
- [docs/CONTEXT_BEST_PRACTICES.md](docs/CONTEXT_BEST_PRACTICES.md) - Context usage guide
- [docs/GRACEFUL_SHUTDOWN_GUIDE.md](docs/GRACEFUL_SHUTDOWN_GUIDE.md) - Graceful shutdown guide
- [docs/TRACING_GUIDE.md](docs/TRACING_GUIDE.md) - Distributed tracing guide
- [RETRY.md](RETRY.md) - Retry mechanism guide

### Publisher API
- [docs/PUBLISHER_API_QUICK_REFERENCE.md](docs/PUBLISHER_API_QUICK_REFERENCE.md) - Quick reference
- [docs/PUBLISHER_MIGRATION_GUIDE.md](docs/PUBLISHER_MIGRATION_GUIDE.md) - Migration guide
- [PUBLISHER_API_SUMMARY.md](PUBLISHER_API_SUMMARY.md) - Implementation summary

### Examples
- [examples/](examples/) - 18 working examples for all features

