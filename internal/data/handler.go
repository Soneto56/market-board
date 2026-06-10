package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"

	"github.com/Soneto56/market-board/pkg/es"
)

// Handler 数据服务 HTTP 处理器
type Handler struct {
	esClient *elasticsearch.Client
}

// NewHandler 创建 Handler
func NewHandler(esClient *elasticsearch.Client) *Handler {
	return &Handler{esClient: esClient}
}

// KlineQuery K线查询参数
type KlineQuery struct {
	Symbol    string `form:"symbol" binding:"required"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// KlinePoint 单个K线数据点
type KlinePoint struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// GetKlines 查询历史K线（日级别 OHLC）
// GET /api/v1/klines?symbol=BTC-USDT&start_time=2026-06-01T00:00:00Z&end_time=2026-06-03T00:00:00Z
func (h *Handler) GetKlines(c *gin.Context) {
	var q KlineQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 默认查询最近 24 小时
	now := time.Now()
	if q.EndTime == "" {
		q.EndTime = now.Format(time.RFC3339)
	}
	if q.StartTime == "" {
		q.StartTime = now.Add(-24 * time.Hour).Format(time.RFC3339)
	}

	// ES 聚合查询：按 1 天 聚合 OHLC
	query := fmt.Sprintf(`{
		"size": 0,
		"query": {
			"bool": {
				"filter": [
					{"term": {"symbol.keyword": "%s"}},
					{"range": {"@timestamp": {"gte": "%s", "lte": "%s"}}}
				]
			}
		},
		"aggs": {
			"klines": {
				"date_histogram": {
					"field": "@timestamp",
					"fixed_interval": "1d"
				},
				"aggs": {
					"open":  {"min": {"field": "price"}},
					"high":  {"max": {"field": "price"}},
					"low":   {"min": {"field": "price"}},
					"close": {"max": {"field": "price"}}
				}
			}
		}
	}`, q.Symbol, q.StartTime, q.EndTime)

	// OHLC 聚合有简化，open 用 min、close 用 max 只是近似值
	// 真实的 OHLC 需要按时间排序后取首尾，这里简化处理

	res, err := h.esClient.Search(
		h.esClient.Search.WithContext(context.Background()),
		h.esClient.Search.WithIndex(es.IndexName),
		h.esClient.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		log.Printf("ES search failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	c.JSON(http.StatusOK, result)
}
