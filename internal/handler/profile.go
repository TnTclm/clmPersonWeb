package handler

import (
	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetUserProfile 用户个人空间数据：用户信息 + TA 发过的说说
// GET /api/users/:id （公开，游客也能逛别人的空间）
func GetUserProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户ID不合法"})
		return
	}
	// 查空间主人
	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}
	// 说说总数（页面资料行展示用）
	var talkCount int64
	database.DB.Model(&model.Talk{}).Where("user_id = ?", id).Count(&talkCount)
	// 最近 20 条说说（新的在前）；空切片而非 nil（nil 转 JSON 是 null，前端会炸）
	talks := []model.Talk{}
	database.DB.Where("user_id = ?", id).
		Order("created_at DESC").
		Limit(20).
		Find(&talks)

	// Format("2006-01-02")：Go 的时间格式化用"参考时间"当模板，1=月 2=日
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"user": gin.H{
				"id":        user.ID,
				"nickname":  user.Nickname,
				"avatar":    user.Avatar,
				"bio":       user.Bio,
				"joinDate":  user.CreatedAt.Format("2006-01-02"),
				"talkCount": talkCount,
				"is_owner":  user.ID == uint(config.Cfg.OwnerID), // 主人是不是博主（决定展示不展示说说列表）
			},
			"talks": talks,
		},
	})
}
