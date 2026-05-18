package model

import "time"

// Order 交易订单
type Order struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Symbol    string    `gorm:"index;size:16;not null" json:"symbol"`  // 如 BTC-USDT
	Side      string    `gorm:"size:4;not null" json:"side"`           // buy 或 sell
	Type      string    `gorm:"size:8;not null" json:"type"`           // market 或 limit
	Price     float64   `json:"price"`                                 // 仅限 limit 订单
	Quantity  float64   `json:"quantity"`                              // 仅限 market 订单
	Status    string    `gorm:"size:16;default:PENDING" json:"status"` // PENDING/FILLED/CANCELLED
}
