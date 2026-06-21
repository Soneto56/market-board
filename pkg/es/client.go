package es

import (
	"log"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

const IndexName = "market-ticks"

func NewClient() *elasticsearch.Client {
	addr := getEnv("ES_ADDR", "http://localhost:9200")

	cfg := elasticsearch.Config{
		Addresses: strings.Split(addr, ","),
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to create ES client: %v", err)
	}

	res, err := es.Ping()
	if err != nil {
		log.Fatalf("failed to ping ES: %v", err)
	}
	res.Body.Close()

	log.Println("Elasticsearch connected")
	return es
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
