package v2

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/wenpiner/rabbitmq-go/logger"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

func TestFuncHandler_Handle(t *testing.T) {
	called := false
	handler := NewFuncHandler(func(ctx context.Context, msg *Message) error {
		called = true
		return nil
	})

	msg := &Message{ConsumerName: "test"}
	err := handler.Handle(context.Background(), msg)

	if err != nil {
		t.Errorf("Handle() error = %v, want nil", err)
	}
	if !called {
		t.Error("处理函数未被调用")
	}
}

func TestFuncHandler_HandleError(t *testing.T) {
	expectedErr := errors.New("test error")
	handler := NewFuncHandler(func(ctx context.Context, msg *Message) error {
		return expectedErr
	})

	msg := &Message{ConsumerName: "test"}
	err := handler.Handle(context.Background(), msg)

	if !errors.Is(err, expectedErr) {
		t.Errorf("Handle() error = %v, want %v", err, expectedErr)
	}
}

func TestFuncHandler_OnError_Default(t *testing.T) {
	handler := NewFuncHandler(
		func(ctx context.Context, msg *Message) error { return nil },
		WithHandlerLogger(logger.NewNoopLogger()),
	)

	msg := &Message{ConsumerName: "test", RetryCount: 1}
	// 默认错误处理不应 panic
	handler.OnError(context.Background(), msg, errors.New("test error"))
}

func TestFuncHandler_OnError_Custom(t *testing.T) {
	var gotErr error
	var gotMsg *Message

	handler := NewFuncHandler(
		func(ctx context.Context, msg *Message) error { return nil },
		WithErrorHandler(func(ctx context.Context, msg *Message, err error) {
			gotErr = err
			gotMsg = msg
		}),
	)

	expectedErr := errors.New("custom error")
	msg := &Message{ConsumerName: "test-consumer"}
	handler.OnError(context.Background(), msg, expectedErr)

	if !errors.Is(gotErr, expectedErr) {
		t.Errorf("OnError gotErr = %v, want %v", gotErr, expectedErr)
	}
	if gotMsg != msg {
		t.Error("OnError gotMsg 不匹配")
	}
}

func TestChainHandler_Handle(t *testing.T) {
	var order []string

	baseHandler := NewFuncHandler(func(ctx context.Context, msg *Message) error {
		order = append(order, "handler")
		return nil
	})

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			order = append(order, "m1-before")
			err := next(ctx, msg)
			order = append(order, "m1-after")
			return err
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			order = append(order, "m2-before")
			err := next(ctx, msg)
			order = append(order, "m2-after")
			return err
		}
	}

	chain := NewChainHandler(baseHandler, middleware1, middleware2)
	msg := &Message{ConsumerName: "test"}
	_ = chain.Handle(context.Background(), msg)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("执行顺序长度不匹配: got %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("执行顺序[%d] = %s, want %s", i, order[i], v)
		}
	}
}

func TestChainHandler_OnError(t *testing.T) {
	var called bool
	baseHandler := NewFuncHandler(
		func(ctx context.Context, msg *Message) error { return nil },
		WithErrorHandler(func(ctx context.Context, msg *Message, err error) {
			called = true
		}),
	)

	chain := NewChainHandler(baseHandler)
	msg := &Message{ConsumerName: "test"}
	chain.OnError(context.Background(), msg, errors.New("test"))

	if !called {
		t.Error("基础处理器的 OnError 未被调用")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	l := logger.NewNoopLogger()
	middleware := LoggingMiddleware(l)

	var called bool
	next := func(ctx context.Context, msg *Message) error {
		called = true
		return nil
	}

	wrapped := middleware(next)
	msg := &Message{ConsumerName: "test", TraceInfo: tracing.TraceInfo{TraceID: "test-trace"}}
	_ = wrapped(context.Background(), msg)

	if !called {
		t.Error("next 函数未被调用")
	}
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	l := logger.NewNoopLogger()
	middleware := LoggingMiddleware(l)

	expectedErr := errors.New("test error")
	next := func(ctx context.Context, msg *Message) error {
		return expectedErr
	}

	wrapped := middleware(next)
	msg := &Message{ConsumerName: "test", TraceInfo: tracing.TraceInfo{TraceID: "test-trace"}}
	err := wrapped(context.Background(), msg)

	if !errors.Is(err, expectedErr) {
		t.Errorf("错误未正确传递: got %v, want %v", err, expectedErr)
	}
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	l := logger.NewNoopLogger()
	middleware := RecoveryMiddleware(l)

	var called bool
	next := func(ctx context.Context, msg *Message) error {
		called = true
		return nil
	}

	wrapped := middleware(next)
	msg := &Message{ConsumerName: "test"}
	err := wrapped(context.Background(), msg)

	if err != nil {
		t.Errorf("无 panic 时不应返回错误: %v", err)
	}
	if !called {
		t.Error("next 函数未被调用")
	}
}

func TestRecoveryMiddleware_PanicWithError(t *testing.T) {
	l := logger.NewNoopLogger()
	middleware := RecoveryMiddleware(l)

	expectedErr := errors.New("panic error")
	next := func(ctx context.Context, msg *Message) error {
		panic(expectedErr)
	}

	wrapped := middleware(next)
	msg := &Message{ConsumerName: "test"}
	err := wrapped(context.Background(), msg)

	if !errors.Is(err, expectedErr) {
		t.Errorf("panic error 未正确捕获: got %v, want %v", err, expectedErr)
	}
}

func TestRecoveryMiddleware_PanicWithString(t *testing.T) {
	l := logger.NewNoopLogger()
	middleware := RecoveryMiddleware(l)

	next := func(ctx context.Context, msg *Message) error {
		panic("string panic")
	}

	wrapped := middleware(next)
	msg := &Message{ConsumerName: "test"}
	err := wrapped(context.Background(), msg)

	if err == nil {
		t.Error("panic 应该返回错误")
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Errorf("错误类型应为 PanicError, got %T", err)
	}
}

func TestPanicError_Error(t *testing.T) {
	err := &PanicError{Value: "test value"}
	if err.Error() != "panic recovered" {
		t.Errorf("PanicError.Error() = %s, want 'panic recovered'", err.Error())
	}
}

func TestChainHandler_MiddlewareOrder(t *testing.T) {
	var callCount int32

	baseHandler := NewFuncHandler(func(ctx context.Context, msg *Message) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// 创建多个中间件
	createMiddleware := func(name string) MiddlewareFunc {
		return func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, msg *Message) error {
				return next(ctx, msg)
			}
		}
	}

	chain := NewChainHandler(baseHandler,
		createMiddleware("m1"),
		createMiddleware("m2"),
		createMiddleware("m3"),
	)

	msg := &Message{ConsumerName: "test"}
	_ = chain.Handle(context.Background(), msg)

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("处理器应该只被调用一次, got %d", callCount)
	}
}

func TestChainHandler_MiddlewareCanShortCircuit(t *testing.T) {
	var handlerCalled bool

	baseHandler := NewFuncHandler(func(ctx context.Context, msg *Message) error {
		handlerCalled = true
		return nil
	})

	shortCircuitMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			// 不调用 next，直接返回错误
			return errors.New("short circuit")
		}
	}

	chain := NewChainHandler(baseHandler, shortCircuitMiddleware)
	msg := &Message{ConsumerName: "test"}
	err := chain.Handle(context.Background(), msg)

	if err == nil {
		t.Error("应该返回错误")
	}
	if handlerCalled {
		t.Error("处理器不应该被调用")
	}
}

