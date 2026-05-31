package trading

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/pkg/model"
	"github.com/Soneto56/market-board/pkg/mq"
)

// Consumer RabbitMQ 消费者，消费订单队列
type Consumer struct {
	db     *gorm.DB
	engine *Engine
	ch     *amqp.Channel
}

// NewConsumer 创建消费者
func NewConsumer(db *gorm.DB, engine *Engine, ch *amqp.Channel) *Consumer {
	return &Consumer{db: db, engine: engine, ch: ch}
}

// Start 开始消费订单队列
func (c *Consumer) Start() error {
	msgs, err := c.ch.Consume(
		mq.OrderQueueName,
		"",
		false, // 手动确认
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			c.handleMessage(msg)
		}
	}()

	log.Println("order consumer started")
	return nil
}

// handleMessage 处理单条消息
func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var order model.Order
	if err := json.Unmarshal(msg.Body, &order); err != nil {
		log.Printf("failed to unmarshal order: %v", err)
		msg.Reject(false)
		return
	}

	// 使用日志 Notifier（后续替换为 WebSocket 推送）
	notifier := &LogNotifier{}

	if err := c.engine.ProcessOrder(&order, notifier); err != nil {
		log.Printf("failed to process order %d: %v", order.ID, err)
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
	log.Printf("order %d processed successfully", order.ID)
}

// LogNotifier 简单的日志通知器（后续替换为 WebSocket 推送）
type LogNotifier struct{}

func (n *LogNotifier) Notify(userID uint, data any) {
	log.Printf("notification to user %d: %+v", userID, data)
}
