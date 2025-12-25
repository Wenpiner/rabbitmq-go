package rabbitmq

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (g *RabbitMQ) handler(key string, d amqp.Delivery) {
	if d.Headers == nil {
		d.Headers = make(amqp.Table)
	}

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

	// Process message
	err := g.receivers[key].Receive(key, d)

	if err != nil {
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
			e := g.SendDelayMsgByKey(key, d, delayMs)
			if e != nil {
				log.Printf("消息进入ttl延时队列失败 (key: %s, retry: %d, delay: %v) err: %s\n",
					key, retryNum+1, delay, e)
			} else {
				log.Printf("消息重试已调度 (key: %s, retry: %d, delay: %v)\n",
					key, retryNum+1, delay)
			}
		} else {
			// Max retries reached or retry disabled, invoke exception handler
			log.Printf("消息达到最大重试次数或重试已禁用 (key: %s, retry: %d), 进入异常环节\n", key, retryNum)
			funcName(key, g, d, channel)
			g.receivers[key].Exception(key, err, d)
		}
	} else {
		// Message processed successfully
		funcName(key, g, d, channel)
	}
}
