package handler

import (
	"log/slog"
	"net/http"

	"blog/internal/mq"
	"blog/internal/notify"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 通知（未读提醒）
// 流程：评论/留言接口发消息 → 消费者写 Redis → 这里的接口负责"读"
// ============================================================

// publishNotify 把"有新评论/留言"丢进消息队列（异步通知博主）
//
// 关键设计：发消息失败**只记日志、不影响用户**。
// 评论已经存进数据库了，通知只是锦上添花 —— 这就是"异步"的意义：
// 副业失败不能拖累主业。
func publishNotify(c *gin.Context, kind, content string, talkID uint) {
	msg := notify.Msg{
		Kind:     kind,                    // "comment" 或 "message"
		FromID:   c.GetUint("user_id"),    // 谁发的（消费者用它过滤"自己发给自己"）
		Nickname: c.GetString("nickname"), // 昵称从令牌拿，前端说了不算
		Content:  content,                 // 原文
		TalkID:   talkID,                  // 只有评论才有
	}
	if err := mq.Publish(mq.QueueNotify, msg); err != nil {
		slog.Warn("通知消息发送失败（不影响本次操作）", "err", err)
	}
}

// ListNotifications 我的通知列表 + 未读数
// GET /api/notifications?page=1&size=20（需要登录）
func ListNotifications(c *gin.Context) {
	page, size := pageParams(c) // 分页参数（和别的列表共用同一套规则）
	// Redis 列表按下标取：第 page 页 = 从 (page-1)*size 开始取。
	// 多取 1 条用来判断"还有没有更早的"（和别的列表同一个套路）
	items, unread, err := notify.List(c.GetUint("user_id"), (page-1)*size, size+1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "读取通知失败"})
		return
	}
	hasMore := len(items) > size
	if hasMore {
		items = items[:size] // 探路的那条不返回
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"items":  items,  // 通知列表（空也是 []）
			"unread": unread, // 未读数
		},
		"has_more": hasMore, // 还有更早的通知
	})
}

// MarkNotificationsRead 把未读数清零（用户点开看过了）
// POST /api/notifications/read（需要登录）
func MarkNotificationsRead(c *gin.Context) {
	if err := notify.MarkRead(c.GetUint("user_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "操作失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "ok"})
}
