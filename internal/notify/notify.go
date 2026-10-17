package notify

import (
	"encoding/json"
	"log/slog"
	"time"

	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/model"
)

// ============================================================
// 通知模块 —— 消费者负责"写"，接口负责"读"
//
// 数据存 MySQL 的 notifications 表（原来存 Redis）：
// 通知是要长期保留的数据，Redis 一重启就没了，不适合。
// 分页、已读状态也都交给数据库处理。
// ============================================================

// Msg 队列里传的消息格式（生产者和消费者之间的"约定"）
// 改这里的字段名 = 改协议，两边都要同步改
type Msg struct {
	Kind     string `json:"kind"`              // comment=评论 / message=留言
	FromID   uint   `json:"from_id"`           // 谁发的（用来过滤"自己发给自己"）
	Nickname string `json:"nickname"`          // 昵称（展示用）
	Content  string `json:"content"`           // 原文（前端自己截断）
	TalkID   uint   `json:"talk_id,omitempty"` // 评论挂在哪条说说上（留言没有）
}

// Item 存进 Redis、返回给前端的通知（比 Msg 多一个时间）
type Item struct {
	Kind     string `json:"kind"`
	Nickname string `json:"nickname"`
	Content  string `json:"content"`
	TalkID   uint   `json:"talk_id,omitempty"`
	Time     string `json:"time"`
}

// HandleMessage 处理一条队列消息 —— 这就是 mq.Consume 的回调
// 返回 nil = 处理成功（会被 ack）；返回 error = 处理失败（会被丢弃并记日志）
func HandleMessage(body []byte) error {
	var msg Msg
	if err := json.Unmarshal(body, &msg); err != nil {
		return err // 格式不对的消息没法处理，返回错误让它被丢掉
	}

	ownerID := uint(config.Cfg.OwnerID)
	// 博主自己发的评论/留言，没必要再提醒自己
	if msg.FromID == ownerID {
		return nil
	}

	item := Item{
		Kind:     msg.Kind,
		Nickname: msg.Nickname,
		Content:  msg.Content,
		TalkID:   msg.TalkID,
		Time:     time.Now().Format("2006-01-02 15:04"),
	}
	return Add(ownerID, item)
}

// Add 给某个用户记一条通知（入库，重启不丢）
func Add(userID uint, item Item) error {
	n := model.Notification{
		ToUserID: userID,
		Kind:     item.Kind,
		Nickname: item.Nickname,
		Content:  item.Content,
		TalkID:   item.TalkID,
	}
	return database.DB.Create(&n).Error
}

// List 读"从第 offset 条开始的 limit 条"通知 + 未读数
// （数据库分页 = ORDER BY + OFFSET + LIMIT，和别的列表接口一个套路）
func List(userID uint, offset, limit int) ([]Item, int64, error) {
	rows := []model.Notification{}
	err := database.DB.
		Where("to_user_id = ?", userID).
		Order("id DESC"). // 新的在前（id 自增，比按时间排序还稳）
		Offset(offset).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// 转成前端要的形状（Item.Time 是格式化好的字符串，前端不用处理时区）
	items := []Item{}
	for _, r := range rows {
		items = append(items, Item{
			Kind:     r.Kind,
			Nickname: r.Nickname,
			Content:  r.Content,
			TalkID:   r.TalkID,
			Time:     r.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	// 未读数 = 这个人的未读通知有几条
	var unread int64
	database.DB.Model(&model.Notification{}).
		Where("to_user_id = ? AND is_read = ?", userID, false).
		Count(&unread)

	return items, unread, nil
}

// HandleFailed 死信队列的消费者：处理失败的消息会流到这里
// 这里只记录（方便排查），也可以在这里做告警或延迟重试
func HandleFailed(body []byte) error {
	slog.Error("收到死信（消息处理失败的原始内容）", "body", string(body))
	return nil // 记录完就算处理成功，避免"死信的死信"无限套娃
}

// MarkRead 把某人的通知全部标记为已读
func MarkRead(userID uint) error {
	return database.DB.Model(&model.Notification{}).
		Where("to_user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
