package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/internal/user"
	"github.com/Soneto56/market-board/pkg/model"
)

func main() {
	db, err := gorm.Open(mysql.Open(getDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	log.Println("database connected successfully")

	if err := db.AutoMigrate(&model.User{}, &model.Position{}, &model.Order{}); err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}
	log.Println("database migrated successfully")

	userHandler := user.NewHandler(db)

	r := gin.Default()
	api := r.Group("/api/v1")
	{
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)
	}

	log.Println("User service starting on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatal("failed to start server:", err)
	}
}

func getDSN() string {
	user := getEnv("MYSQL_USER", "root")
	pass := getEnv("MYSQL_PASSWORD", "yzj24243456")
	host := getEnv("MYSQL_HOST", "127.0.0.1")
	port := getEnv("MYSQL_PORT", "3306")
	dbName := getEnv("MYSQL_DATABASE", "market_board")
	return user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
