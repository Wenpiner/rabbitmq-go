package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wenpiner/rabbitmq-go/conf"
)

type RabbitMQ struct {
	rabbitConf conf.RabbitConf
	conn       *amqp.Connection
	// 注册的消费者信息,用于断线后重连
	consumers map[string]conf.ConsumerConf
	// 消费者的RabbitMQ Channel
	channels map[string]*amqp.Channel
	// 消费者回调函数
	receivers map[string]conf.Receive
	// 连接互斥锁
	mu sync.Mutex
	// 链接成功通道
	connected chan interface{}
	// 停止通道
	stop chan interface{}
	// 消费者所有消息通道
	consumer chan amqp.Delivery
	// 当前连接状态
	isClose bool
	// 是否准备停止
	isStop bool
	// 是否已经启动
	isStart bool
}

func (g *RabbitMQ) Channel() (channel *amqp.Channel,err error) {
	if g.IsClose() {
		err = g.connect()
		if err != nil {
			return
		}
	}
	// 声明通道
	if channel == nil || channel.IsClosed() {
		channel, err = g.conn.Channel()
		if err != nil {
			return
		}
	}
	return
}

func (g *RabbitMQ) SendMessageClose(ctx context.Context, exchange, route string, conn bool, msg amqp.Publishing) error {
	channel, err := g.SendMessage(ctx, exchange, route, conn, msg)
	if err != nil {
		return err
	}
	channel.Close()
	return nil
}

func (g *RabbitMQ) SendMessage(ctx context.Context, exchange string, route string, conn bool, msg amqp.Publishing) (
	channel *amqp.Channel, err error,
) {
	if g.IsClose() {
		if conn {
			err = g.connect()
			if err != nil {
				return
			}
		} else {
			err = errors.New("rabbitmq connect fail")
			channel = nil
			return
		}
	}
	// 声明通道
	if channel == nil || channel.IsClosed() {
		channel, err = g.conn.Channel()
		if err != nil {
			return
		}
	}
	err = channel.PublishWithContext(ctx, exchange, route, false, false, msg)
	return
}

func (g *RabbitMQ) Bind(exchange conf.ExchangeConf, queue conf.QueueConf, routeKey string) (err error) {
	if g.IsClose() {
		return errors.New("rabbitmq connect fail")
	}
	// 声明通道
	var channel *amqp.Channel
	channel, err = g.conn.Channel()
	if err != nil {
		return
	}
	defer func(channel *amqp.Channel) {
		_ = channel.Close()
	}(channel)

	var passiveChannel *amqp.Channel
	passiveChannel, err = g.conn.Channel()
	if err != nil {
		return
	}

	// 读取队列
	_, err = passiveChannel.QueueDeclarePassive(
		queue.Name, queue.Durable, queue.AutoDelete, queue.Exclusive, queue.NoWait, nil,
	)
	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			// 队列不存在,声明队列
			_, err = channel.QueueDeclare(
				queue.Name, queue.Durable, queue.AutoDelete, queue.Exclusive, queue.NoWait, nil,
			)
			if err != nil {
				return
			}
		} else {
			err = e
			return
		}
	}
	passiveChannel, err = g.conn.Channel()
	if err != nil {
		return
	}
	// 绑定队列
	err = passiveChannel.ExchangeDeclarePassive(
		exchange.ExchangeName, exchange.Type, exchange.Durable, exchange.AutoDelete, exchange.Internal, exchange.NoWait,
		nil,
	)
	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			// 交换机不存在,声明交换机
			err = channel.ExchangeDeclare(
				exchange.ExchangeName, exchange.Type, exchange.Durable, exchange.AutoDelete, exchange.Internal,
				exchange.NoWait,
				nil,
			)
			if err != nil {
				return err
			}
		}
	}

	// 绑定路由
	err = channel.QueueBind(
		queue.Name, routeKey, exchange.ExchangeName, queue.NoWait, nil,
	)
	return err

}

func NewRabbitMQ(rabbitConf conf.RabbitConf) *RabbitMQ {
	return &RabbitMQ{
		rabbitConf: rabbitConf, isClose: true, consumer: make(chan amqp.Delivery),
		consumers: make(map[string]conf.ConsumerConf),
		channels:  make(map[string]*amqp.Channel),
		receivers: make(map[string]conf.Receive),
		connected: make(chan interface{}),
		stop:      make(chan interface{}),
		isStop:    false,
	}
}

func (g *RabbitMQ) onChan() {
	// 设置心跳
	ticker := time.NewTicker(500 * time.Millisecond)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			// 检查连接是否断开
			if g.IsClose() {
				// 连接断开，进行重连
				go g.connect()
			}
			break
		case <-g.connected:
			// 连接成功，进行消费者注册
			log.Println("连接RabbitMQ成功,进行消费者注册")
			g.registerAll()
			break
		case <-g.stop:
			{
				// 停止
				// 关闭所有消费者
				for _, channel := range g.channels {
					_ = channel.Close()
				}
				_ = g.conn.Close()
			}
			return
		}
	}
}

func (g *RabbitMQ) registerAll() {
	for key, consumer := range g.consumers {
		e := g.register(key, consumer, g.receivers[key])
		if e != nil {
			log.Println("注册消费者失败:", e)
		}
	}
}

func (g *RabbitMQ) register(key string, consumer conf.ConsumerConf, receiver conf.Receive) error {
	// 判断是否存在消费者
	if v, ok := g.channels[key]; ok && v != nil && !v.IsClosed() {
		return nil
	}
	g.channels[key] = nil
	g.receivers[key] = receiver
	g.consumers[key] = consumer
	if g.IsClose() {
		if !g.isStart {
			return nil
		}
		log.Println("rabbitmq connect fail")
		return errors.New("rabbitmq connect fail")
	}
	// 创建消费者通道
	channel, err := g.conn.Channel()
	if err != nil {
		log.Println("Failed to open a channel: ", err)
		return err
	} else {
		log.Println("rabbitmq channel connect success")
	}
	g.channels[key] = channel
	// 如果存在队列，则先进行队列声明
	if consumer.Exchange.ExchangeName != "" && consumer.Queue.Name != "" {
		err = g.Bind(consumer.Exchange, consumer.Queue, consumer.RouteKey)
		if err != nil {
			log.Println("绑定队列失败:", err)
		}
	}
	// 注册消费者
	msgs, err := g.channels[key].Consume(
		consumer.Queue.Name,
		consumer.Name,
		consumer.AutoAck,
		consumer.Exclusive,
		consumer.NoLocal,
		consumer.NoWait,
		nil,
	)
	if err != nil {
		log.Println("Failed to register a consumer:", err)
		return err
	}
	go func(k string) {
		for d := range msgs {
			go g.handler(k, d)
		}
		if !g.isStop {
			// 重连
			go g.connect()
		}
	}(key)
	return nil
}

// SendDelayMsg 发送延时消息
func (g *RabbitMQ) SendDelayMsg(key string, msg amqp.Delivery, delay int32) error {
	if g.IsClose() {
		err := g.connect()
		if err != nil {
			return err
		}
	}
	// 声明通道
	var err error
	var channel *amqp.Channel
	channel, err = g.conn.Channel()

	queueName := g.consumers[key].Queue.Name + "_delay"
	// 声明延时队列
	_, err = channel.QueueDeclarePassive(
		queueName, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    g.consumers[key].Exchange.ExchangeName,
			"x-dead-letter-routing-key": g.consumers[key].RouteKey,
		},
	)

	if err != nil {
		var e *amqp.Error
		if errors.As(err, &e) && e.Code == amqp.NotFound {
			channel, err = g.conn.Channel()
			if err != nil {
				return err
			}
			_, err = channel.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				amqp.Table{
					"x-dead-letter-exchange":    g.consumers[key].Exchange.ExchangeName,
					"x-dead-letter-routing-key": g.consumers[key].RouteKey,
				},
			)
			if err != nil {
				return err
			}
		} else {
			log.Println("Failed to open a channel: ", err)
			return e
		}
	}
	defer func(channel *amqp.Channel) {
		err := channel.Close()
		if err != nil {
			log.Printf("关闭channel失败 err :%s \n", err)
		}
	}(channel)

	// 发送消息
	err = channel.PublishWithContext(
		context.Background(),
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			Headers:         msg.Headers,
			ContentType:     msg.ContentType,
			ContentEncoding: msg.ContentEncoding,
			DeliveryMode:    msg.DeliveryMode,
			Priority:        msg.Priority,
			CorrelationId:   msg.CorrelationId,
			ReplyTo:         msg.ReplyTo,
			MessageId:       msg.MessageId,
			Type:            msg.Type,
			UserId:          msg.UserId,
			AppId:           msg.AppId,
			Body:            msg.Body,
			Expiration:      strconv.Itoa(int(delay)),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (g *RabbitMQ) handler(key string, d amqp.Delivery) {
	if d.Headers == nil {
		d.Headers = make(amqp.Table)
	}
	retryNum, ok := d.Headers["retry_nums"].(int32)
	if !ok {
		retryNum = int32(0)
	}
	err := g.receivers[key].Receive(key, d)
	if err != nil {
		if retryNum < 3 {
			d.Headers["retry_nums"] = retryNum + 1
			e := g.SendDelayMsg(key, d, 1000*3*(retryNum+1))
			if e != nil {
				log.Printf("消息进入ttl延时队列失败 err :%s \n", e)
			}
		} else {
			//消息失败 入库db
			log.Println("消息3次处理失败,进入异常环节")
			funcName(key, g, d)
			g.receivers[key].Exception(key, err, d)
		}
	} else {
		funcName(key, g, d)
	}
}

func funcName(key string, g *RabbitMQ, d amqp.Delivery) {
	// 判断是否需要ack
	if !g.consumers[key].AutoAck {
		// 手动ack
		err := d.Ack(false)
		if err != nil {
			log.Printf("消息消费ack失败 err :%s \n", err)
		}
	}
}

func (g *RabbitMQ) Register(key string, consumer conf.ConsumerConf, receiver conf.Receive) error {
	// 注册消费者
	return g.register(key, consumer, receiver)

}

func (g *RabbitMQ) Start() {
	g.isStart = true
	// 输出日志
	log.Println("RabbitMQ Start")
	go g.onChan()
	// 等待连接成功
	err := g.connect()
	if err != nil {
		return
	}
}

func (g *RabbitMQ) Stop() {
	g.isStop = true
	g.stop <- true
}

func (g *RabbitMQ) IsClose() bool {
	return g.conn == nil || g.conn.IsClosed()
}

func (g *RabbitMQ) connect() error {
	lock := g.mu.TryLock()
	if !lock {
		return nil
	}
	defer g.mu.Unlock()
	for g.IsClose() {
		if !g.isClose {
			g.isClose = true
			log.Println("RabbitMQ connect close")
		}
		var err error
		log.Println("Connect RabbitMQ:", getRabbitURL(g.rabbitConf))
		if g.rabbitConf.Scheme == "amqps" {
			c := &tls.Config{
				InsecureSkipVerify: true,
			}
			g.conn, err = amqp.DialTLS(getRabbitURL(g.rabbitConf), c)
		} else {
			g.conn, err = amqp.Dial(getRabbitURL(g.rabbitConf))
		}
		if err != nil {
			// 延迟5秒后重试连接
			log.Println("Failed to connect to RabbitMQ:", err)
			time.Sleep(5 * time.Second) // 等待一段时间后重试连接
			continue
		}
	}
	log.Println("RabbitMQ connect success")
	if g.isClose {
		g.connected <- true
	}
	g.isClose = false
	return nil
}

func getRabbitURL(rabbitConf conf.RabbitConf) string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%d/%s", rabbitConf.Scheme, rabbitConf.Username, rabbitConf.Password,
		rabbitConf.Host, rabbitConf.Port, rabbitConf.VHost,
	)
}
