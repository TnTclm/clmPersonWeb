package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	RedisAddr   string
	RabbitMQURL string
	JWTSecret   string
	OwnerID     int // 博主的用户ID：相册上传/删除只有TA可以（.env 里配 OWNER_ID）
}

var Cfg *Config

func Load() {
	godotenv.Load()
	Cfg = &Config{
		Port:        getEnv("PORT", "8080"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBUser:      getEnv("DB_USER", "root"),
		DBPassword:  getEnv("DB_PASSWORD", "123456"),
		DBName:      getEnv("DB_NAME", "clm_blog"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://admin:123456@localhost:5672/"), // ← 新增
		JWTSecret:   getEnv("JWT_SECRET", "your_jwt_secret"),
		OwnerID:     getEnvInt("OWNER_ID", 4), //
	}
}

// getEnv：字符串版
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt：整数版（环境变量里全是字符串，要 Atoi 转一下）
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
