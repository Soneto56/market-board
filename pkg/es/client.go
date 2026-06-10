package es

import (
	"log"

	"github.com/elastic/go-elasticsearch/v8"
)

const IndexName = "market-ticks"

func NewClient() *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to create ES client: %v", err)
	}

	// 验证连接
	res, err := es.Ping()
	if err != nil {
		log.Fatalf("failed to ping ES: %v", err)
	}
	res.Body.Close()

	log.Println("Elasticsearch connected")
	return es
}
