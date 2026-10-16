package middleware

import (
	"blog/internal/config"
	"blog/internal/model"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// 自定义MyClaims
type MyClaims struct {
	UserID               uint   `json:"user_id"`
	Username             string `json:"username"`
	Nickname             string `json:"nickname"`
	jwt.RegisteredClaims        // 这个隐式实现了 jwt.Claims 的 接口
}

// 生成JWT
func GenerateToken(user *model.User) (string, error) {
	claims := MyClaims{
		UserID:   user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 有效时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                         // 签发时间
			Issuer:    "clm",
		},
	}
	// 生成token 使用HS256算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 返回tokenString error
	return token.SignedString([]byte(config.Cfg.JWTSecret))
}

// AuthMiddleware 登录校验中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 约定 前端在请求头里带  Authorization: Bearer  tokenString
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "请先登录...",
			})
			return
		}
		// 按空格拆分  Authorization
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "令牌格式错误",
			})
			return
		}
		// 验证
		claims := &MyClaims{}
		// parts[1] == tokenString
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("非法的签名算法")
			}
			return []byte(config.Cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "登录已过期，请重新登录",
			})
			return // 回了错误响应立刻return，避免Abort后的代码继续执行
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Next()
	}
}
