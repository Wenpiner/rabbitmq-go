package logger

import (
	"bytes"
	"errors"
	"log"
	"testing"
	"time"
)

// TestNewNoopLogger 测试 NoopLogger 的创建。
func TestNewNoopLogger(t *testing.T) {
	logger := NewNoopLogger()
	if logger == nil {
		t.Fatal("NewNoopLogger returned nil")
	}
}

// TestNoopLoggerNoOutput 测试 NoopLogger 不输出任何日志。
func TestNoopLoggerNoOutput(t *testing.T) {
	// 捕获日志输出
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	logger := NewNoopLogger()

	// 调用所有日志方法
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 42))
	logger.Warn("warn message", Bool("flag", true))
	logger.Error("error message", Error(errors.New("test error")))

	// 验证没有任何输出
	output := buf.String()
	if len(output) > 0 {
		t.Errorf("NoopLogger produced output: %q", output)
	}
}

// TestNoopLoggerWithManyFields 测试 NoopLogger 处理大量字段时不输出。
func TestNoopLoggerWithManyFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	logger := NewNoopLogger()

	// 创建大量字段
	fields := []Field{
		String("field1", "value1"),
		String("field2", "value2"),
		Int("field3", 3),
		Int64("field4", 4),
		Bool("field5", true),
		Duration("field6", time.Second),
		Error(errors.New("test error")),
		Any("field8", map[string]string{"key": "value"}),
	}

	logger.Info("message with many fields", fields...)

	// 验证没有任何输出
	output := buf.String()
	if len(output) > 0 {
		t.Errorf("NoopLogger produced output: %q", output)
	}
}

// TestNoopLoggerImplementsInterface 测试 NoopLogger 实现了 Logger 接口。
func TestNoopLoggerImplementsInterface(t *testing.T) {
	var _ Logger = (*NoopLogger)(nil)
}

// BenchmarkNoopLogger 基准测试 NoopLogger 的性能。
func BenchmarkNoopLogger(b *testing.B) {
	logger := NewNoopLogger()

	b.Run("Debug", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Debug("test message")
		}
	})

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("test message")
		}
	})

	b.Run("Warn", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Warn("test message")
		}
	})

	b.Run("Error", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Error("test message")
		}
	})

	b.Run("WithFields", func(b *testing.B) {
		fields := []Field{
			String("key1", "value1"),
			Int("key2", 42),
			Bool("key3", true),
		}
		for i := 0; i < b.N; i++ {
			logger.Info("test message", fields...)
		}
	})
}
