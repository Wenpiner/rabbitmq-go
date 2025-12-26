// Package logger 提供了一个轻量级、可插拔的日志接口。
//
// 该包定义了 Logger 接口，允许用户使用任何日志库（如 zap、logrus、zerolog）
// 或使用内置的 DefaultLogger 和 NoopLogger 实现。
//
// 基本用法:
//
//	// 使用默认日志器
//	logger := logger.NewDefaultLogger(logger.LevelInfo)
//	logger.Info("服务启动", logger.String("port", "8080"))
//
//	// 使用空日志器（不输出任何日志）
//	logger := logger.NewNoopLogger()
//
//	// 使用自定义日志器（实现 Logger 接口）
//	type MyLogger struct{}
//	func (l *MyLogger) Debug(msg string, fields ...Field) { /* ... */ }
//	func (l *MyLogger) Info(msg string, fields ...Field)  { /* ... */ }
//	func (l *MyLogger) Warn(msg string, fields ...Field)  { /* ... */ }
//	func (l *MyLogger) Error(msg string, fields ...Field) { /* ... */ }
package logger

// Logger 定义了日志接口。
//
// 所有日志方法都接受一个消息字符串和可选的结构化字段。
// 结构化字段使用 Field 类型表示，可以通过辅助函数创建：
//   - String(key, value) - 字符串字段
//   - Int(key, value) - 整数字段
//   - Error(err) - 错误字段
//   - 等等...
//
// 示例:
//
//	logger.Info("用户登录",
//	    logger.String("username", "alice"),
//	    logger.Int("user_id", 123),
//	    logger.Duration("elapsed", time.Second))
type Logger interface {
	// Debug 记录调试级别的日志。
	// 用于详细的调试信息，通常在生产环境中不启用。
	Debug(msg string, fields ...Field)

	// Info 记录信息级别的日志。
	// 用于一般的信息性消息，如服务启动、配置加载等。
	Info(msg string, fields ...Field)

	// Warn 记录警告级别的日志。
	// 用于警告性消息，表示可能存在问题但不影响正常运行。
	Warn(msg string, fields ...Field)

	// Error 记录错误级别的日志。
	// 用于错误消息，表示发生了错误但程序可以继续运行。
	Error(msg string, fields ...Field)
}

// Field 表示一个结构化日志字段。
//
// Field 包含一个键和一个值，用于在日志中添加结构化信息。
// 不要直接创建 Field，而是使用辅助函数：
//   - String(key, value)
//   - Int(key, value)
//   - Error(err)
//   - 等等...
//
// 示例:
//
//	field := logger.String("username", "alice")
//	// field.Key = "username"
//	// field.Value = "alice"
type Field struct {
	// Key 是字段的键名
	Key string

	// Value 是字段的值，可以是任意类型
	Value interface{}
}
