package handler

import (
	"blog/internal/database"
	"blog/internal/model"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================================
// 活跃度记录 —— LeetCode 式热力图的数据源
// 记什么：上线（/me 被调用）记 1 次；发文章/说说/评论/留言/相册再 +1
// 怎么存：一人一天一行（user_id + date 联合唯一索引），重复活动 count+1
// ============================================================

// TouchActivity 给某用户今天的活跃次数 +1（今天还没有记录就插一行）
// 被到处调用，所以设计成"静默失败"：热力图挂了绝不能拖累正常功能，
// 所以这里不返回错误、调用方也不用接——最多损失一格小方块，值
func TouchActivity(userID uint) {
	today := time.Now().Format("2006-01-02") // Go 参考时间固定是 2006-01-02 15:04:05（记忆口诀：1234567）
	database.DB.Clauses(clause.OnConflict{
		// 撞上 (user_id, date) 唯一索引 = 今天已经有记录了
		Columns: []clause.Column{{Name: "user_id"}, {Name: "date"}},
		// 已有记录 → 次数+1（对应 MySQL 的 ON DUPLICATE KEY UPDATE）
		DoUpdates: clause.Assignments(map[string]interface{}{
			"count": gorm.Expr("`count` + 1"),
		}),
	}).Create(&model.Activity{UserID: userID, Date: today, Count: 1})
}

// UserActivities 某用户过去一年的活跃记录（公开，访客也能看任何人的热力图）
// GET /api/users/:id/activities
// 返回 [{date: "2026-10-14", count: 3}, ...] 只有"动过"的日子，没动的天前端自己补灰色
func UserActivities(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户ID不合法"})
		return
	}
	// 一年前的今天（AddDate 负数=往回退），热力图只显示最近一年
	since := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	// 老规矩：必须初始化空切片！nil 转 JSON 是 null，前端 .map 直接炸
	days := []model.Activity{}
	database.DB.Where("user_id = ? AND date >= ?", id, since).
		Order("date ASC").
		Find(&days)
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": days})
}
