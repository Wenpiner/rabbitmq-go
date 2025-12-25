package main

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "github.com/wenpiner/rabbitmq-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

// Example 1: Using default exponential backoff retry
type DefaultRetryReceiver struct {
	processCount int
}

func (r *DefaultRetryReceiver) Receive(key string, message amqp.Delivery) error {
	r.processCount++
	log.Printf("[DefaultRetry] Processing message (attempt %d): %s", r.processCount, string(message.Body))
	
	// Simulate failure for first 2 attempts
	if r.processCount < 3 {
		return fmt.Errorf("simulated error on attempt %d", r.processCount)
	}
	
	log.Printf("[DefaultRetry] Message processed successfully!")
	return nil
}

func (r *DefaultRetryReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[DefaultRetry] Exception handler called: %v", err)
}

// Example 2: Using custom exponential backoff configuration
type CustomExponentialReceiver struct{}

func (r *CustomExponentialReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[CustomExponential] Processing message: %s", string(message.Body))
	// Your business logic here
	return nil
}

func (r *CustomExponentialReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[CustomExponential] Exception: %v", err)
}

// Example 3: Using linear backoff retry
type LinearRetryReceiver struct{}

func (r *LinearRetryReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[LinearRetry] Processing message: %s", string(message.Body))
	// Your business logic here
	return nil
}

func (r *LinearRetryReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[LinearRetry] Exception: %v", err)
}

// Example 4: Using custom retry strategy via interface
type AdvancedReceiver struct {
	customStrategy conf.RetryStrategy
}

func (r *AdvancedReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[Advanced] Processing message: %s", string(message.Body))
	// Your business logic here
	return nil
}

func (r *AdvancedReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[Advanced] Exception: %v", err)
}

func (r *AdvancedReceiver) GetRetryStrategy() conf.RetryStrategy {
	return r.customStrategy
}

// Example 5: Disabling retry
type NoRetryReceiver struct{}

func (r *NoRetryReceiver) Receive(key string, message amqp.Delivery) error {
	log.Printf("[NoRetry] Processing message: %s", string(message.Body))
	// Your business logic here
	return nil
}

func (r *NoRetryReceiver) Exception(key string, err error, message amqp.Delivery) {
	log.Printf("[NoRetry] Exception: %v", err)
}

func main() {
	// Initialize RabbitMQ connection
	rabbit := rabbitmq.NewRabbitMQ(conf.RabbitConf{
		Scheme:   "amqp",
		Username: "guest",
		Password: "guest",
		Host:     "127.0.0.1",
		Port:     5672,
		VHost:    "/",
	})

	// Example 1: Default exponential backoff (5 retries, 1s initial, 2x multiplier)
	err := rabbit.Register("default-retry", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("default-retry-queue"),
		RouteKey: "",
		Name:     "default-retry-consumer",
		AutoAck:  false,
		Retry:    conf.NewRetryConf(), // Uses default exponential backoff
	}, &DefaultRetryReceiver{})
	if err != nil {
		log.Printf("Failed to register default-retry consumer: %v", err)
	}

	// Example 2: Custom exponential backoff (10 retries, 500ms initial, 1.5x multiplier)
	err = rabbit.Register("custom-exponential", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("custom-exp-queue"),
		RouteKey: "",
		Name:     "custom-exp-consumer",
		AutoAck:  false,
		Retry:    conf.NewExponentialRetryConf(10, 500, 1.5),
	}, &CustomExponentialReceiver{})
	if err != nil {
		log.Printf("Failed to register custom-exponential consumer: %v", err)
	}

	// Example 3: Linear backoff (3 retries, 2s initial delay)
	err = rabbit.Register("linear-retry", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("linear-retry-queue"),
		RouteKey: "",
		Name:     "linear-retry-consumer",
		AutoAck:  false,
		Retry:    conf.NewLinearRetryConf(3, 2000),
	}, &LinearRetryReceiver{})
	if err != nil {
		log.Printf("Failed to register linear-retry consumer: %v", err)
	}

	// Example 4: Custom retry strategy via interface
	customStrategy := conf.NewExponentialRetry(7, 1000, 3.0, 120000, true)
	err = rabbit.Register("advanced-retry", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("advanced-queue"),
		RouteKey: "",
		Name:     "advanced-consumer",
		AutoAck:  false,
		// Retry config will be ignored because receiver implements GetRetryStrategy
	}, &AdvancedReceiver{customStrategy: customStrategy})
	if err != nil {
		log.Printf("Failed to register advanced-retry consumer: %v", err)
	}

	// Example 5: Disable retry
	err = rabbit.Register("no-retry", conf.ConsumerConf{
		Exchange: conf.NewFanoutExchange("example"),
		Queue:    conf.NewQueue("no-retry-queue"),
		RouteKey: "",
		Name:     "no-retry-consumer",
		AutoAck:  false,
		Retry: conf.RetryConf{
			Enable: false, // Disable retry
		},
	}, &NoRetryReceiver{})
	if err != nil {
		log.Printf("Failed to register no-retry consumer: %v", err)
	}

	// Start RabbitMQ
	rabbit.Start()

	// Send a test message
	time.Sleep(2 * time.Second)
	_, err = rabbit.SendMessage(context.Background(), "example", "", true, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte("Test message with retry"),
	})
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}

	// Keep running
	log.Println("Examples running... Press Ctrl+C to exit")
	select {}
}

