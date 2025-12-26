package logger

import (
	"fmt"
	"time"
)

// String 创建一个字符串类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: 字符串值
//
// 示例:
//
//	logger.Info("用户登录", logger.String("username", "alice"))
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建一个 int 类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: 整数值
//
// 示例:
//
//	logger.Info("处理请求", logger.Int("count", 42))
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int32 创建一个 int32 类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: int32 值
//
// 示例:
//
//	logger.Info("重试次数", logger.Int32("retry", 3))
func Int32(key string, value int32) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建一个 int64 类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: int64 值
//
// 示例:
//
//	logger.Info("文件大小", logger.Int64("size", 1024000))
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建一个布尔类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: 布尔值
//
// 示例:
//
//	logger.Info("配置加载", logger.Bool("success", true))
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration 创建一个时间间隔类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: time.Duration 值
//
// 示例:
//
//	logger.Info("请求完成", logger.Duration("elapsed", time.Second*2))
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Error 创建一个错误类型的 Field。
//
// 参数:
//   - err: 错误对象
//
// 返回:
//   - 如果 err 为 nil，返回 Field{Key: "error", Value: nil}
//   - 否则返回 Field{Key: "error", Value: err.Error()}
//
// 示例:
//
//	if err := doSomething(); err != nil {
//	    logger.Error("操作失败", logger.Error(err))
//	}
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any 创建一个任意类型的 Field。
//
// 参数:
//   - key: 字段的键名
//   - value: 任意类型的值
//
// 注意:
//   - 该函数会使用 fmt.Sprintf("%+v", value) 格式化值
//   - 对于复杂类型，建议使用更具体的 Field 函数
//
// 示例:
//
//	logger.Info("配置", logger.Any("config", myConfig))
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: fmt.Sprintf("%+v", value)}
}
