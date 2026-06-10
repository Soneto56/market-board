package data

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"

	"github.com/Soneto56/market-board/internal/market"
	"github.com/Soneto56/market-board/pkg/es"
)

// Writer 将行情数据批量写入 Elasticsearch
type Writer struct {
	es      *elasticsearch.Client
	indexer esutil.BulkIndexer
}

// NewWriter 创建写入器
func NewWriter(esClient *elasticsearch.Client) *Writer {
	indexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         es.IndexName,
		Client:        esClient,
		NumWorkers:    2,
		FlushBytes:    5e6, // 5MB
		FlushInterval: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create bulk indexer: %v", err)
	}

	return &Writer{es: esClient, indexer: indexer}
}

// WriteTick 写入一笔行情快照到 ES
func (w *Writer) WriteTick(ticker *market.Ticker) {
	doc := map[string]interface{}{
		"symbol":     ticker.Symbol,
		"price":      ticker.Price,
		"bid":        ticker.Bid,
		"ask":        ticker.Ask,
		"high_24h":   ticker.High24h,
		"low_24h":    ticker.Low24h,
		"volume_24h": ticker.Volume24h,
		"@timestamp": time.Now(),
	}

	body, err := json.Marshal(doc)
	if err != nil {
		log.Printf("failed to marshal tick: %v", err)
		return
	}

	w.indexer.Add(context.Background(), esutil.BulkIndexerItem{
		Action: "index",
		Body:   bytes.NewReader(body),
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				log.Printf("ES bulk index failed: %v", err)
			}
		},
	})
}

// Close 关闭写入器，刷新缓冲区
func (w *Writer) Close() {
	w.indexer.Close(context.Background())
}
