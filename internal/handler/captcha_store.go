package handler

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// 图像验证码数据前缀   方便和其它数据区分
const captchaKeyPrefix = "captcha:"

// 实现base64Captcha 接口  Set Get Verify
type RedisStore struct {
	client *redis.Client
	expire time.Duration
}

// New RedisClient
func NewRedisStore(client *redis.Client, expire time.Duration) *RedisStore {
	return &RedisStore{client: client, expire: expire}
}

// Set
func (s *RedisStore) Set(id string, value string) error {
	ctx := context.Background()
	return s.client.Set(ctx, captchaKeyPrefix+id, value, s.expire).Err()
}

// Get
func (s *RedisStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := captchaKeyPrefix + id
	if clear {
		val, err := s.client.GetDel(ctx, key).Result()
		if err != nil {
			return ""
		}
		return val
	}
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return val
}

// Verify
func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	return s.Get(id, clear) == answer
}
