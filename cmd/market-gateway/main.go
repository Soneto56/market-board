package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/cache"
)

func main() {
	// 1. 连接 Redis
	rdb := cache.NewRedisClient()

	// 2. 创建 Hub 并启动
	hub := market.NewHub()
	go hub.Run()

	// 3. 创建行情模拟器（传入 Redis 客户端）
	simulator := market.NewSimulator(hub, rdb)
	simulator.Start()
	log.Println("market simulator started with 10 trading pairs")

	// 4. 创建 Handler
	handler := market.NewHandler(hub)

	// 5. 设置路由
	r := gin.Default()
	r.GET("/ws", handler.ServeWS)

	api := r.Group("/api/v1")
	{
		api.GET("/tickers", handler.GetTickers)
	}

	r.Static("/web", "./web")

	// 6. 启动
	log.Println("Market Gateway starting on :8082")
	if err := http.ListenAndServe(":8082", r); err != nil {
		log.Fatal(err)
	}
}
