package rabbitmq

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wenpiner/rabbitmq-go/conf"
	"github.com/wenpiner/rabbitmq-go/logger"
)

// TestWarnDeprecation 测试废弃警告功能
func TestWarnDeprecation(t *testing.T) {
	// 创建一个可以捕获输出的 logger
	var buf bytes.Buffer
	testLogger := logger.NewDefaultLogger(logger.LevelDebug)
	testLogger.SetOutput(&buf)

	rabbit := NewRabbitMQ(conf.RabbitConf{
		Host:     "127.0.0.1",
		Port:     5672,
		Username: "guest",
		Password: "guest",
		VHost:    "/",
	})
	rabbit.SetLogger(testLogger)

	// 测试第一次警告
	buf.Reset()
	rabbit.warnDeprecation("OldAPI()", "NewAPI()", "https://example.com/migration")

	output := buf.String()
	if !strings.Contains(output, "API 已废弃") {
		t.Errorf("Expected 'API 已废弃' in output, got: %s", output)
	}
	if !strings.Contains(output, "OldAPI()") {
		t.Errorf("Expected OldAPI() in output, got: %s", output)
	}
	if !strings.Contains(output, "NewAPI()") {
		t.Errorf("Expected NewAPI() in output, got: %s", output)
	}
	if !strings.Contains(output, "https://example.com/migration") {
		t.Errorf("Expected migration URL in output, got: %s", output)
	}

	// 测试第二次调用同一个 API - 不应该有警告
	buf.Reset()
	rabbit.warnDeprecation("OldAPI()", "NewAPI()", "https://example.com/migration")

	output = buf.String()
	if strings.Contains(output, "API 已废弃") {
		t.Error("Should not show deprecation warning on second call")
	}

	// 测试不同的 API - 应该有警告
	buf.Reset()
	rabbit.warnDeprecation("AnotherOldAPI()", "AnotherNewAPI()", "")

	output = buf.String()
	if !strings.Contains(output, "API 已废弃") {
		t.Errorf("Expected 'API 已废弃' for different API, got: %s", output)
	}
	if !strings.Contains(output, "AnotherOldAPI()") {
		t.Errorf("Expected AnotherOldAPI() in output, got: %s", output)
	}
}
