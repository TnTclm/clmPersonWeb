package main

import (
	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/handler"
	"blog/internal/logger"
	"blog/internal/router"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 第一件事：装日志。后面每一步的动静都要靠它记录
	logger.Init()
	// /internal/config 加载全局变量 Cfg
	config.Load()
	// /internal/database  连接mysql
	database.ConnectDB()
	// /internal/database   连接redis
	database.ConnectRedis()
	// /internal/database  自动建表
	database.AutoMigrate()
	// /internal/handler  创建验证码生成器
	handler.InitCaptcha()
	// uploads 及各分类子目录不存在就自动创建
	// （资源分类管理：头像/说说/相册各回各家，MkdirAll 已存在也不报错）
	for _, dir := range []string{"uploads", "uploads/avatar", "uploads/talk", "uploads/album"} {
		_ = os.MkdirAll(dir, 0755)
	}
	// gin.Default() 自带两个中间件：Logger（控制台访问日志）+ Recovery（panic 兜底）
	r := gin.Default()
	// 上传缓冲调大：默认32MB，传大视频时减少写临时文件的次数
	r.MaxMultipartMemory = 128 << 20 // 128MB
	// 所有路由统统交给 /internal/router 统一登记（main.go 从此只管"启动流程"）
	router.Setup(r)
	// 最后一步：监听端口跑起来，这行会一直阻塞在这里（直到 Ctrl+C）
	r.Run(":" + config.Cfg.Port)
}
