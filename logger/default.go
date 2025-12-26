package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// DefaultLogger 是使用标准库 log 包的默认日志实现。
//
// DefaultLogger 支持日志级别过滤和结构化字段输出。
// 它使用标准库的 log 包进行输出，因此输出格式与标准库一致。
//
// 特性:
//   - 支持日志级别过滤（Debug/Info/Warn/Error）
//   - 支持结构化字段输出
//   - 使用标准库 log 包，无额外依赖
//   - 线程安全
//
// 示例:
//
//	logger := logger.NewDefaultLogger(logger.LevelInfo)
//	logger.Debug("这条不会输出")  // 因为级别低于 Info
//	logger.Info("服务启动", logger.String("port", "8080"))
//	logger.Error("连接失败", logger.Error(err))
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewDefaultLogger 创建一个新的 DefaultLogger。
//
// 参数:
//   - level: 日志级别，只有该级别及更高级别的日志会被输出
//
// 返回:
//   - *DefaultLogger: 新创建的日志器
//
// 示例:
//
//	// 创建 Info 级别的日志器（推荐用于生产环境）
//	logger := logger.NewDefaultLogger(logger.LevelInfo)
//
//	// 创建 Debug 级别的日志器（用于开发环境）
//	logger := logger.NewDefaultLogger(logger.LevelDebug)
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// SetOutput 设置日志输出目标（主要用于测试）。
func (l *DefaultLogger) SetOutput(w interface{ Write([]byte) (int, error) }) {
	l.logger.SetOutput(w)
}

// Debug 记录调试级别的日志。
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

// Info 记录信息级别的日志。
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

// Warn 记录警告级别的日志。
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

// Error 记录错误级别的日志。
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

// log 是内部日志方法，处理日志级别过滤和格式化。
func (l *DefaultLogger) log(level LogLevel, msg string, fields ...Field) {
	// 日志级别过滤
	if level < l.level {
		return
	}

	// 格式化日志消息
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s", level.String(), msg))

	// 添加结构化字段
	if len(fields) > 0 {
		sb.WriteString(" |")
		for _, field := range fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", field.Key, formatValue(field.Value)))
		}
	}

	// 输出日志
	l.logger.Println(sb.String())
}

// formatValue 格式化字段值。
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case time.Duration:
		return v.String()
	case error:
		return v.Error()
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%v", v)
	}
}
