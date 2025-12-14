# rabbitmq-go

## Install

```bash
go get github.com/wenpiner/rabbitmq-go
```

## Usage

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

## Channel Management

The library provides channel management functions to help troubleshoot channel issues:

### LogUnclosedChannels

Print status of all channels to logs for troubleshooting:

```go
// Log all channel status (open/closed) to help debug unclosed channels
rabbit.LogUnclosedChannels()
```

### GetChannelStatus

Get programmatic access to channel status:

```go
// Get status information for all registered channels
channelStatus := rabbit.GetChannelStatus()
for _, info := range channelStatus {
    if !info.IsClosed {
        log.Printf("Channel %s is still open", info.Key)
    }
}
```


