package database

import (
	"blog/internal/config"
	"blog/internal/logger"
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func ConnectRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: config.Cfg.RedisAddr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := RDB.Ping(ctx).Err(); err != nil {
		logger.Fatal("连接redis失败", "err:", err)
	}
	slog.Info("连接redis成功!")
}
