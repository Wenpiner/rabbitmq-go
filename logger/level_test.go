package logger

import "testing"

// TestLogLevelString 测试 LogLevel 的 String 方法。
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  string
	}{
		{"debug level", LevelDebug, "DEBUG"},
		{"info level", LevelInfo, "INFO"},
		{"warn level", LevelWarn, "WARN"},
		{"error level", LevelError, "ERROR"},
		{"unknown level", LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLogLevelOrdering 测试日志级别的顺序。
func TestLogLevelOrdering(t *testing.T) {
	if LevelDebug >= LevelInfo {
		t.Error("LevelDebug should be less than LevelInfo")
	}
	if LevelInfo >= LevelWarn {
		t.Error("LevelInfo should be less than LevelWarn")
	}
	if LevelWarn >= LevelError {
		t.Error("LevelWarn should be less than LevelError")
	}
}

// TestLogLevelValues 测试日志级别的值。
func TestLogLevelValues(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  int
	}{
		{"debug is 0", LevelDebug, 0},
		{"info is 1", LevelInfo, 1},
		{"warn is 2", LevelWarn, 2},
		{"error is 3", LevelError, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int(tt.level)
			if got != tt.want {
				t.Errorf("LogLevel value = %v, want %v", got, tt.want)
			}
		})
	}
}
