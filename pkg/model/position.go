package model

import "time"

// Position 用户持仓
type Position struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Symbol    string    `gorm:"index;size:16;not null" json:"symbol"` // 如 BTC-USDT
	Quantity  float64   `gorm:"not null" json:"quantity"`             // 持有数量
	AvgPrice  float64   `gorm:"not null" json:"avg_price"`            // 平均成本价
}
