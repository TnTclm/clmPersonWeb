package handler

import (
	"blog/internal/database"
	"blog/internal/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 留言板模块 —— 弹幕式：留言会化作樱花雨里飞过的弹幕
// 规则：登录才能留（弹幕带真实头像昵称），游客可以看
// 删除：本人可删自己的，博主作为"执法者"可删任何人的
// ============================================================

// CreateMessageRequest 发表留言参数
type CreateMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=50"` // 弹幕要短，50字封顶
}

// CreateMessage 发表留言（需要登录）
// POST /api/guestbook
func CreateMessage(c *gin.Context) {
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "留言1-50字"})
		return
	}
	msg := model.Message{
		UserID:   c.GetUint("user_id"),
		Nickname: c.GetString("nickname"), // 昵称从令牌拿，防伪造
		Content:  req.Content,
	}
	if err := database.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "留言失败，请稍后再试"})
		return
	}
	TouchActivity(c.GetUint("user_id")) // 活跃度+1（热力图）
	// 异步通知博主："有人留言了"（丢进消息队列，失败了也不影响留言本身）
	publishNotify(c, "message", msg.Content, 0)
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "留言成功", "data": msg})
}

// MessageWithUser 留言 + 留言人头像（连表临时结构，老套路）
type MessageWithUser struct {
	model.Message
	Avatar string `json:"avatar"`
}

// ListMessages 留言列表（公开，游客可看弹幕）
// GET /api/guestbook
func ListMessages(c *gin.Context) {
	// 空切片初始化（nil 转 JSON 是 null，老规矩）
	messages := []MessageWithUser{}
	page, size := pageParams(c)
	database.DB.Table("messages").
		Select("messages.*, users.avatar AS avatar").
		Joins("LEFT JOIN users ON users.id = messages.user_id").
		Order("messages.created_at DESC"). // 先按"最新在前"取一页 —— 第1页就是最新的一批
		Offset((page - 1) * size).
		Limit(size + 1). // 多取 1 条用来判断"后面还有没有"
		Scan(&messages)

	// 有探路的那条 → 后面还有（把它切掉再返回）
	hasMore := len(messages) > size
	if hasMore {
		messages = messages[:size]
	}
	// 弹幕渲染希望"旧的先起飞"，所以把这一页再倒过来（时间正序）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	pageResult(c, messages, hasMore)
}

// DeleteMessage 删除留言（需要登录：自己的留言自己删，博主谁的都能删）
// DELETE /api/guestbook/:id
func DeleteMessage(c *gin.Context) {
	// 路径参数转数字，老三样
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "ID不合法"})
		return
	}
	// 先把留言整条查出来——必须拿到它的 UserID 才能判断归属
	// （不能拿路径里的 id 直接 Delete，那样等于谁都能删）
	var msg model.Message
	if err := database.DB.First(&msg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "留言不存在"})
		return
	}
	// 归属校验：不是自己的 → 只有博主能动手
	//   && 的含义：两个条件【同时成立】才拦人
	//     ① msg.UserID != 我   → 这条留言不是我写的
	//     ② !isOwner(c)        → 我也不是博主
	//   任何一个不成立（是我的 / 我是博主）都会放行
	// isOwner 定义在 album.go 里，和本文件同属 handler 包，可以直接喊，不用 import
	if msg.UserID != c.GetUint("user_id") && !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "只能删除自己的留言"})
		return
	}
	// 走到这儿 = 有权限，真删
	if err := database.DB.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败，请稍后再试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}
