package handler

import (
	"blog/internal/database"
	"blog/internal/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 说说评论 —— 评论都挂在说说下（文章已并入说说）
// ============================================================

// CreateCommentRequest 发表评论参数
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"` // 内容必填，最长500字
}

// CommentWithUser 评论 + 评论者头像（连表查询的临时结构，不是数据库表）
// 匿名嵌入 model.Comment：JSON 输出时评论字段原样展开，再额外多一个 avatar
type CommentWithUser struct {
	model.Comment
	Avatar string `json:"avatar"` // 评论者头像（users 表连出来的），空串=没传过头像
}

// CreateTalkComment 发表评论（需要登录）
// POST /api/talks/:id/comments
func CreateTalkComment(c *gin.Context) {
	// 路径参数转数字，老三样
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说说ID不合法"})
		return
	}
	// 评论前先确认说说存在——不能给不存在的内容留言
	var talk model.Talk
	if err := database.DB.First(&talk, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "说说不存在"})
		return
	}
	// 防刷屏闸门：评论和说说共用发言额度
	if !AllowSpeak(c.GetUint("user_id")) {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "发太快啦，休息一下再评"})
		return
	}
	// 拆包校验内容
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "评论内容不合法"})
		return
	}
	// 组装评论。昵称/用户ID都从 JWT 令牌里拿（中间件塞进上下文）——
	// 前端说"我是谁"不算数，得听令牌的，防伪造
	comment := model.Comment{
		TalkID:   uint(id),
		UserID:   c.GetUint("user_id"),
		Nickname: c.GetString("nickname"),
		Content:  req.Content,
	}
	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "评论失败，请稍后再试"})
		return
	}
	// 把存好的评论（带ID和创建时间）原样返回，前端直接塞进列表，不用重新请求
	TouchActivity(c.GetUint("user_id")) // 活跃度+1（热力图）
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "评论成功", "data": comment})
}

// ListTalkComments 某条说说的评论列表（公开，游客可看）
// GET /api/talks/:id/comments
func ListTalkComments(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "说说ID不合法"})
		return
	}
	// 空切片而不是 nil：nil 转 JSON 是 null，前端拿到会炸；[] 才是安全的空数组
	comments := []CommentWithUser{}
	// JOIN（连表查询）：评论表的 user_id 对上用户表的 id，
	// 一条结果里同时有评论内容和评论者的头像
	// LEFT JOIN：就算对不上号（用户没了），评论也要显示，头像给空串
	database.DB.Table("comments").
		Select("comments.*, users.avatar AS avatar").
		Joins("LEFT JOIN users ON users.id = comments.user_id").
		Where("comments.talk_id = ?", id).
		Order("comments.created_at ASC"). // 时间正序：像聊天记录从上往下读
		Limit(200).
		Scan(&comments)

	// data 直接就是数组，前端取的时候别搞混
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": comments})
}

// DeleteTalkComment 删除评论（需要登录：自己的评论自己删，博主谁的都能删）
// DELETE /api/talks/:id/comments/:cid
func DeleteTalkComment(c *gin.Context) {
	talkID, err1 := strconv.Atoi(c.Param("id"))
	cid, err2 := strconv.Atoi(c.Param("cid"))
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "ID不合法"})
		return
	}
	var comment model.Comment
	if err := database.DB.First(&comment, cid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "评论不存在"})
		return
	}
	// 评论必须真的挂在这条说说下（防有人拿别处的评论ID来套）
	if comment.TalkID != uint(talkID) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "评论不属于这条说说"})
		return
	}
	// 归属校验：自己的评论随便删；博主是"执法者"，谁的都能删
	if comment.UserID != c.GetUint("user_id") && !isOwner(c) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "只能删除自己的评论"})
		return
	}
	if err := database.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败，请稍后再试"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}
