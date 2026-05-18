package market

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler 行情网关的 HTTP 处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建 Handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeWS 处理 WebSocket 升级请求
// 客户端通过 GET /ws 建立 WebSocket 连接
func (h *Handler) ServeWS(c *gin.Context) {
	// 将 HTTP 连接升级为 WebSocket 连接
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	// 创建客户端
	client := &Client{
		hub:  h.hub,
		conn: conn,
		send: make(chan []byte, 256), // 缓冲 256 条消息
	}

	// 注册到 Hub
	h.hub.register <- client

	// 订阅所有交易对的行情
	// 实际项目中可以让客户端通过 URL 参数指定要订阅的交易对
	symbols := []string{
		"BTC-USDT", "ETH-USDT", "BNB-USDT", "SOL-USDT", "ADA-USDT",
		"XRP-USDT", "DOGE-USDT", "AVAX-USDT", "DOT-USDT", "MATIC-USDT",
	}

	for _, sym := range symbols {
		h.hub.Subscribe(client, sym)
	}

	// 启动读写协程
	// 每个 WebSocket 连接需要两个 goroutine：一个读、一个写
	go client.readPump()
	go client.writePump()

	log.Printf("websocket client connected, subscribed to %d symbols", len(symbols))
}

// GetTickers 返回所有交易对的最新行情快照（HTTP 接口，方便调试）
func (h *Handler) GetTickers(c *gin.Context) {
	// TODO: 下一阶段从 Redis 读取缓存的最新行情
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet, use WebSocket for real-time data"})
}
