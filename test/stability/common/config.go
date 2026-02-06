package common

import (
	"os"
	"strconv"
	"time"

	"github.com/wenpiner/rabbitmq-go/v2/conf"
)

// TestConfig 测试配置
type TestConfig struct {
	RabbitMQ      conf.RabbitConf
	TestDuration  time.Duration
	MessageRate   int
	ConsumerCount int
	BatchSize     int
	MetricsAddr   string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *TestConfig {
	return &TestConfig{
		RabbitMQ: conf.RabbitConf{
			Scheme:   "amqp",
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			Username: getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASS", "guest"),
			VHost:    "/",
		},
		TestDuration:  parseDuration(getEnv("TEST_DURATION", "1h")),
		MessageRate:   getEnvInt("MESSAGE_RATE", 100),
		ConsumerCount: getEnvInt("CONSUMER_COUNT", 10),
		BatchSize:     getEnvInt("BATCH_SIZE", 10),
		MetricsAddr:   getEnv("METRICS_ADDR", ":8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 1 * time.Hour
	}
	return d
}

