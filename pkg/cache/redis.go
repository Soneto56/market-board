package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// Redis 配置常量
const (
	RedisAddr     = "localhost:6379"
	RedisPassword = "123456" // 你的 Redis 密码，如果没有就留空
	RedisDB       = 0
)

// 行情数据的 Key 前缀
// 最终 Key 格式：market:ticker:BTC-USDT
const TickerKeyPrefix = "market:ticker:"

// NewRedisClient 创建 Redis 客户端
func NewRedisClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: RedisPassword,
		DB:       RedisDB,
	})

	// 启动时验证连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	log.Println("Redis connected successfully")
	return rdb
}
