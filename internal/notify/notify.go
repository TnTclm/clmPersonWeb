package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"blog/internal/config"
	"blog/internal/database"
)

// ============================================================
// 通知模块 —— 消费者负责"写"，接口负责"读"
//
// 数据全放 Redis（理由：它只是"轻量提醒"，丢了不致命；
// 而且不需要建表、查询也快）：
//   notify:list:{用户ID}    LIST    最近的通知（JSON 字符串，最新的在最前）
//   notify:unread:{用户ID}  STRING  未读计数
// ============================================================

const (
	listKeyPrefix   = "notify:list:"
	unreadKeyPrefix = "notify:unread:"
	maxKeep         = 50 // 每个用户最多留 50 条，更旧的自动滚掉
)

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

// Add 给某个用户加一条通知（未读数 +1）
func Add(userID uint, item Item) error {
	ctx := context.Background()
	uid := strconv.FormatUint(uint64(userID), 10)
	listKey := listKeyPrefix + uid

	body, err := json.Marshal(item)
	if err != nil {
		return err
	}
	// LPUSH：塞到列表头部（最新的排最前，前端直接展示就行）
	if err := database.RDB.LPush(ctx, listKey, body).Err(); err != nil {
		return err
	}
	// LTRIM：只保留前 maxKeep 条，后面的砍掉（防止列表无限变长吃内存）
	if err := database.RDB.LTrim(ctx, listKey, 0, maxKeep-1).Err(); err != nil {
		return err
	}
	// 未读计数 +1
	return database.RDB.Incr(ctx, unreadKeyPrefix+uid).Err()
}

// List 读"从第 offset 条开始的 limit 条"通知 + 未读数
// （Redis 的 LIST 天然支持按下标取，分页就是换个下标的事）
func List(userID uint, offset, limit int) ([]Item, int64, error) {
	ctx := context.Background()
	uid := strconv.FormatUint(uint64(userID), 10)

	// LRANGE start stop：取列表下标 [start, stop] 的闭区间
	// 第 1 页 offset=0；第 2 页 offset=20……
	raw, err := database.RDB.LRange(ctx, listKeyPrefix+uid, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, 0, err
	}
	items := []Item{} // 老规矩：空也要返回 []，不能是 null
	for _, s := range raw {
		var it Item
		if err := json.Unmarshal([]byte(s), &it); err != nil {
			continue // 单条坏了就跳过，别让整个列表读不出来
		}
		items = append(items, it)
	}

	// 未读数：key 不存在时 Get 会返回 redis.Nil 错误，这里当作 0
	unread, err := database.RDB.Get(ctx, unreadKeyPrefix+uid).Int64()
	if err != nil {
		unread = 0
	}
	return items, unread, nil
}

// HandleFailed 死信队列的消费者：处理失败的消息会流到这里
// 这里只记录（方便排查），也可以在这里做告警或延迟重试
func HandleFailed(body []byte) error {
	slog.Error("收到死信（消息处理失败的原始内容）", "body", string(body))
	return nil // 记录完就算处理成功，避免"死信的死信"无限套娃
}

// MarkRead 把未读数清零（用户点开看过了）
func MarkRead(userID uint) error {
	uid := strconv.FormatUint(uint64(userID), 10)
	return database.RDB.Del(context.Background(), unreadKeyPrefix+uid).Err()
}
