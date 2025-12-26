package tracing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ContextKey 用于在 context 中存储追踪信息的 key 类型
type ContextKey string

const (
	// TraceIDKey trace ID 的 context key
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey span ID 的 context key
	SpanIDKey ContextKey = "span_id"
	// ParentSpanIDKey parent span ID 的 context key
	ParentSpanIDKey ContextKey = "parent_span_id"
)

// TraceInfo 追踪信息
type TraceInfo struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// GenerateTraceID 生成新的 trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateSpanID 生成新的 span ID
func GenerateSpanID() string {
	return uuid.New().String()
}

// InjectToHeaders 将追踪信息注入到消息 headers 中
func InjectToHeaders(headers amqp.Table, info TraceInfo) {
	if headers == nil {
		return
	}

	if info.TraceID != "" {
		headers["trace_id"] = info.TraceID
	}
	if info.SpanID != "" {
		headers["span_id"] = info.SpanID
	}
	if info.ParentSpanID != "" {
		headers["parent_span_id"] = info.ParentSpanID
	}
}

// ExtractFromHeaders 从消息 headers 中提取追踪信息
func ExtractFromHeaders(headers amqp.Table) TraceInfo {
	info := TraceInfo{}

	if headers == nil {
		return info
	}

	if traceID, ok := headers["trace_id"].(string); ok {
		info.TraceID = traceID
	}
	if spanID, ok := headers["span_id"].(string); ok {
		info.SpanID = spanID
	}
	if parentSpanID, ok := headers["parent_span_id"].(string); ok {
		info.ParentSpanID = parentSpanID
	}

	return info
}

// InjectToContext 将追踪信息注入到 context 中
func InjectToContext(ctx context.Context, info TraceInfo) context.Context {
	if info.TraceID != "" {
		ctx = context.WithValue(ctx, TraceIDKey, info.TraceID)
	}
	if info.SpanID != "" {
		ctx = context.WithValue(ctx, SpanIDKey, info.SpanID)
	}
	if info.ParentSpanID != "" {
		ctx = context.WithValue(ctx, ParentSpanIDKey, info.ParentSpanID)
	}
	return ctx
}

// ExtractFromContext 从 context 中提取追踪信息
func ExtractFromContext(ctx context.Context) TraceInfo {
	info := TraceInfo{}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		info.TraceID = traceID
	}
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		info.SpanID = spanID
	}
	if parentSpanID, ok := ctx.Value(ParentSpanIDKey).(string); ok {
		info.ParentSpanID = parentSpanID
	}

	return info
}

// GetTraceID 从 context 中获取 trace ID
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID 从 context 中获取 span ID
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// GetParentSpanID 从 context 中获取 parent span ID
func GetParentSpanID(ctx context.Context) string {
	if parentSpanID, ok := ctx.Value(ParentSpanIDKey).(string); ok {
		return parentSpanID
	}
	return ""
}

// FormatTraceLog 格式化追踪日志前缀
func FormatTraceLog(ctx context.Context, message string) string {
	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	if traceID != "" && spanID != "" {
		return fmt.Sprintf("[trace_id=%s span_id=%s] %s", traceID, spanID, message)
	} else if traceID != "" {
		return fmt.Sprintf("[trace_id=%s] %s", traceID, message)
	}

	return message
}
