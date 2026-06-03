package trading

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/cache"
)

// RedisPriceHub 通过 Redis 获取最新行情
// 实现了 trading.PriceHub 接口
type RedisPriceHub struct {
	rdb *redis.Client
}

// NewRedisPriceHub 创建 Redis 行情查询器
func NewRedisPriceHub(rdb *redis.Client) *RedisPriceHub {
	return &RedisPriceHub{rdb: rdb}
}

// GetLatestPrice 从 Redis 获取指定交易对的最新行情
// 实现了 PriceHub 接口，可以传入 Engine
func (h *RedisPriceHub) GetLatestPrice(symbol string) (price, bid, ask float64) {
	tickerKey := cache.TickerKeyPrefix + symbol

	data, err := h.rdb.Get(context.Background(), tickerKey).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("failed to get ticker from Redis: %v", err)
		}
		// Redis 中没有数据或查询失败，返回 0 表示价格不可用
		return 0, 0, 0
	}

	// 反序列化 JSON
	var ticker market.Ticker
	if err := json.Unmarshal([]byte(data), &ticker); err != nil {
		log.Printf("failed to unmarshal ticker: %v", err)
		return 0, 0, 0
	}

	return ticker.Price, ticker.Bid, ticker.Ask
}
