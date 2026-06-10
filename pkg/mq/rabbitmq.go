package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 队列名称常量
const (
	OrderQueueName = "order.queue" // 委托订单队列
	TickQueueName  = "tick.queue"  // ← 新增：行情数据队列
)

// DefaultRabbitMQURL 默认连接地址
// 生产环境应从配置文件读取
const DefaultRabbitMQURL = "amqp://guest:guest@localhost:5672/"

// ConnectRabbitMQ 建立 RabbitMQ 连接并声明队列
func ConnectRabbitMQ(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		conn.Close()
		return nil, nil, err
	}
	// 声明队列（持久化 + 非排他 + 非自动删除）
	_, err = ch.QueueDeclare(
		OrderQueueName, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	log.Println("RabbitMQ connected, order queue ready")
	return conn, ch, nil
}
