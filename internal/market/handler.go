package market

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Soneto56/market-board/pkg/middleware"
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
// 支持通过 URL 参数传递 token：/ws?token=xxx
// 如果提供了有效 token，则将连接与用户绑定，成交回报可推送至此连接
func (h *Handler) ServeWS(c *gin.Context) {
	// 升级 WebSocket
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	// 尝试解析用户身份
	var userID uint
	tokenString := c.Query("token")

	if tokenString != "" {
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return middleware.JWTSecret, nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if uid, ok := claims["user_id"].(float64); ok {
					userID = uint(uid)
				}
			}
		}
	}

	// 创建客户端
	client := &Client{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		connID: uuid.New().String()[:8], // 取前 8 位作为短标识
	}

	// 注册到 Hub
	h.hub.register <- client

	// 订阅所有交易对
	symbols := []string{
		"BTC-USDT", "ETH-USDT", "BNB-USDT", "SOL-USDT", "ADA-USDT",
		"XRP-USDT", "DOGE-USDT", "AVAX-USDT", "DOT-USDT", "MATIC-USDT",
	}
	for _, sym := range symbols {
		h.hub.Subscribe(client, sym)
	}

	// 启动读写协程
	go client.writePump()
	go client.readPump()

	if userID != 0 {
		log.Printf("websocket client connected: connID=%s, userID=%d, subscribed to %d symbols",
			client.connID, userID, len(symbols))
	} else {
		log.Printf("websocket client connected: connID=%s, anonymous, subscribed to %d symbols",
			client.connID, len(symbols))
	}
}

// GetTickers 返回所有交易对的最新行情快照
func (h *Handler) GetTickers(c *gin.Context) {
	// TODO: 从 Redis 读取缓存的最新行情
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet, use WebSocket for real-time data"})
}
