package mq

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrderQueueName = "order.queue"
	TickQueueName  = "tick.queue"
)

func ConnectRabbitMQ(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	// 声明订单队列
	_, err = ch.QueueDeclare(OrderQueueName, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}

	// 声明行情队列
	_, err = ch.QueueDeclare(TickQueueName, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}

	log.Println("RabbitMQ connected, queues ready")
	return conn, ch, nil
}

func GetRabbitMQURL() string {
	user := getEnv("RABBITMQ_USER", "guest")
	pass := getEnv("RABBITMQ_PASSWORD", "guest")
	host := getEnv("RABBITMQ_HOST", "localhost")
	port := getEnv("RABBITMQ_PORT", "5672")
	return "amqp://" + user + ":" + pass + "@" + host + ":" + port + "/"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
