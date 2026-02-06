package v2

import (
	"context"

	"github.com/wenpiner/rabbitmq-go/logger"
)

// HandlerFunc 消息处理函数类型
type HandlerFunc func(ctx context.Context, msg *Message) error

// ErrorHandlerFunc 错误处理函数类型
type ErrorHandlerFunc func(ctx context.Context, msg *Message, err error)

// FuncHandler 基于函数的处理器
type FuncHandler struct {
	handleFunc HandlerFunc
	errorFunc  ErrorHandlerFunc
	logger     logger.Logger
}

// NewFuncHandler 创建函数处理器
func NewFuncHandler(handleFunc HandlerFunc, opts ...FuncHandlerOption) *FuncHandler {
	h := &FuncHandler{
		handleFunc: handleFunc,
		logger:     logger.NewDefaultLogger(logger.LevelInfo),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// FuncHandlerOption 函数处理器选项
type FuncHandlerOption func(*FuncHandler)

// WithErrorHandler 设置错误处理函数
func WithErrorHandler(fn ErrorHandlerFunc) FuncHandlerOption {
	return func(h *FuncHandler) {
		h.errorFunc = fn
	}
}

// WithHandlerLogger 设置日志器
func WithHandlerLogger(l logger.Logger) FuncHandlerOption {
	return func(h *FuncHandler) {
		h.logger = l
	}
}

// Handle 实现 MessageHandler 接口
func (h *FuncHandler) Handle(ctx context.Context, msg *Message) error {
	return h.handleFunc(ctx, msg)
}

// OnError 实现 MessageHandler 接口
func (h *FuncHandler) OnError(ctx context.Context, msg *Message, err error) {
	if h.errorFunc != nil {
		h.errorFunc(ctx, msg, err)
		return
	}

	// 默认错误处理：记录日志
	h.logger.Error("消息处理失败",
		logger.String("consumer", msg.ConsumerName),
		logger.Int32("retry", msg.RetryCount),
		logger.Error(err))
}

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// ChainHandler 链式处理器，支持中间件
type ChainHandler struct {
	handler     MessageHandler
	middlewares []MiddlewareFunc
}

// NewChainHandler 创建链式处理器
func NewChainHandler(handler MessageHandler, middlewares ...MiddlewareFunc) *ChainHandler {
	return &ChainHandler{
		handler:     handler,
		middlewares: middlewares,
	}
}

// Handle 实现 MessageHandler 接口
func (ch *ChainHandler) Handle(ctx context.Context, msg *Message) error {
	// 构建处理链
	final := func(ctx context.Context, msg *Message) error {
		return ch.handler.Handle(ctx, msg)
	}

	// 从后往前包装中间件
	for i := len(ch.middlewares) - 1; i >= 0; i-- {
		final = ch.middlewares[i](final)
	}

	return final(ctx, msg)
}

// OnError 实现 MessageHandler 接口
func (ch *ChainHandler) OnError(ctx context.Context, msg *Message, err error) {
	ch.handler.OnError(ctx, msg, err)
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(l logger.Logger) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) error {
			l.Debug("开始处理消息",
				logger.String("consumer", msg.ConsumerName),
				logger.String("trace_id", msg.TraceInfo.TraceID))

			err := next(ctx, msg)

			if err != nil {
				l.Warn("消息处理出错",
					logger.String("consumer", msg.ConsumerName),
					logger.Error(err))
			} else {
				l.Debug("消息处理完成",
					logger.String("consumer", msg.ConsumerName))
			}

			return err
		}
	}
}

// RecoveryMiddleware 恢复中间件，捕获 panic
func RecoveryMiddleware(l logger.Logger) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					l.Error("消息处理 panic",
						logger.String("consumer", msg.ConsumerName),
						logger.Any("panic", r))
					if e, ok := r.(error); ok {
						err = e
					} else {
						err = &PanicError{Value: r}
					}
				}
			}()

			return next(ctx, msg)
		}
	}
}

// PanicError panic 错误
type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	return "panic recovered"
}

