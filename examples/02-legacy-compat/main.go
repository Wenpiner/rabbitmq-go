package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

// 旧接口实现（仍然可以工作）
type LegacyReceiver struct{}

func (r *LegacyReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[LegacyReceiver] Processing message: %s", string(message.Body))
	return nil
}

func (r *LegacyReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[LegacyReceiver] Exception: %v", err)
}

func main() {
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// 旧代码无需修改，仍然可以正常工作
	err := rabbit.Register("legacy-example", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("legacy-queue"),
		Name:     "legacy-consumer",
		AutoAck:  false,
	}, &LegacyReceiver{})

	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	rabbit.Start()
	select {}
}
