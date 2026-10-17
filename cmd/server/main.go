package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/handler"
	"blog/internal/logger"
	"blog/internal/mq"
	"blog/internal/notify"
	"blog/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// ① 先读配置：端口、数据库、日志级别这些都写在 .env 里
	config.Load()
	// ② 再装日志 —— 顺序不能反！
	//    日志级别要从 .env 里读，所以必须先有配置、才有日志。
	//    ParseLevel 负责把 "debug" 这种字符串翻译成 slog 认识的级别。
	logger.Init(logger.ParseLevel(config.Cfg.LogLevel))
	// /internal/database  连接mysql
	database.ConnectDB()
	// /internal/database   连接redis
	database.ConnectRedis()
	// /internal/database  自动建表
	database.AutoMigrate()
	// /internal/handler  创建验证码生成器
	handler.InitCaptcha()
	// 连接 RabbitMQ 并启动"通知消费者"
	// 注意：连不上不致命 —— 博客照常跑，只是少了未读提醒这个功能。
	// （启动顺序也有讲究：消费者要用 Redis，所以得在 ConnectRedis 之后）
	if err := mq.Connect(config.Cfg.RabbitMQURL); err != nil {
		slog.Warn("RabbitMQ 连接失败，通知功能暂不可用", "err", err)
	} else if err := mq.Consume(mq.QueueNotify, notify.HandleMessage); err != nil {
		slog.Warn("通知消费者启动失败", "err", err)
	} else {
		// 再起一个"死信消费者"：只负责把处理失败的消息记进日志，方便排查
		_ = mq.Consume(mq.QueueNotifyFailed, notify.HandleFailed)
		slog.Info("RabbitMQ 已连接，通知消费者已启动（含死信队列）")
	}
	// uploads 及各分类子目录不存在就自动创建
	// （资源分类管理：头像/说说/相册各回各家，MkdirAll 已存在也不报错）
	for _, dir := range []string{"uploads", "uploads/avatar", "uploads/talk", "uploads/album"} {
		_ = os.MkdirAll(dir, 0755)
	}

	// 运行模式：gin 只认 debug / release / test 三种，写错了它会直接 panic，
	// 所以这里自己先过一道筛子（认不出来就保持默认的 debug）
	if config.Cfg.GinMode == gin.ReleaseMode || config.Cfg.GinMode == gin.TestMode {
		gin.SetMode(config.Cfg.GinMode)
	}

	// gin.Default() 自带两个中间件：Logger（控制台访问日志）+ Recovery（panic 兜底）
	r := gin.Default()
	// 上传缓冲调大：默认32MB，传大视频时减少写临时文件的次数
	r.MaxMultipartMemory = 128 << 20 // 128MB
	// 关掉"信任所有代理"的默认行为：本机直连用不着它，
	// 不然每次启动都要打一句安全警告（以后挂到 Nginx 后面再按需配置）
	_ = r.SetTrustedProxies(nil)

	// 所有路由统统交给 /internal/router 统一登记（main.go 从此只管"启动流程"）
	router.Setup(r)

	// ---------- 优雅关闭 ----------
	// 原来的 r.Run() 一按 Ctrl+C 就"咔"地掐断所有连接，
	// 正在处理的请求会拿到半截响应、写到一半的数据也可能丢。
	// 优雅关闭 = 退出前先把"还在手上的活儿"干完。
	// 做法：换成标准库的 http.Server（优雅关闭的方法长在它身上）
	srv := &http.Server{
		Addr:    ":" + config.Cfg.Port, // 监听哪个端口
		Handler: r,                     // 来了请求交给 gin 处理
	}

	// 服务得"在后台跑"：另开一个 goroutine 去监听。
	// 不这么做的话，主线程会一直卡在 ListenAndServe 上，
	// 下面"等退出信号"的代码永远轮不到执行
	go func() {
		slog.Info("服务已启动", "port", config.Cfg.Port)
		// 正常关闭时 ListenAndServe 会返回 ErrServerClosed —— 那是预期结果，不算错误
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("服务启动失败", "err", err)
		}
	}()

	// 主线程在这里"趴着等信号"：Ctrl+C（SIGINT）或系统要求退出（SIGTERM）
	quit := make(chan os.Signal, 1)                      // 造一个"收件箱"，专门接信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 订阅这两种信号
	<-quit                                               // 阻塞在这儿，收到信号才往下走
	slog.Info("收到退出信号，开始优雅关闭…")

	// 给"正在处理中的请求"最多 5 秒做完；5 秒还没完就放弃。
	// （不设时限的话，万一有个请求卡死，你就永远退不出去了）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 函数结束时释放定时器（好习惯，防泄漏）
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("优雅关闭超时，强制退出", "err", err)
	}

	// 收尾：把数据库 / Redis / RabbitMQ 连接也关掉，干干净净地走
	if sqlDB, err := database.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = database.RDB.Close()
	mq.Close()

	slog.Info("已安全退出，拜拜 👋")
}
