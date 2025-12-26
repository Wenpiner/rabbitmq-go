package rabbitmq

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

func (g *RabbitMQ) handler(key string, d amqp.Delivery) {
	if d.Headers == nil {
		d.Headers = make(amqp.Table)
	}

	// 创建带超时的 context
	timeout := g.getHandlerTimeout(key)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 从 headers 中提取追踪信息
	traceInfo := tracing.ExtractFromHeaders(d.Headers)

	// 如果没有 trace ID，生成新的
	if traceInfo.TraceID == "" {
		traceInfo.TraceID = tracing.GenerateTraceID()
	}

	// 生成新的 span ID（当前处理的 span）
	traceInfo.ParentSpanID = traceInfo.SpanID
	traceInfo.SpanID = tracing.GenerateSpanID()

	// 将追踪信息注入到 context
	ctx = tracing.InjectToContext(ctx, traceInfo)

	// 记录追踪日志
	g.logger.Debug(tracing.FormatTraceLog(ctx, "开始处理消息 (key: "+key+")"))

	// Get current retry count from headers
	retryNum, ok := d.Headers["retry_nums"].(int32)
	if !ok {
		retryNum = int32(0)
	}

	// Store channel reference to check if it's still valid before ACK
	channel := g.channels[key]

	// Record first attempt time if not exists
	if _, exists := d.Headers["first_attempt_time"]; !exists {
		d.Headers["first_attempt_time"] = time.Now().Unix()
	}

	// Process message - 检测接口类型
	var err error
	if receiverWithCtx, ok := g.receivers[key].(conf.ReceiveWithContext); ok {
		// 使用新接口
		err = receiverWithCtx.Receive(ctx, key, d)
	} else {
		// 使用旧接口（兼容模式）
		err = g.receivers[key].(conf.Receive).Receive(key, d)
	}

	if err != nil {
		// 检查是否是 context 超时错误
		if errors.Is(err, context.DeadlineExceeded) {
			g.logger.Warn("消息处理超时", logger.String("key", key), logger.Duration("timeout", timeout))
		}

		// Get retry strategy for this consumer
		strategy := g.getRetryStrategy(key)

		// Check if should retry
		if strategy.ShouldRetry(retryNum, err) {
			// Calculate delay for next retry
			delay := strategy.CalculateDelay(retryNum)
			delayMs := int32(delay.Milliseconds())

			// Update retry metadata in headers
			d.Headers["retry_nums"] = retryNum + 1
			d.Headers["retry_delay"] = delayMs
			d.Headers["last_error"] = err.Error()

			// Send message to delay queue for retry
			// 传递 context，实现级联超时控制
			e := g.SendDelayMsgByKey(ctx, key, d, delayMs)
			if e != nil {
				g.logger.Error("消息进入ttl延时队列失败",
					logger.String("key", key),
					logger.Int32("retry", retryNum+1),
					logger.Duration("delay", delay),
					logger.Error(e))
			} else {
				g.logger.Info("消息重试已调度",
					logger.String("key", key),
					logger.Int32("retry", retryNum+1),
					logger.Duration("delay", delay))
			}
		} else {
			// Max retries reached or retry disabled, invoke exception handler
			g.logger.Warn("消息达到最大重试次数或重试已禁用，进入异常环节",
				logger.String("key", key),
				logger.Int32("retry", retryNum))
			funcName(key, g, d, channel)

			// 调用异常处理 - 检测接口类型
			if receiverWithCtx, ok := g.receivers[key].(conf.ReceiveWithContext); ok {
				receiverWithCtx.Exception(ctx, key, err, d)
			} else {
				g.receivers[key].(conf.Receive).Exception(key, err, d)
			}
		}
	} else {
		// Message processed successfully
		funcName(key, g, d, channel)
	}
}
