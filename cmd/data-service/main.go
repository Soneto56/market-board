package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Soneto56/market-board/internal/data"
	"github.com/Soneto56/market-board/pkg/es"
	"github.com/Soneto56/market-board/pkg/mq"
)

func main() {
	// 1. 连接 ES
	esClient := es.NewClient()

	// 2. 连接 RabbitMQ
	conn, ch, err := mq.ConnectRabbitMQ(mq.DefaultRabbitMQURL)
	if err != nil {
		log.Fatalf("failed to connect RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// 3. 创建写入器
	writer := data.NewWriter(esClient)
	defer writer.Close()

	// 4. 启动消费者（消费 tick 队列，写入 ES）
	consumer := data.NewConsumer(writer, ch)
	if err := consumer.Start(); err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	// 5. 路由
	handler := data.NewHandler(esClient)
	r := gin.Default()
	api := r.Group("/api/v1")
	{
		api.GET("/klines", handler.GetKlines)
	}

	// 6. 启动
	log.Println("Data Service starting on :8084")
	if err := http.ListenAndServe(":8084", r); err != nil {
		log.Fatal(err)
	}
}
