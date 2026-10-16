package handler

import (
	"blog/internal/database"
	"blog/internal/middleware"
	"blog/internal/model"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// LoginRequest 登录参数
type LoginRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=20"`
	Password    string `json:"password" binding:"required,min=6,max=32"`
	CaptchaID   string `json:"captcha_id" binding:"required"`   // 验证码号牌
	CaptchaCode string `json:"captcha_code" binding:"required"` // 用户输入的答案
}

// Login 用户登录：第一段验证身份 → 第二段签发JWT
// POST /api/login
// xxxxxxxxxxx    首先你有jwt 不用登录
// 1. 先检验验证码是否正确
// 2. 查询数据库是否有改用户      无-> 返回错误   有-> 顺带拿取该用户信息
// 3. 通过拿到的用户信息  与前端传来的密码进行对比  bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
// 4. 验证通过之后给用户发jwt
// 5. 向前端传递成功数据
func Login(c *gin.Context) {
	var req LoginRequest
	// 拆包
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数不合法",
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
	// 查询用户：SELECT * FROM users WHERE username=? LIMIT 1
	var user model.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不区分"用户不存在"和"密码错误"，防止别人探测哪些用户名已注册
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "用户名或密码错误",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "服务器开小差了, 请稍后尝试",
		})
		return
	}
	// 比对密码：注册时Generate加密，登录时Compare比对，是一对
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "用户名或密码错误",
		})
		return
	}
	// 验证通过，签发JWT门禁卡
	token, err := middleware.GenerateToken(&user) // 我们要的是token 传入相应的数据  获取相应的token
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "签发令牌失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": gin.H{
			"token":    token,
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
		},
	})
}
