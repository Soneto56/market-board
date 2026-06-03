package trading

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/model"
	"github.com/Soneto56/market-board/pkg/mq"
)

// Consumer RabbitMQ 消费者，消费订单队列
type Consumer struct {
	db       *gorm.DB
	engine   *Engine
	ch       *amqp.Channel
	notifier Notifier // ← 新增：持有 notifier
}

// NewConsumer 创建消费者
func NewConsumer(db *gorm.DB, engine *Engine, ch *amqp.Channel, hub *market.Hub) *Consumer {
	return &Consumer{
		db:       db,
		engine:   engine,
		ch:       ch,
		notifier: &HubNotifier{hub: hub},
	}
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

	// 传入 notifier（不再是 nil）
	if err := c.engine.ProcessOrder(&order, c.notifier); err != nil {
		log.Printf("failed to process order %d: %v", order.ID, err)
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
	log.Printf("order %d processed successfully", order.ID)
}

// HubNotifier 通过 Hub 向用户推送成交消息
type HubNotifier struct {
	hub *market.Hub
}

func (n *HubNotifier) Notify(userID uint, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("failed to marshal notification: %v", err)
		return
	}
	n.hub.SendToUser(userID, payload)
}
