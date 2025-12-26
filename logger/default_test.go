package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestNewDefaultLogger 测试 DefaultLogger 的创建。
func TestNewDefaultLogger(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{"debug level", LevelDebug},
		{"info level", LevelInfo},
		{"warn level", LevelWarn},
		{"error level", LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewDefaultLogger(tt.level)
			if logger == nil {
				t.Fatal("NewDefaultLogger returned nil")
			}
			if logger.level != tt.level {
				t.Errorf("level = %v, want %v", logger.level, tt.level)
			}
		})
	}
}

// TestDefaultLoggerLevelFiltering 测试日志级别过滤。
func TestDefaultLoggerLevelFiltering(t *testing.T) {
	tests := []struct {
		name         string
		loggerLevel  LogLevel
		logLevel     LogLevel
		shouldOutput bool
	}{
		// Debug 级别的日志器
		{"debug logger logs debug", LevelDebug, LevelDebug, true},
		{"debug logger logs info", LevelDebug, LevelInfo, true},
		{"debug logger logs warn", LevelDebug, LevelWarn, true},
		{"debug logger logs error", LevelDebug, LevelError, true},

		// Info 级别的日志器
		{"info logger skips debug", LevelInfo, LevelDebug, false},
		{"info logger logs info", LevelInfo, LevelInfo, true},
		{"info logger logs warn", LevelInfo, LevelWarn, true},
		{"info logger logs error", LevelInfo, LevelError, true},

		// Warn 级别的日志器
		{"warn logger skips debug", LevelWarn, LevelDebug, false},
		{"warn logger skips info", LevelWarn, LevelInfo, false},
		{"warn logger logs warn", LevelWarn, LevelWarn, true},
		{"warn logger logs error", LevelWarn, LevelError, true},

		// Error 级别的日志器
		{"error logger skips debug", LevelError, LevelDebug, false},
		{"error logger skips info", LevelError, LevelInfo, false},
		{"error logger skips warn", LevelError, LevelWarn, false},
		{"error logger logs error", LevelError, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 捕获日志输出
			var buf bytes.Buffer
			logger := NewDefaultLogger(tt.loggerLevel)
			logger.SetOutput(&buf)

			logger.log(tt.logLevel, "test message")

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldOutput {
				t.Errorf("output = %v, want %v (output: %q)", hasOutput, tt.shouldOutput, output)
			}
		})
	}
}

// TestDefaultLoggerOutput 测试日志输出格式。
func TestDefaultLoggerOutput(t *testing.T) {
	tests := []struct {
		name         string
		level        LogLevel
		msg          string
		fields       []Field
		wantContains []string
	}{
		{
			name:         "debug without fields",
			level:        LevelDebug,
			msg:          "debug message",
			fields:       nil,
			wantContains: []string{"[DEBUG]", "debug message"},
		},
		{
			name:         "info with string field",
			level:        LevelInfo,
			msg:          "info message",
			fields:       []Field{String("key", "value")},
			wantContains: []string{"[INFO]", "info message", "key=value"},
		},
		{
			name:         "warn with multiple fields",
			level:        LevelWarn,
			msg:          "warn message",
			fields:       []Field{String("user", "alice"), Int("count", 42)},
			wantContains: []string{"[WARN]", "warn message", "user=alice", "count=42"},
		},
		{
			name:         "error with error field",
			level:        LevelError,
			msg:          "error message",
			fields:       []Field{Error(errors.New("test error"))},
			wantContains: []string{"[ERROR]", "error message", "error=test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 捕获日志输出
			var buf bytes.Buffer
			logger := NewDefaultLogger(LevelDebug)
			logger.SetOutput(&buf)

			logger.log(tt.level, tt.msg, tt.fields...)

			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output %q does not contain %q", output, want)
				}
			}
		})
	}
}

// TestDefaultLoggerMethods 测试各个日志方法。
func TestDefaultLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDefaultLogger(LevelDebug)
	logger.SetOutput(&buf)

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug msg", String("key", "value"))
		if !strings.Contains(buf.String(), "[DEBUG]") {
			t.Errorf("Debug output does not contain [DEBUG]: %q", buf.String())
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info msg", Int("count", 1))
		if !strings.Contains(buf.String(), "[INFO]") {
			t.Errorf("Info output does not contain [INFO]: %q", buf.String())
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn msg", Bool("flag", true))
		if !strings.Contains(buf.String(), "[WARN]") {
			t.Errorf("Warn output does not contain [WARN]: %q", buf.String())
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error("error msg", Duration("elapsed", time.Second))
		if !strings.Contains(buf.String(), "[ERROR]") {
			t.Errorf("Error output does not contain [ERROR]: %q", buf.String())
		}
	})
}
