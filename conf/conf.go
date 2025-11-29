package conf

import amqp "github.com/rabbitmq/amqp091-go"

type QueueConf struct {
	Name       string
	Durable    bool `json:",default=true"`
	AutoDelete bool `json:",default=false"`
	Exclusive  bool `json:",default=false"`
	NoWait     bool `json:",default=false"`
}

type ExchangeConf struct {
	ExchangeName string
	Type         string `json:",options=direct|fanout|topic|headers"` // exchange type
	Durable      bool   `json:",default=true"`
	AutoDelete   bool   `json:",default=false"`
	Internal     bool   `json:",default=false"`
	NoWait       bool   `json:",default=false"`
	Queues       []QueueConf
}

func NewFanoutExchange(exchange string) ExchangeConf {
	return ExchangeConf{
		ExchangeName: exchange,
		Type:         "fanout",
		Durable:      true,
		AutoDelete:   false,
		Internal:     false,
		NoWait:       false,
	}
}

func NewDirectExchange(exchange string) ExchangeConf {
	return ExchangeConf{
		ExchangeName: exchange,
		Type:         "direct",
		Durable:      true,
		AutoDelete:   false,
		Internal:     false,
		NoWait:       false,
	}
}

func NewQueue(queue string) QueueConf {
	return QueueConf{
		Name:       queue,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	}
}
func NewQueueAutoDelete(queue string) QueueConf {
	return QueueConf{
		Name:       queue,
		Durable:    true,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
	}
}

type ConsumerConf struct {
	Exchange ExchangeConf
	Queue    QueueConf
	RouteKey string
	Name     string
	AutoAck  bool `json:",default=true"`
	// Set to true, which means that messages sent by producers in the same connection
	// cannot be delivered to consumers in this connection.
	NoLocal bool `json:",default=false"`
	// Whether to block processing
	NoWait    bool `json:",default=false"`
	Exclusive bool `json:",default=false"`
	// QoS configuration for the consumer
	Qos QosConf `json:",optional"`
}

// QosConf contains QoS (Quality of Service) settings for consumers
type QosConf struct {
	// PrefetchCount specifies how many messages the server will deliver before
	// requiring acknowledgements. A value of 0 means no limit.
	PrefetchCount int `json:",default=0"`
	// PrefetchSize specifies the prefetch window size in bytes. A value of 0 means no limit.
	PrefetchSize int `json:",default=0"`
	// Global specifies whether the QoS settings should be applied globally (true)
	// or per-consumer (false).
	Global bool `json:",default=false"`
	// Enable specifies whether to enable QoS settings. If false, QoS will not be applied.
	Enable bool `json:",default=false"`
}

// NewQos creates a new QoS configuration with the specified prefetch count.
// The prefetchCount specifies how many messages the server will deliver before
// requiring acknowledgements.
func NewQos(prefetchCount int) QosConf {
	return QosConf{
		PrefetchCount: prefetchCount,
		PrefetchSize:  0,
		Global:        false,
		Enable:        true,
	}
}

// NewQosWithSize creates a new QoS configuration with the specified prefetch count and size.
func NewQosWithSize(prefetchCount, prefetchSize int) QosConf {
	return QosConf{
		PrefetchCount: prefetchCount,
		PrefetchSize:  prefetchSize,
		Global:        false,
		Enable:        true,
	}
}

// NewGlobalQos creates a new global QoS configuration with the specified prefetch count.
// Global QoS applies to all consumers on the channel.
func NewGlobalQos(prefetchCount int) QosConf {
	return QosConf{
		PrefetchCount: prefetchCount,
		PrefetchSize:  0,
		Global:        true,
		Enable:        true,
	}
}

type RabbitConf struct {
	Scheme   string `json:",default=amqp,options=amqp|amqps"`
	Username string
	Password string
	Host     string
	Port     int
	VHost    string `json:",optional"`
}

type Receive interface {
	// Receive 消息接收
	Receive(key string, message amqp.Delivery) error
	// Exception 异常处理
	Exception(key string, err error, message amqp.Delivery)
}
