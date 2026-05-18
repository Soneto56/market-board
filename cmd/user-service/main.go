package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/internal/user"
	"github.com/Soneto56/market-board/pkg/model"
)

func main() {
	// ==================== 1. 连接数据库 ====================
	// 修改为你的 MySQL 连接信息
	// parseTime=True：让 GORM 自动把 MySQL 的 datetime 转为 Go 的 time.Time
	// loc=Local：使用本地时区
	dsn := "root:yzj24243456@tcp(127.0.0.1:3306)/market_board?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// log.Fatal：打印错误并退出程序（OS exit 1）
		log.Fatalf("连接数据库失败: %v", err)
	}
	log.Println("database connected successfully")

	// ==================== 2. 自动迁移 ====================
	// 开发阶段用 AutoMigrate 自动建表/加字段
	// 生产环境应使用独立迁移工具（如 golang-migrate）
	if err := db.AutoMigrate(&model.User{}, &model.Position{}, &model.Order{}); err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}
	log.Println("database migrated successfully")

	// ==================== 3. 初始化 Handler ====================
	// 将 db 注入到 Handler 中（依赖注入）
	userHandler := user.NewHandler(db)

	// ==================== 4. 设置路由 ====================
	// gin.Default() 自带 Logger 和 Recovery 中间件
	r := gin.Default()

	// 路由分组：所有 API 都在 /api/v1 下
	api := r.Group("/api/v1")
	{
		// 用户注册
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)
	}
	// ==================== 5. 启动服务 ====================
	log.Println("User service starting on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
