package model

import "time"

// Tick 一笔行情快照，存入 Elasticsearch
type Tick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	High24h   float64   `json:"high_24h"`
	Low24h    float64   `json:"low_24h"`
	Volume24h float64   `json:"volume_24h"`
	Timestamp time.Time `json:"@timestamp"` // ES 默认时间字段名
}
