package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Soneto56/market-board/pkg/model"
)

// ==================== 结构体定义 ====================

// Handler 用户服务处理器，持有数据库连接
// 这种"把依赖放在结构体里"的模式叫"依赖注入"，
// 好处是可以方便地替换依赖（比如测试时用 Mock 数据库），
// 也避免了全局变量带来的耦合问题。
type Handler struct {
	db *gorm.DB
}

// NewHandler 构造函数，创建 Handler 实例
// Go 没有"构造函数"关键字，社区约定用 NewXxx 函数来初始化结构体。
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// RegisterRequest 注册请求体
// binding 标签是 Gin 的参数校验规则：
// required=必填, min=最小长度, max=最大长度
// 在这里声明请求结构，让函数签名简洁明了。

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ==================== 业务方法 ====================

// Register 用户注册
// (h *Handler) 是"指针接收者"——方法"挂"在 Handler 类型上，
// 用指针而不是值，是为了避免拷贝整个 Handler（内部有 db 连接）。

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	// ShouldBindJSON：解析 JSON 请求体 + 执行 binding 校验规则
	// 如果校验失败（比如用户名太短），err 不为 nil
	if err := c.ShouldBindJSON(&req); err != nil {
		// gin.H 是 map[string]interface{} 的简写，用来构造 JSON 响应
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// bcrypt 加密密码
	// DefaultCost=10，在安全性和性能之间取平衡
	// 密码永远不要明文存储，这是基本安全原则。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := model.User{
		Username: req.Username,
		Password: string(hashedPassword), // 存储加密后的密码
		Balance:  100000,                 // 初始余额
	}

	// Create 插入数据库
	// 如果 username 重复（uniqueIndex 约束），会返回错误
	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "registration successful",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 根据 username 查询用户
	var user model.User
	// Where + First：条件查询第一条记录
	// 找不到记录时返回 gorm.ErrRecordNotFound
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 不管用户名不存在还是其他错误，统一返回 "invalid credentials"
			// 避免给攻击者提供"用户名是否存在"的信息差
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
	}

	// bcrypt.CompareHashAndPassword：比对加密密码和明文密码
	// 看密码是否正确
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// TODO: 下一阶段会在这里生成 JWT token 并返回

	c.JSON(http.StatusOK, gin.H{
		"message":  "login successful",
		"user_id":  user.ID,
		"username": user.Username,
	})
}
