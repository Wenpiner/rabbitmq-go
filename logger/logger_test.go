package logger

import (
	"errors"
	"testing"
	"time"
)

// TestField 测试 Field 结构体的基本功能。
func TestField(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		wantKey  string
		wantType string
	}{
		{
			name:     "string field",
			field:    String("name", "alice"),
			wantKey:  "name",
			wantType: "string",
		},
		{
			name:     "int field",
			field:    Int("count", 42),
			wantKey:  "count",
			wantType: "int",
		},
		{
			name:     "bool field",
			field:    Bool("enabled", true),
			wantKey:  "enabled",
			wantType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.wantKey {
				t.Errorf("Field.Key = %v, want %v", tt.field.Key, tt.wantKey)
			}
			if tt.field.Value == nil {
				t.Errorf("Field.Value is nil")
			}
		})
	}
}

// TestStringField 测试 String 辅助函数。
func TestStringField(t *testing.T) {
	field := String("username", "alice")
	if field.Key != "username" {
		t.Errorf("Key = %v, want username", field.Key)
	}
	if field.Value != "alice" {
		t.Errorf("Value = %v, want alice", field.Value)
	}
}

// TestIntField 测试 Int 辅助函数。
func TestIntField(t *testing.T) {
	field := Int("count", 42)
	if field.Key != "count" {
		t.Errorf("Key = %v, want count", field.Key)
	}
	if field.Value != 42 {
		t.Errorf("Value = %v, want 42", field.Value)
	}
}

// TestInt32Field 测试 Int32 辅助函数。
func TestInt32Field(t *testing.T) {
	field := Int32("retry", 3)
	if field.Key != "retry" {
		t.Errorf("Key = %v, want retry", field.Key)
	}
	if field.Value != int32(3) {
		t.Errorf("Value = %v, want 3", field.Value)
	}
}

// TestInt64Field 测试 Int64 辅助函数。
func TestInt64Field(t *testing.T) {
	field := Int64("size", 1024000)
	if field.Key != "size" {
		t.Errorf("Key = %v, want size", field.Key)
	}
	if field.Value != int64(1024000) {
		t.Errorf("Value = %v, want 1024000", field.Value)
	}
}

// TestBoolField 测试 Bool 辅助函数。
func TestBoolField(t *testing.T) {
	field := Bool("enabled", true)
	if field.Key != "enabled" {
		t.Errorf("Key = %v, want enabled", field.Key)
	}
	if field.Value != true {
		t.Errorf("Value = %v, want true", field.Value)
	}
}

// TestDurationField 测试 Duration 辅助函数。
func TestDurationField(t *testing.T) {
	field := Duration("elapsed", time.Second*2)
	if field.Key != "elapsed" {
		t.Errorf("Key = %v, want elapsed", field.Key)
	}
	if field.Value != time.Second*2 {
		t.Errorf("Value = %v, want 2s", field.Value)
	}
}

// TestErrorField 测试 Error 辅助函数。
func TestErrorField(t *testing.T) {
	t.Run("non-nil error", func(t *testing.T) {
		err := errors.New("test error")
		field := Error(err)
		if field.Key != "error" {
			t.Errorf("Key = %v, want error", field.Key)
		}
		if field.Value != "test error" {
			t.Errorf("Value = %v, want 'test error'", field.Value)
		}
	})

	t.Run("nil error", func(t *testing.T) {
		field := Error(nil)
		if field.Key != "error" {
			t.Errorf("Key = %v, want error", field.Key)
		}
		if field.Value != nil {
			t.Errorf("Value = %v, want nil", field.Value)
		}
	})
}

// TestAnyField 测试 Any 辅助函数。
func TestAnyField(t *testing.T) {
	type testStruct struct {
		Name string
		Age  int
	}

	field := Any("data", testStruct{Name: "alice", Age: 30})
	if field.Key != "data" {
		t.Errorf("Key = %v, want data", field.Key)
	}
	// Any 会格式化为字符串
	if _, ok := field.Value.(string); !ok {
		t.Errorf("Value type = %T, want string", field.Value)
	}
}
