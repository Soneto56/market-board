package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/cache"
	"github.com/Soneto56/market-board/pkg/mq"
)

// tickPublisher 行情数据发送器
type tickPublisher struct {
	ch *amqp.Channel
}

func (p *tickPublisher) Publish(ticker *market.Ticker) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	body, err := json.Marshal(ticker)
	if err != nil {
		return
	}

	p.ch.PublishWithContext(ctx,
		"",               // exchange
		mq.TickQueueName, // routing key
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Transient, // 行情数据允许丢失，不需要持久化
		},
	)
}

func main() {
	// 1. 连接 Redis
	rdb := cache.NewRedisClient()

	// 2. 连接 RabbitMQ
	conn, ch, err := mq.ConnectRabbitMQ(mq.DefaultRabbitMQURL)
	if err != nil {
		log.Fatalf("failed to connect RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// 声明 tick 队列
	_, err = ch.QueueDeclare(
		mq.TickQueueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("failed to declare tick queue: %v", err)
	}
	log.Println("tick queue declared")
	// 声明 tick 队列
	ch.QueueDeclare(mq.TickQueueName, true, false, false, false, nil)

	// 3. 创建行情发送器
	publisher := &tickPublisher{ch: ch}

	// 4. 创建 Hub
	hub := market.NewHub()
	go hub.Run()

	// 5. 创建行情模拟器
	simulator := market.NewSimulator(hub, rdb, publisher)
	simulator.Start()
	log.Println("market simulator started with 10 trading pairs, publishing to MQ")

	// 6. 路由
	marketHandler := market.NewHandler(hub)
	r := gin.Default()
	r.GET("/ws", marketHandler.ServeWS)
	r.Static("/web", "./web")

	log.Println("Market Gateway starting on :8082")
	if err := http.ListenAndServe(":8082", r); err != nil {
		log.Fatal(err)
	}
}
