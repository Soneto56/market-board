package data

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/mq"
)

// Consumer 消费行情数据，写入 ES
type Consumer struct {
	writer *Writer
	ch     *amqp.Channel
}

// NewConsumer 创建消费者
func NewConsumer(writer *Writer, ch *amqp.Channel) *Consumer {
	return &Consumer{writer: writer, ch: ch}
}

// Start 开始消费 tick 队列
func (c *Consumer) Start() error {
	msgs, err := c.ch.Consume(
		mq.TickQueueName,
		"data-service",
		false, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			var ticker market.Ticker
			if err := json.Unmarshal(msg.Body, &ticker); err != nil {
				msg.Reject(false)
				continue
			}
			c.writer.WriteTick(&ticker)
			msg.Ack(false)
		}
	}()

	log.Println("tick consumer started, writing to ES")
	return nil
}
