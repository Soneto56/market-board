package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret 签名密钥
// 生产环境应从配置文件或环境变量读取，绝对不能硬编码
var JWTSecret = []byte("market-board-secret-key-2026")

// AuthMiddleware JWT 鉴权中间件
// 从请求头 Authorization: Bearer <token> 中提取并验证 JWT
//
// 使用方式：
//
//	r.Use(AuthMiddleware())  // 保护所有路由
//	api.Use(AuthMiddleware()) // 保护路由组
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort() // 终止后续处理
			return
		}

		// 2. 检查 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format, expected: Bearer <token>"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 解析并验证 JWT
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法是否为 HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// 4. 提取 Claims 中的用户信息
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// user_id 在 Claims 中以 float64 存储（JSON 数字默认类型）
		userID, ok := claims["user_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
			c.Abort()
			return
		}

		// 5. 将用户信息注入上下文，后续 Handler 通过 c.Get("user_id") 获取
		c.Set("user_id", uint(userID))

		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}

		// 继续执行后续处理
		c.Next()
	}
}
