package tracing

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestGenerateTraceID(t *testing.T) {
	traceID := GenerateTraceID()
	if traceID == "" {
		t.Error("Generated trace ID should not be empty")
	}

	// 生成两个 trace ID 应该不同
	traceID2 := GenerateTraceID()
	if traceID == traceID2 {
		t.Error("Generated trace IDs should be unique")
	}
}

func TestGenerateSpanID(t *testing.T) {
	spanID := GenerateSpanID()
	if spanID == "" {
		t.Error("Generated span ID should not be empty")
	}

	// 生成两个 span ID 应该不同
	spanID2 := GenerateSpanID()
	if spanID == spanID2 {
		t.Error("Generated span IDs should be unique")
	}
}

func TestInjectExtractHeaders(t *testing.T) {
	info := TraceInfo{
		TraceID:      "trace-123",
		SpanID:       "span-456",
		ParentSpanID: "parent-789",
	}

	headers := make(amqp.Table)
	InjectToHeaders(headers, info)

	extracted := ExtractFromHeaders(headers)

	if extracted.TraceID != info.TraceID {
		t.Errorf("Expected trace ID %s, got %s", info.TraceID, extracted.TraceID)
	}
	if extracted.SpanID != info.SpanID {
		t.Errorf("Expected span ID %s, got %s", info.SpanID, extracted.SpanID)
	}
	if extracted.ParentSpanID != info.ParentSpanID {
		t.Errorf("Expected parent span ID %s, got %s", info.ParentSpanID, extracted.ParentSpanID)
	}
}

func TestInjectExtractHeadersNil(t *testing.T) {
	info := TraceInfo{
		TraceID: "trace-123",
	}

	// 测试 nil headers
	InjectToHeaders(nil, info)

	extracted := ExtractFromHeaders(nil)
	if extracted.TraceID != "" {
		t.Error("Extracted trace ID from nil headers should be empty")
	}
}

func TestInjectExtractContext(t *testing.T) {
	info := TraceInfo{
		TraceID:      "trace-123",
		SpanID:       "span-456",
		ParentSpanID: "parent-789",
	}

	ctx := context.Background()
	ctx = InjectToContext(ctx, info)

	extracted := ExtractFromContext(ctx)

	if extracted.TraceID != info.TraceID {
		t.Errorf("Expected trace ID %s, got %s", info.TraceID, extracted.TraceID)
	}
	if extracted.SpanID != info.SpanID {
		t.Errorf("Expected span ID %s, got %s", info.SpanID, extracted.SpanID)
	}
	if extracted.ParentSpanID != info.ParentSpanID {
		t.Errorf("Expected parent span ID %s, got %s", info.ParentSpanID, extracted.ParentSpanID)
	}
}

func TestGetTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "test-trace-id")

	traceID := GetTraceID(ctx)
	if traceID != "test-trace-id" {
		t.Errorf("Expected trace ID 'test-trace-id', got '%s'", traceID)
	}

	// 测试空 context
	emptyCtx := context.Background()
	emptyTraceID := GetTraceID(emptyCtx)
	if emptyTraceID != "" {
		t.Error("Expected empty trace ID from empty context")
	}
}

func TestGetSpanID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, SpanIDKey, "test-span-id")

	spanID := GetSpanID(ctx)
	if spanID != "test-span-id" {
		t.Errorf("Expected span ID 'test-span-id', got '%s'", spanID)
	}
}

func TestGetParentSpanID(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ParentSpanIDKey, "test-parent-span-id")

	parentSpanID := GetParentSpanID(ctx)
	if parentSpanID != "test-parent-span-id" {
		t.Errorf("Expected parent span ID 'test-parent-span-id', got '%s'", parentSpanID)
	}
}

func TestFormatTraceLog(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		message  string
		expected string
	}{
		{
			name:     "with trace ID and span ID",
			ctx:      InjectToContext(context.Background(), TraceInfo{TraceID: "trace-123", SpanID: "span-456"}),
			message:  "test message",
			expected: "[trace_id=trace-123 span_id=span-456] test message",
		},
		{
			name:     "with trace ID only",
			ctx:      InjectToContext(context.Background(), TraceInfo{TraceID: "trace-123"}),
			message:  "test message",
			expected: "[trace_id=trace-123] test message",
		},
		{
			name:     "without trace info",
			ctx:      context.Background(),
			message:  "test message",
			expected: "test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTraceLog(tt.ctx, tt.message)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
