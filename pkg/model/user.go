package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `gorm:"uniqueIndex;size:32;not null" json:"username"`
	Password  string         `gorm:"size:128;not null" json:"-"` // json:"-" 不会返回给前端
	Balance   float64        `gorm:"default:0" json:"balance"`   // 账户余额
}
