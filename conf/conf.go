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
