package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Soneto56/market-board/internal/market"
)

func main() {
	// 1. 创建 Hub 并启动事件循环
	hub := market.NewHub()
	go hub.Run() // Hub.Run 必须在独立的 goroutine 中运行，因为它会阻塞在 for-select 上

	// 2. 创建行情模拟器并启动
	Simulator := market.NewSimulator(hub)
	Simulator.Start() // Simulator.Start 也会阻塞在 for-select 上，所以在独立的 goroutine 中运行
	log.Println("market simulator started with 10 trading pairs")

	// 3. 创建 Handler
	handler := market.NewHandler(hub)

	// 4. 设置路由
	r := gin.Default()

	// WebSocket 端点：前端通过这个地址建立连接
	r.GET("/ws", handler.ServeWS)

	// HTTP 端点（调试用）
	api := r.Group("/api/v1")
	{
		api.GET("/tickers", handler.GetTickers)
	}

	// 5. 静态文件服务：前端页面
	r.Static("/web", "./web")

	// 6. 启动行情网关服务
	log.Println("Market Gateway starting on :8082")
	if err := http.ListenAndServe(":8082", r); err != nil {
		log.Fatal("failed to start market gateway:", err) //打印致命错误，程序无法继续运行
	}
}
