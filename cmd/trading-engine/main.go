package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/internal/trading"
	"github.com/Soneto56/market-board/pkg/middleware"
	"github.com/Soneto56/market-board/pkg/mq"
)

// 行情价格提供者（从 Redis 获取，暂用简单 mock）
// 这个需要行情网关把最新价写入 Redis，下一阶段做
type SimplePriceHub struct{}

func (h *SimplePriceHub) GetLatestPrice(symbol string) (float64, float64, float64) {
	// TODO: 下一阶段从 Redis 读取行情网关写入的最新价
	// 暂时返回模拟价格
	return 67500.0, 67499.0, 67501.0
}

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

	// 3. 创建撮合引擎
	priceHub := &SimplePriceHub{}
	engine := trading.NewEngine(db, priceHub)

	// 4. 启动消费者（异步消费订单队列）
	consumer := trading.NewConsumer(db, engine, ch)
	if err := consumer.Start(); err != nil {
		log.Fatal("failed to start consumer:", err)
	}

	// 5. 创建 Handler
	handler := trading.NewHandler(db, ch)

	// 6. 设置路由
	r := gin.Default()

	// 需要鉴权的路由
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware()) // JWT 鉴权中间件
	{
		api.POST("/orders", handler.PlaceOrder)
		api.GET("/orders", handler.GetOrders)
		api.GET("/positions", handler.GetPositions)
	}

	// 7. 启动交易服务
	log.Println("Trading Engine starting on :8083")
	if err := http.ListenAndServe(":8083", r); err != nil {
		log.Fatal("failed to start trading engine:", err)
	}
}
