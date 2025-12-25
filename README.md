# rabbitmq-go

A flexible and feature-rich RabbitMQ client for Go with built-in retry mechanisms.

## Features

- ✅ **Automatic Reconnection** - Handles connection failures gracefully
- ✅ **Flexible Retry Strategies** - Exponential backoff, linear backoff, or custom strategies
- ✅ **QoS Support** - Fine-grained control over message prefetching
- ✅ **Backward Compatible** - Seamless upgrade from older versions
- ✅ **Type Safe** - Leverages Go's type system for safer code
- ✅ **Production Ready** - Battle-tested in production environments

## Install

```bash
go get github.com/wenpiner/rabbitmq-go
```

## Quick Start

```go
package rabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
	"log"
	"testing"
)

type TestReceive struct {
}

func (t *TestReceive) Receive(key string, message amqp.Delivery) error {
	return nil
}

func (t *TestReceive) Exception(key string, err error, message amqp.Delivery) {

}

func TestConn(t *testing.T) {
	rabbit := NewRabbitMQ(
		conf.RabbitConf{
			Scheme:   "amqp",
			Username: "guest",
			Password: "guest",
			Host:     "127.0.0.1",
			Port:     5671,
			VHost:    "/",
		},
	)
	err := rabbit.register(
		"test", conf.ConsumerConf{
			Exchange:  conf.NewFanoutExchange("test"),
			Queue:     conf.NewQueue("test"),
			RouteKey:  "",
			Name:      "test",
			AutoAck:   false,
			NoLocal:   false,
			NoWait:    false,
			Exclusive: false,
		}, &TestReceive{},
	)
	if err != nil {
		log.Println("rabbitmq connect fail")
		return
	}

	message, err := rabbit.SendMessage(
		context.Background(), "test", "", true, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Hello World!"),
		},
	)
	if err != nil {
		log.Println("rabbitmq send message fail")
		return
	}
	log.Println(message)
}

```

## Retry Mechanism

### Default Exponential Backoff

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    AutoAck:  false,
    Retry:    conf.NewRetryConf(), // Default: 5 retries, exponential backoff
}, receiver)
```

**Retry timeline:** ~1s → ~2s → ~4s → ~8s → ~16s

### Custom Retry Configuration

```go
// Exponential backoff with custom parameters
Retry: conf.NewExponentialRetryConf(10, 500, 1.5)
// 10 retries, 500ms initial delay, 1.5x multiplier

// Linear backoff
Retry: conf.NewLinearRetryConf(3, 2000)
// 3 retries, 2s interval

// Disable retry
Retry: conf.RetryConf{Enable: false}
```

### Advanced: Custom Retry Strategy

```go
type MyReceiver struct {}

func (r *MyReceiver) Receive(key string, message amqp.Delivery) error {
    // Your business logic
    return nil
}

func (r *MyReceiver) Exception(key string, err error, message amqp.Delivery) {
    // Handle max retries exceeded
}

func (r *MyReceiver) GetRetryStrategy() conf.RetryStrategy {
    return conf.NewExponentialRetry(15, 100, 2.0, 3600000, true)
}
```

📖 **For detailed retry documentation, see [RETRY.md](RETRY.md)**

## QoS Configuration

```go
rabbit.Register("my-consumer", conf.ConsumerConf{
    Exchange: conf.NewFanoutExchange("my-exchange"),
    Queue:    conf.NewQueue("my-queue"),
    Qos:      conf.NewQos(10), // Prefetch 10 messages
    AutoAck:  false,
}, receiver)
```

## Examples

See the [examples](examples/) directory for more usage examples:
- [retry_example.go](examples/retry_example.go) - Comprehensive retry examples

## Documentation

- [Retry Mechanism Guide](RETRY.md) - Detailed retry configuration and best practices

## License

MIT

```
