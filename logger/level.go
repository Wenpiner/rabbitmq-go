package logger

// LogLevel 表示日志级别。
//
// 日志级别用于控制哪些日志消息应该被输出。
// 级别从低到高依次为：Debug < Info < Warn < Error
//
// 当设置日志级别为某个级别时，只有该级别及更高级别的日志会被输出。
// 例如，设置为 LevelInfo 时，Debug 日志不会输出，但 Info、Warn、Error 会输出。
type LogLevel int

const (
	// LevelDebug 是最低的日志级别，用于详细的调试信息。
	// 通常在开发环境使用，生产环境不建议启用。
	LevelDebug LogLevel = iota

	// LevelInfo 是信息级别，用于一般的信息性消息。
	// 这是推荐的默认日志级别。
	LevelInfo

	// LevelWarn 是警告级别，用于警告性消息。
	// 表示可能存在问题但不影响正常运行。
	LevelWarn

	// LevelError 是错误级别，用于错误消息。
	// 表示发生了错误但程序可以继续运行。
	LevelError
)

// String 返回日志级别的字符串表示。
//
// 返回值:
//   - "DEBUG" - 调试级别
//   - "INFO"  - 信息级别
//   - "WARN"  - 警告级别
//   - "ERROR" - 错误级别
//   - "UNKNOWN" - 未知级别
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
