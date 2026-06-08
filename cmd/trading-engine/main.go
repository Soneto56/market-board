package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/internal/trading"
	"github.com/Soneto56/market-board/pkg/cache"
	"github.com/Soneto56/market-board/pkg/middleware"
	"github.com/Soneto56/market-board/pkg/mq"
)

func main() {
	// 1. 连接数据库
	dsn := "root:yzj24243456@tcp(127.0.0.1:3306)/market_board?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}
	log.Println("database connected")

	// 2. 连接 RabbitMQ
	conn, ch, err := mq.ConnectRabbitMQ(mq.DefaultRabbitMQURL)
	if err != nil {
		log.Fatal("failed to connect RabbitMQ:", err)
	}
	defer conn.Close()
	defer ch.Close()

	// 3. 连接 Redis
	rdb := cache.NewRedisClient()

	// 4. 创建 Hub（交易服务自己的 Hub，用于成交推送）
	hub := market.NewHub()
	go hub.Run()
	log.Println("trading hub started for fill notifications")

	// 5. 创建撮合引擎
	priceHub := trading.NewRedisPriceHub(rdb)
	engine := trading.NewEngine(db, priceHub)

	// 6. 启动消费者（传入 hub 用于成交推送）
	consumer := trading.NewConsumer(db, engine, ch, hub)
	if err := consumer.Start(); err != nil {
		log.Fatal("failed to start consumer:", err)
	}

	// 7. 创建 Handler
	handler := trading.NewHandler(db, ch)

	// 8. 设置路由
	r := gin.Default()

	// CORS 跨域中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	// WebSocket 端点：前端建立连接获取成交推送
	r.GET("/ws", func(c *gin.Context) {
		// 复用行情网关的 handler，复用 ServeWS 逻辑
		wsHandler := market.NewHandler(hub)
		wsHandler.ServeWS(c)
	})

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/orders", handler.PlaceOrder)
		api.GET("/orders", handler.GetOrders)
		api.GET("/positions", handler.GetPositions)
	}

	// 9. 启动
	log.Println("Trading Engine starting on :8083")
	if err := http.ListenAndServe(":8083", r); err != nil {
		log.Fatal("failed to start trading engine:", err)
	}
}
