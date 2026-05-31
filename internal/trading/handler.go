package trading

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/pkg/model"
	"github.com/Soneto56/market-board/pkg/mq"
)

// Handler 交易模块 HTTP 处理器
type Handler struct {
	db *gorm.DB
	ch *amqp.Channel
}

// NewHandler 创建 Handler
func NewHandler(db *gorm.DB, ch *amqp.Channel) *Handler {
	return &Handler{db: db, ch: ch}
}

// PlaceOrderRequest 下单请求体
type PlaceOrderRequest struct {
	Symbol   string  `json:"symbol" binding:"required"`
	Side     string  `json:"side" binding:"required,oneof=BUY SELL"`
	Type     string  `json:"type" binding:"required,oneof=MARKET LIMIT"`
	Price    float64 `json:"price"` // 限价单需要
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
}

// PlaceOrder 下单接口（需要 JWT 鉴权）
func (h *Handler) PlaceOrder(c *gin.Context) {
	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文获取用户ID（JWT中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 创建订单（状态为 PENDING）
	order := model.Order{
		UserID:   userID.(uint),
		Symbol:   req.Symbol,
		Side:     req.Side,
		Type:     req.Type,
		Price:    req.Price,
		Quantity: req.Quantity,
		Status:   "PENDING",
	}

	// 写入数据库
	if err := h.db.Create(&order).Error; err != nil {
		log.Printf("Failed to create order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// 发送到 RabbitMQ 队列，由消费者异步撮合
	orderBytes, err := json.Marshal(order)
	if err != nil {
		log.Printf("Failed to marshal order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process order"})
		return
	}

	err = h.ch.Publish(
		"",                // exchange：默认 exchange
		mq.OrderQueueName, // routing key = 队列名
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         orderBytes,
			DeliveryMode: amqp.Persistent, // 消息持久化
		},
	)
	if err != nil {
		log.Printf("Failed to publish order to RabbitMQ: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit order"})
		return
	}

	log.Printf("order placed: user=%d, symbol=%s, side=%s, qty=%.4f",
		order.UserID, order.Symbol, order.Side, order.Quantity)

	c.JSON(http.StatusOK, gin.H{
		"message":  "order submitted",
		"order_id": order.ID,
		"status":   "PENDING",
	})
}

// GetOrders 查询用户订单列表
func (h *Handler) GetOrders(c *gin.Context) {
	// 从上下文获取用户ID（JWT中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 查询用户订单列表
	var orders []model.Order
	h.db.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&orders)
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// GetPositions 查询用户持仓
func (h *Handler) GetPositions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var positions []model.Position
	h.db.Where("user_id = ? AND quantity > 0", userID).Find(&positions)
	c.JSON(http.StatusOK, gin.H{"positions": positions})
}
