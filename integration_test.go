package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/v2/conf"
	"github.com/wenpiner/rabbitmq-go/v2/logger"
)

// 测试配置 - 根据你的 Docker RabbitMQ 配置修改
var testConfig = conf.RabbitConf{
	Scheme:   "amqp",
	Host:     "localhost",
	Port:     5672,
	Username: "guest",
	Password: "guest",
	VHost:    "/",
}

// TestHandler 测试消息处理器
type TestHandler struct {
	mu            sync.Mutex
	receivedCount int32
	messages      []string
	onHandle      func(ctx context.Context, msg *Message) error
	onError       func(ctx context.Context, msg *Message, err error)
}

func NewTestHandler() *TestHandler {
	return &TestHandler{
		messages: make([]string, 0),
	}
}

func (h *TestHandler) Handle(ctx context.Context, msg *Message) error {
	atomic.AddInt32(&h.receivedCount, 1)
	
	h.mu.Lock()
	h.messages = append(h.messages, string(msg.Body()))
	h.mu.Unlock()

	if h.onHandle != nil {
		return h.onHandle(ctx, msg)
	}
	return nil
}

func (h *TestHandler) OnError(ctx context.Context, msg *Message, err error) {
	if h.onError != nil {
		h.onError(ctx, msg, err)
	}
}

func (h *TestHandler) GetReceivedCount() int32 {
	return atomic.LoadInt32(&h.receivedCount)
}

func (h *TestHandler) GetMessages() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.messages))
	copy(result, h.messages)
	return result
}

// 测试基本连接
func TestIntegration_BasicConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("客户端应该处于连接状态")
	}

	t.Logf("✅ 连接成功，状态: %s", client.State())
}

// 测试发布和消费消息
func TestIntegration_PublishAndConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	// 配置队列和交换机
	queueName := fmt.Sprintf("test_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_exchange_%d", time.Now().Unix())
	routingKey := "test.key"

	// 创建消费者
	handler := NewTestHandler()

	err := client.RegisterConsumer("test-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(true),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	// 等待消费者启动
	time.Sleep(500 * time.Millisecond)

	// 发送测试消息
	testMessages := []string{"message1", "message2", "message3"}
	for _, msg := range testMessages {
		err := client.Publish(ctx, exchangeName, routingKey, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
		if err != nil {
			t.Errorf("发送消息失败: %v", err)
		}
	}

	// 等待消息处理
	time.Sleep(1 * time.Second)

	// 验证接收到的消息
	receivedCount := handler.GetReceivedCount()
	if receivedCount != int32(len(testMessages)) {
		t.Errorf("接收消息数量 = %d, 期望 %d", receivedCount, len(testMessages))
	}

	t.Logf("✅ 成功发送和接收 %d 条消息", receivedCount)
}

// 测试批量发送
func TestIntegration_BatchPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	queueName := fmt.Sprintf("test_batch_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_batch_exchange_%d", time.Now().Unix())
	routingKey := "batch.key"

	handler := NewTestHandler()

	err := client.RegisterConsumer("batch-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(true),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 批量发送消息
	batchSize := 10
	messages := make([]amqp.Publishing, batchSize)
	for i := 0; i < batchSize; i++ {
		messages[i] = amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("batch_message_%d", i)),
		}
	}

	err = client.PublishBatch(ctx, exchangeName, routingKey, messages)
	if err != nil {
		t.Fatalf("批量发送失败: %v", err)
	}

	// 等待消息处理
	time.Sleep(2 * time.Second)

	receivedCount := handler.GetReceivedCount()
	if receivedCount != int32(batchSize) {
		t.Errorf("接收消息数量 = %d, 期望 %d", receivedCount, batchSize)
	}

	t.Logf("✅ 批量发送成功，发送 %d 条，接收 %d 条", batchSize, receivedCount)
}

// 测试 Confirm 模式
func TestIntegration_PublishWithConfirm(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	queueName := fmt.Sprintf("test_confirm_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_confirm_exchange_%d", time.Now().Unix())
	routingKey := "confirm.key"

	// 先绑定队列
	err := client.Bind(
		conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		},
		conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		},
		routingKey,
	)
	if err != nil {
		t.Fatalf("绑定队列失败: %v", err)
	}

	// 发送带确认的消息
	err = client.PublishWithConfirm(ctx, exchangeName, routingKey, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("confirm_message"),
	})
	if err != nil {
		t.Fatalf("发送确认消息失败: %v", err)
	}

	t.Logf("✅ Confirm 模式发送成功")
}

// 测试追踪功能
func TestIntegration_PublishWithTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	queueName := fmt.Sprintf("test_trace_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_trace_exchange_%d", time.Now().Unix())
	routingKey := "trace.key"

	// 创建消费者验证追踪信息
	handler := NewTestHandler()
	handler.onHandle = func(ctx context.Context, msg *Message) error {
		if msg.TraceInfo.TraceID == "" {
			t.Error("TraceID 不应为空")
		}
		if msg.TraceInfo.SpanID == "" {
			t.Error("SpanID 不应为空")
		}
		t.Logf("收到追踪消息: TraceID=%s, SpanID=%s", msg.TraceInfo.TraceID, msg.TraceInfo.SpanID)
		return nil
	}

	err := client.RegisterConsumer("trace-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(true),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 发送带追踪的消息
	err = client.PublishWithTrace(ctx, exchangeName, routingKey, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("trace_message"),
	})
	if err != nil {
		t.Fatalf("发送追踪消息失败: %v", err)
	}

	time.Sleep(1 * time.Second)

	if handler.GetReceivedCount() != 1 {
		t.Errorf("应该接收到 1 条消息，实际接收 %d 条", handler.GetReceivedCount())
	}

	t.Logf("✅ 追踪功能测试成功")
}

// 测试优雅关闭
func TestIntegration_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}

	queueName := fmt.Sprintf("test_shutdown_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_shutdown_exchange_%d", time.Now().Unix())
	routingKey := "shutdown.key"

	handler := NewTestHandler()

	err := client.RegisterConsumer("shutdown-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(true),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 发送一些消息
	for i := 0; i < 5; i++ {
		err := client.Publish(ctx, exchangeName, routingKey, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("shutdown_message_%d", i)),
		})
		if err != nil {
			t.Errorf("发送消息失败: %v", err)
		}
	}

	time.Sleep(1 * time.Second)

	// 优雅关闭
	err = client.Close()
	if err != nil {
		t.Errorf("关闭客户端失败: %v", err)
	}

	if client.IsConnected() {
		t.Error("客户端应该已断开连接")
	}

	t.Logf("✅ 优雅关闭测试成功，处理了 %d 条消息", handler.GetReceivedCount())
}

// 测试消费者管理
func TestIntegration_ConsumerManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	queueName := fmt.Sprintf("test_mgmt_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_mgmt_exchange_%d", time.Now().Unix())
	routingKey := "mgmt.key"

	handler := NewTestHandler()

	// 注册消费者
	err := client.RegisterConsumer("mgmt-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(true),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	// 检查消费者数量
	if client.ConsumerCount() != 1 {
		t.Errorf("消费者数量 = %d, 期望 1", client.ConsumerCount())
	}

	// 获取消费者
	consumer, ok := client.GetConsumer("mgmt-consumer")
	if !ok {
		t.Fatal("应该能获取到消费者")
	}

	if consumer.State() != ConsumerRunning {
		t.Errorf("消费者状态 = %s, 期望 Running", consumer.State())
	}

	// 注销消费者
	err = client.UnregisterConsumer("mgmt-consumer")
	if err != nil {
		t.Errorf("注销消费者失败: %v", err)
	}

	if client.ConsumerCount() != 0 {
		t.Errorf("消费者数量 = %d, 期望 0", client.ConsumerCount())
	}

	t.Logf("✅ 消费者管理测试成功")
}

// 测试 QoS 设置
func TestIntegration_QoS(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	client := New(
		WithConfig(testConfig),
		WithLogger(logger.NewDefaultLogger(logger.LevelInfo)),
	)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	queueName := fmt.Sprintf("test_qos_queue_%d", time.Now().Unix())
	exchangeName := fmt.Sprintf("test_qos_exchange_%d", time.Now().Unix())
	routingKey := "qos.key"

	handler := NewTestHandler()

	err := client.RegisterConsumer("qos-consumer",
		WithQueue(conf.QueueConf{
			Name:       queueName,
			Durable:    false,
			AutoDelete: true,
		}),
		WithExchange(conf.ExchangeConf{
			ExchangeName: exchangeName,
			Type:         "topic",
			Durable:      false,
			AutoDelete:   true,
		}),
		WithRouteKey(routingKey),
		WithAutoAck(false), // 关闭自动确认以测试 QoS
		WithQos(conf.QosConf{
			Enable:        true,
			PrefetchCount: 5,
			PrefetchSize:  0,
			Global:        false,
		}),
		WithHandler(handler),
	)
	if err != nil {
		t.Fatalf("注册消费者失败: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 发送消息
	for i := 0; i < 10; i++ {
		err := client.Publish(ctx, exchangeName, routingKey, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("qos_message_%d", i)),
		})
		if err != nil {
			t.Errorf("发送消息失败: %v", err)
		}
	}

	time.Sleep(2 * time.Second)

	receivedCount := handler.GetReceivedCount()
	if receivedCount != 10 {
		t.Errorf("接收消息数量 = %d, 期望 10", receivedCount)
	}

	t.Logf("✅ QoS 测试成功，接收 %d 条消息", receivedCount)
}

