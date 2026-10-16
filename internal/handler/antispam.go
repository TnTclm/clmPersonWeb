package handler

import (
	"blog/internal/database"
	"context"
	"log/slog"
	"strconv"
	"time"
)

const (
	speakLimitPerMinute = 3  // 每分钟最多3条：刷屏当场掐死
	speakLimitPerHour   = 15 // 每小时最多15条：慢速水军也拦住
)

// AllowSpeak 发言前过一道闸。true=放行，false=超限
// Redis 挂了怎么办？宁可放行也不能把正常网站搞挂——返回 true（fail open）
func AllowSpeak(userID uint) bool {
	ctx := context.Background()

	// 分钟桶：INCR 计数，第一条进来时才顺手设个过期时间
	minKey := "rate:min:" + strconv.FormatUint(uint64(userID), 10) + ":" + time.Now().Format("200601021504")
	n, err := database.RDB.Incr(ctx, minKey).Result()
	if err != nil {
		// Warn 而不是 Error：这不是"坏了"，是"降级了"——
		// 限流失效但网站照常跑（fail open）。
		// 级别选得准，以后配告警规则才不会天天半夜响。
		slog.Warn("频率检查redis出错，宁放行勿拦人", "err", err)
		return true
	}
	if n == 1 {
		database.RDB.Expire(ctx, minKey, 2*time.Minute) // 多给1分钟余量，防边界误伤
	}
	if n > speakLimitPerMinute {
		return false
	}

	// 小时桶：同一个套路
	hourKey := "rate:hour:" + strconv.FormatUint(uint64(userID), 10) + ":" + time.Now().Format("2006010215")
	hn, err := database.RDB.Incr(ctx, hourKey).Result()
	if err != nil {
		// 和分钟桶保持同一套日志：走 slog.Warn，键值对写 "err", err
		// （原来的 log.Println 输出没级别、没时间格式，两处风格不一致）
		slog.Warn("频率检查redis出错，宁放行勿拦人", "err", err)
		return true
	}
	if hn == 1 {
		database.RDB.Expire(ctx, hourKey, 2*time.Hour)
	}
	if hn > speakLimitPerHour {
		return false
	}
	return true
}
