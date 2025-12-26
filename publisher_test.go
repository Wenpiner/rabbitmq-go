package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/tracing"
)

// getTestRabbitMQ 创建测试用的 RabbitMQ 实例
func getTestRabbitMQ(t *testing.T) *RabbitMQ {
	rabbit := NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 测试连接
	if rabbit.IsClose() {
		if err := rabbit.connect(); err != nil {
			t.Skipf("Skipping test: RabbitMQ not available: %v", err)
		}
	}

	return rabbit
}

// TestWithPublisher 测试 WithPublisher 基本功能
func TestWithPublisher(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 测试成功场景
	err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 验证 channel 不为 nil
		if ch == nil {
			return errors.New("channel is nil")
		}
		// 验证 channel 未关闭
		if ch.IsClosed() {
			return errors.New("channel is closed")
		}
		return nil
	})

	if err != nil {
		t.Errorf("WithPublisher failed: %v", err)
	}
}

// TestWithPublisherChannelClose 测试 channel 自动关闭
func TestWithPublisherChannelClose(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()
	var capturedChannel *amqp.Channel

	// 捕获 channel 引用
	err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		capturedChannel = ch
		return nil
	})

	if err != nil {
		t.Errorf("WithPublisher failed: %v", err)
	}

	// 验证 channel 在函数返回后被关闭
	if capturedChannel != nil && !capturedChannel.IsClosed() {
		t.Error("Channel should be closed after WithPublisher returns")
	}
}

// TestWithPublisherUserError 测试用户函数返回错误
func TestWithPublisherUserError(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()
	expectedErr := errors.New("user error")

	err := rabbit.WithPublisher(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

// TestPublish 测试单条消息发送
func TestPublish(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 发送测试消息
	err := rabbit.Publish(ctx, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test Message"),
	})

	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}
}

// TestPublishWithTimeout 测试带超时的发送
func TestPublishWithTimeout(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rabbit.Publish(ctx, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test Message with Timeout"),
	})

	if err != nil {
		t.Errorf("Publish with timeout failed: %v", err)
	}
}

// TestPublishBatch 测试批量发送
func TestPublishBatch(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 准备批量消息
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Message 1")},
		{ContentType: "text/plain", Body: []byte("Message 2")},
		{ContentType: "text/plain", Body: []byte("Message 3")},
	}

	err := rabbit.PublishBatch(ctx, "", "test-queue", messages)

	if err != nil {
		t.Errorf("PublishBatch failed: %v", err)
	}
}

// TestPublishBatchEmpty 测试空消息列表
func TestPublishBatchEmpty(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 空消息列表应该成功返回
	err := rabbit.PublishBatch(ctx, "", "test-queue", []amqp.Publishing{})

	if err != nil {
		t.Errorf("PublishBatch with empty messages should succeed, got error: %v", err)
	}
}

// TestPublishBatchWithCancel 测试批量发送时的取消
func TestPublishBatchWithCancel(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	// 准备大量消息
	messages := make([]amqp.Publishing, 100)
	for i := 0; i < 100; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Message"),
		}
	}

	// 立即取消 context
	cancel()

	err := rabbit.PublishBatch(ctx, "", "test-queue", messages)

	// 应该返回取消错误
	if err == nil {
		t.Error("PublishBatch should fail when context is cancelled")
	}
}

// TestPublishBatchLarge 测试大批量发送
func TestPublishBatchLarge(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 准备1000条消息
	messages := make([]amqp.Publishing, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Large batch message"),
		}
	}

	err := rabbit.PublishBatch(ctx, "", "test-queue", messages)

	if err != nil {
		t.Errorf("PublishBatch with large batch failed: %v", err)
	}
}

// BenchmarkPublish 基准测试：单条发送
func BenchmarkPublish(b *testing.B) {
	rabbit := NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	if rabbit.IsClose() {
		if err := rabbit.connect(); err != nil {
			b.Skipf("Skipping benchmark: RabbitMQ not available: %v", err)
		}
	}
	defer rabbit.Stop()

	ctx := context.Background()
	msg := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Benchmark Message"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rabbit.Publish(ctx, "", "bench-queue", msg)
	}
}

// BenchmarkPublishBatch 基准测试：批量发送
func BenchmarkPublishBatch(b *testing.B) {
	rabbit := NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	if rabbit.IsClose() {
		if err := rabbit.connect(); err != nil {
			b.Skipf("Skipping benchmark: RabbitMQ not available: %v", err)
		}
	}
	defer rabbit.Stop()

	ctx := context.Background()
	messages := make([]amqp.Publishing, 100)
	for i := 0; i < 100; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Benchmark Batch Message"),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rabbit.PublishBatch(ctx, "", "bench-queue", messages)
	}
}

// TestPublishWithTrace 测试追踪集成
func TestPublishWithTrace(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 测试 1: 自动生成追踪信息
	err := rabbit.PublishWithTrace(ctx, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test Message with Trace"),
	})
	if err != nil {
		t.Errorf("PublishWithTrace failed: %v", err)
	}

	// 测试 2: 使用已有的追踪信息
	traceInfo := tracing.TraceInfo{
		TraceID: "test-trace-id",
		SpanID:  "test-span-id",
	}
	ctxWithTrace := tracing.InjectToContext(ctx, traceInfo)

	err = rabbit.PublishWithTrace(ctxWithTrace, "", "test-queue", amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test Message with Existing Trace"),
	})
	if err != nil {
		t.Errorf("PublishWithTrace with existing trace failed: %v", err)
	}
}

// TestPublishBatchWithTrace 测试批量追踪
func TestPublishBatchWithTrace(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 准备批量消息
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Trace Message 1")},
		{ContentType: "text/plain", Body: []byte("Trace Message 2")},
		{ContentType: "text/plain", Body: []byte("Trace Message 3")},
	}

	err := rabbit.PublishBatchWithTrace(ctx, "", "test-queue", messages)
	if err != nil {
		t.Errorf("PublishBatchWithTrace failed: %v", err)
	}
}

// TestPublishBatchWithTraceEmpty 测试空消息列表
func TestPublishBatchWithTraceEmpty(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	err := rabbit.PublishBatchWithTrace(ctx, "", "test-queue", []amqp.Publishing{})
	if err != nil {
		t.Errorf("PublishBatchWithTrace with empty messages should succeed, got error: %v", err)
	}
}

// TestWithPublisherConfirm 测试 Confirm 模式
func TestWithPublisherConfirm(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	err := rabbit.WithPublisherConfirm(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 创建确认通道
		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

		// 发送消息
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Test Confirm Message"),
		}
		if err := ch.PublishWithContext(ctx, "", "test-queue", false, false, msg); err != nil {
			return err
		}

		// 等待确认
		select {
		case confirm := <-confirms:
			if !confirm.Ack {
				return errors.New("message not confirmed")
			}
		case <-time.After(5 * time.Second):
			return errors.New("confirmation timeout")
		}

		return nil
	})

	if err != nil {
		t.Errorf("WithPublisherConfirm failed: %v", err)
	}
}

// TestPublishBatchWithConfirm 测试批量确认
func TestPublishBatchWithConfirm(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 准备批量消息
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Confirm Message 1")},
		{ContentType: "text/plain", Body: []byte("Confirm Message 2")},
		{ContentType: "text/plain", Body: []byte("Confirm Message 3")},
	}

	err := rabbit.PublishBatchWithConfirm(ctx, "", "test-queue", messages)
	if err != nil {
		t.Errorf("PublishBatchWithConfirm failed: %v", err)
	}
}

// TestPublishBatchWithConfirmEmpty 测试空消息列表
func TestPublishBatchWithConfirmEmpty(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	err := rabbit.PublishBatchWithConfirm(ctx, "", "test-queue", []amqp.Publishing{})
	if err != nil {
		t.Errorf("PublishBatchWithConfirm with empty messages should succeed, got error: %v", err)
	}
}

// TestPublishBatchWithConfirmTimeout 测试确认超时
func TestPublishBatchWithConfirmTimeout(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	// 创建一个很短的超时 context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 等待 context 超时
	time.Sleep(10 * time.Millisecond)

	messages := []amqp.Publishing{
		{Body: []byte("Message 1")},
	}

	err := rabbit.PublishBatchWithConfirm(ctx, "", "test-queue", messages)
	if err == nil {
		t.Error("PublishBatchWithConfirm should fail with timeout")
	}
}

// TestWithPublisherTx 测试事务模式
func TestWithPublisherTx(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 测试 1: 成功提交
	err := rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 发送多条消息
		for i := 0; i < 3; i++ {
			msg := amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("Tx Message %d", i+1)),
			}
			if err := ch.PublishWithContext(ctx, "", "test-queue", false, false, msg); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithPublisherTx should succeed, got error: %v", err)
	}

	// 测试 2: 自动回滚
	err = rabbit.WithPublisherTx(ctx, func(ctx context.Context, ch *amqp.Channel) error {
		// 发送一条消息
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Message before error"),
		}
		if err := ch.PublishWithContext(ctx, "", "test-queue", false, false, msg); err != nil {
			return err
		}

		// 返回错误，触发回滚
		return errors.New("simulated error")
	})
	if err == nil {
		t.Error("WithPublisherTx should fail and rollback")
	}
	if !errors.Is(err, errors.New("simulated error")) && err.Error() != "transaction rolled back: simulated error" {
		t.Errorf("Expected rollback error, got: %v", err)
	}
}

// TestPublishBatchTx 测试事务批量发送
func TestPublishBatchTx(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 准备批量消息
	messages := []amqp.Publishing{
		{ContentType: "text/plain", Body: []byte("Tx Batch Message 1")},
		{ContentType: "text/plain", Body: []byte("Tx Batch Message 2")},
		{ContentType: "text/plain", Body: []byte("Tx Batch Message 3")},
	}

	err := rabbit.PublishBatchTx(ctx, "", "test-queue", messages)
	if err != nil {
		t.Errorf("PublishBatchTx failed: %v", err)
	}
}

// TestPublishBatchTxEmpty 测试空消息列表
func TestPublishBatchTxEmpty(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	err := rabbit.PublishBatchTx(ctx, "", "test-queue", []amqp.Publishing{})
	if err != nil {
		t.Errorf("PublishBatchTx with empty messages should succeed, got error: %v", err)
	}
}

// TestBatchPublisher 测试批量发送器
func TestBatchPublisher(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 创建批量发送器
	publisher := rabbit.NewBatchPublisher("", "test-queue")

	// 添加消息
	for i := 0; i < 5; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Batch Publisher Message %d", i+1)),
		}
		if err := publisher.Add(ctx, msg); err != nil {
			t.Errorf("Failed to add message: %v", err)
		}
	}

	// 检查消息数量
	if publisher.Len() != 5 {
		t.Errorf("Expected 5 messages, got %d", publisher.Len())
	}

	// 刷新消息
	if err := publisher.Flush(ctx); err != nil {
		t.Errorf("Failed to flush: %v", err)
	}

	// 检查消息已清空
	if publisher.Len() != 0 {
		t.Errorf("Expected 0 messages after flush, got %d", publisher.Len())
	}
}

// TestBatchPublisherAutoFlush 测试自动刷新
func TestBatchPublisherAutoFlush(t *testing.T) {
	rabbit := getTestRabbitMQ(t)
	defer rabbit.Stop()

	ctx := context.Background()

	// 创建批量发送器，设置批次大小为 3，启用自动刷新
	publisher := rabbit.NewBatchPublisher("", "test-queue").
		SetBatchSize(3).
		SetAutoFlush(true)

	// 添加 2 条消息，不应触发刷新
	for i := 0; i < 2; i++ {
		msg := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("Auto Flush Message %d", i+1)),
		}
		if err := publisher.Add(ctx, msg); err != nil {
			t.Errorf("Failed to add message: %v", err)
		}
	}

	// 应该还有 2 条消息
	if publisher.Len() != 2 {
		t.Errorf("Expected 2 messages, got %d", publisher.Len())
	}

	// 添加第 3 条消息，应触发自动刷新
	msg := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Auto Flush Message 3"),
	}
	if err := publisher.Add(ctx, msg); err != nil {
		t.Errorf("Failed to add message: %v", err)
	}

	// 应该已经自动刷新，消息数量为 0
	if publisher.Len() != 0 {
		t.Errorf("Expected 0 messages after auto flush, got %d", publisher.Len())
	}
}
