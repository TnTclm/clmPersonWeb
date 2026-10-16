package handler

import (
	"blog/internal/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

// captchaGen 声明为指针，等 InitCaptcha 里真正创建
var captchaGen *base64Captcha.Captcha

// InitCaptcha 必须在 ConnectRedis 之后调用（main里调）
func InitCaptcha() {
	captchaGen = base64Captcha.NewCaptcha(
		&base64Captcha.DriverDigit{
			Height:   80,
			Width:    240,
			Length:   5,
			MaxSkew:  0.7,
			DotCount: 50,
		},
		NewRedisStore(database.RDB, 5*time.Minute), // 此时 RDB 已经是真客户端了
	)
}

func GenerateCaptcha(c *gin.Context) {
	id, b64, _, err := captchaGen.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "生成验证码失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"captchaId": id,
			"img":       b64,
		},
	})
}
