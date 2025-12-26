package logger

// NoopLogger 是一个空日志实现，不输出任何日志。
//
// NoopLogger 实现了 Logger 接口，但所有方法都是空操作。
// 这对于以下场景非常有用：
//   - 生产环境需要完全关闭日志以提高性能
//   - 测试环境不需要日志输出
//   - 临时禁用日志进行调试
//
// 性能:
//   - NoopLogger 的性能开销几乎为零（< 1ns/op）
//   - 编译器会优化掉空方法调用
//
// 示例:
//
//	// 创建空日志器
//	logger := logger.NewNoopLogger()
//
//	// 这些调用不会产生任何输出
//	logger.Debug("这不会输出")
//	logger.Info("这也不会输出")
//	logger.Error("即使是错误也不会输出")
type NoopLogger struct{}

// NewNoopLogger 创建一个新的 NoopLogger。
//
// 返回:
//   - *NoopLogger: 新创建的空日志器
//
// 示例:
//
//	logger := logger.NewNoopLogger()
//	rabbit.SetLogger(logger)  // 完全关闭 RabbitMQ 的日志
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Debug 是空操作，不输出任何日志。
func (l *NoopLogger) Debug(msg string, fields ...Field) {
	// 空操作
}

// Info 是空操作，不输出任何日志。
func (l *NoopLogger) Info(msg string, fields ...Field) {
	// 空操作
}

// Warn 是空操作，不输出任何日志。
func (l *NoopLogger) Warn(msg string, fields ...Field) {
	// 空操作
}

// Error 是空操作，不输出任何日志。
func (l *NoopLogger) Error(msg string, fields ...Field) {
	// 空操作
}
