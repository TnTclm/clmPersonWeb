package database

import (
	"blog/internal/config"
	"blog/internal/logger"
	"blog/internal/model"
	"fmt"
	"log/slog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 全局DB
var DB *gorm.DB

// 连接mysql函数
func ConnectDB() {
	cfg := config.Cfg
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		TranslateError: true, // 把底层数据库错误翻译成 GORM 的标准错误（如 1062 → ErrDuplicatedKey）
	})
	if err != nil {
		logger.Fatal("连接mysql失败", "err:", err)
	}
	DB = db
	slog.Info("连接mysql成功!")
}

// autoMigrate   自动建表：表不存在就创建，存在但字段有出入就补齐（不会删数据）
func AutoMigrate() {
	err := DB.AutoMigrate(&model.User{}, &model.Comment{}, &model.Talk{}, &model.AlbumPost{}, &model.AlbumItem{}, &model.Message{}, &model.Activity{})
	if err != nil {
		logger.Fatal("mysql自动建表失败", "err:", err)
	}
	slog.Info("mysql数据表准备就绪!")
}
