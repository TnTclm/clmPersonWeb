package handler

import (
	"blog/internal/database"
	"blog/internal/model"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterRequest  注册接口的参数格式
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=20"` // 必填 3-20 字符
	Password    string `json:"password" binding:"required,min=6,max=32"` // 必填 6-32 字符
	Nickname    string `json:"nickname" binding:"required,min=1,max=20"` // 必填 1-20 字符
	CaptchaID   string `json:"captcha_id" binding:"required"`            // 验证码号牌
	CaptchaCode string `json:"captcha_code" binding:"required"`          // 用户输入的答案
}

// Register  用户注册
// POST /api/register
// 1. 前端已经检验两次密码是否相等
// 2. 获取前端数据
// 3. 先核对验证码
// 4. 对密码加密  hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
// 5. 存入数据库
// 6. 返回前端成功数据
func Register(c *gin.Context) {
	var req RegisterRequest
	// 拆包
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数不合法: " + err.Error(),
		})
		return
	}
	// 验证码核对：号牌 + 用户答案
	if !captchaGen.Store.Verify(req.CaptchaID, req.CaptchaCode, true) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "验证码错误或已经过期",
		})
		return
	}
	// 密码加密 bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "密码加密失败",
		})
		return
	}
	user := model.User{
		Username: req.Username,
		Password: string(hash), // 只存密文
		Nickname: req.Nickname,
	}
	// 存入数据库
	if err := database.DB.Create(&user).Error; err != nil {
		// errors.Is：判断错误链里有没有"用户名重复"这一种
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户名已被注册"})
			return
		}
		// 其他错误（连不上、表坏了……）才是真的服务器问题
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "服务器开小差了，请稍后再试"})
		return
	}

	// 成功创建
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "注册成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}
