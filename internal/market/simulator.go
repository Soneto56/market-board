package market

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

// Ticker 单个交易对的实时行情
type Ticker struct {
	Symbol    string  `json:"symbol"`     // 交易对，如 BTC-USDT
	Price     float64 `json:"price"`      // 最新价
	Bid       float64 `json:"bid"`        // 买一价
	Ask       float64 `json:"ask"`        // 卖一价
	High24h   float64 `json:"high_24h"`   // 24小时最高
	Low24h    float64 `json:"low_24h"`    // 24小时最低
	Volume24h float64 `json:"volume_24h"` // 24小时成交量
	Timestamp int64   `json:"timestamp"`  // 时间戳（毫秒）
}

// SimulateTicker 生成模拟行情数据
type Simulator struct {
	symbols   []string           // 所有要模拟的交易对
	initPrice map[string]float64 // 每个交易对的初始价格
	hub       *Hub               // WebSocket Hub（行情生成后推送到这里）
	stopCh    chan struct{}      // 停止信号
	rdb       *redis.Client      // Redis 客户端
}

// NewSimulator 创建一个新的行情模拟器
func NewSimulator(hub *Hub, rdb *redis.Client) *Simulator {
	return &Simulator{
		symbols: []string{
			"BTC-USDT", "ETH-USDT", "BNB-USDT", "SOL-USDT", "ADA-USDT",
			"XRP-USDT", "DOGE-USDT", "AVAX-USDT", "DOT-USDT", "MATIC-USDT",
		},
		initPrice: map[string]float64{
			"BTC-USDT":   67500.0,
			"ETH-USDT":   3450.0,
			"BNB-USDT":   580.0,
			"SOL-USDT":   142.0,
			"ADA-USDT":   0.45,
			"XRP-USDT":   0.62,
			"DOGE-USDT":  0.15,
			"AVAX-USDT":  35.0,
			"DOT-USDT":   7.2,
			"MATIC-USDT": 0.72,
		},
		hub:    hub,
		rdb:    rdb,
		stopCh: make(chan struct{}),
	}
}

// Start 启动所有交易对的行情模拟
// 每个交易对启动一个独立的 goroutine，每秒更新一次价格
func (s *Simulator) Start() {
	for _, symbol := range s.symbols {
		// 闭包陷阱！必须在循环内用局部变量捕获 symbol
		// 否则所有 goroutine 都会用循环最后一个 symbol
		sym := symbol
		price := s.initPrice[sym]
		go s.runSymbol(sym, price)
	}
}

// runSymbol 模拟单个交易对的行情变化
func (s *Simulator) runSymbol(symbol string, price float64) {
	// 用独立的随机源，避免多个 goroutine 共用全局随机源导致锁竞争
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	baseVol := 1000.0 // 基础成交量

	for {
		select {
		case <-s.stopCh: // 这里一直阻塞，直到收到信号
			return // 收到停止信号，退出 goroutine

		default:
			// 价格波动：在 ±1% 范围内随机变化
			changePct := (rng.Float64() - 0.5) * 0.02 // -1% ~ +1%
			price *= 1 + changePct

			// 买卖价差约 0.05%
			spread := price * 0.0005
			bid := price - spread
			ask := price + spread

			ticker := &Ticker{
				Symbol:    symbol,
				Price:     round2(price),
				Bid:       round2(bid),
				Ask:       round2(ask),
				High24h:   round2(price * (1 + rng.Float64()*0.03)),
				Low24h:    round2(price * (1 - rng.Float64()*0.03)),
				Volume24h: round2(baseVol * (1 + rng.Float64()*10)),
				Timestamp: time.Now().UnixMilli(),
			}

			// 推送到 Hub，由 Hub 分发给订阅该交易对的所有客户端
			s.hub.BroadcastTicker(ticker)

			// ========== 新增：写入 Redis 缓存 ==========
			ctx := context.Background()
			tickerKey := "market:ticker:" + symbol

			// 将 Ticker 序列化为 JSON
			tickerJSON, err := json.Marshal(ticker)
			if err == nil {
				// SETEX：设置 key 并设置过期时间（3秒）
				// 行情数据是高频更新的，3秒过期即使服务异常也不会长时间残留脏数据
				s.rdb.SetEx(ctx, tickerKey, tickerJSON, 3*time.Second)
			}

			// 每秒更新一次
			time.Sleep(1 * time.Second)
		}

	}
}

// Stop 停止所有行情模拟
func (s *Simulator) Stop() {
	close(s.stopCh) // 关闭通道，通知所有 goroutine 停止
}

// round2 保留两位小数
func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
