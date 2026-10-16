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
	OwnerID     int    // 博主的用户ID：相册上传/删除只有TA可以（.env 里配 OWNER_ID）
	PicturesDir string // 主页图片墙的图片文件夹（.env 里配 PICTURES_DIR）
	LogLevel    string // 日志级别：debug/info/warn/error（.env 里配 LOG_LEVEL）
	GinMode     string // gin 运行模式：debug/release（.env 里配 GIN_MODE）
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
		OwnerID:     getEnvInt("OWNER_ID", 4),             //
		PicturesDir: getEnv("PICTURES_DIR", "./pictures"), // 主页图片墙的图放哪（相对仓库根目录）
		LogLevel:    getEnv("LOG_LEVEL", "debug"),         // 日志级别：本地 debug，上线 info
		GinMode:     getEnv("GIN_MODE", "debug"),          // gin 模式：本地 debug，上线 release
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
