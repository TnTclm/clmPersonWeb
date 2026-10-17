# clmPersonWeb · 个人博客小站

一个从零手写的个人博客：**Go + Gin** 后端，**Vue 3 + Vite** 前端。
没有文章系统——只有「说说」：短文字 + 九宫格配图 + 点赞 + 评论。

> 在线地址：待部署
> 学习项目：边写边学 Go 后端 / Vue 前端，代码里带大量中文注释。

## ✨ 功能

| 模块 | 说明 |
| --- | --- |
| 💬 **说说** | 全站唯一的内容形态：文字 + 最多 9 张图，支持点赞（Redis 双账本）、评论 |
| 📷 **相册** | 图片 / 视频混合发布，九宫格 + 点击预览 |
| 🌸 **留言板** | 弹幕墙：留言化作樱花雨里飞过的弹幕 |
| 🖼 **图片墙** | 主页把 `pictures/` 文件夹里的照片按文件名自然排序展示 |
| 🧑‍💻 **个人空间** | 昵称 / 签名 / 头像修改，LeetCode 式活跃度热力图 |
| 🔔 **未读提醒** | 有人评论 / 留言时，博主收到通知（RabbitMQ 异步 + 死信重试） |
| 🛡 **基础防护** | 图形验证码、发言限流（3 条/分 + 15 条/时）、JWT 鉴权 |

## 🧰 技术栈

**后端**

- Go 1.26 / Gin / GORM
- MySQL（业务数据）
- Redis（验证码、点赞、限流计数、缓存）
- RabbitMQ（异步通知，含重试队列与死信队列）
- JWT（HS256，7 天有效）

**前端**

- Vue 3 / Vite / Vue Router / Element Plus
- 无状态库（共享登录态用单例 ref）

## 📁 目录结构

```
clmPersonWeb/
├── cmd/server/main.go        # 启动入口（配置 → 日志 → 依赖 → 路由 → 优雅关闭）
├── internal/
│   ├── config/               # .env 配置加载
│   ├── database/             # MySQL / Redis 连接与自动建表
│   ├── logger/               # slog 日志封装（级别可配）
│   ├── middleware/           # JWT 鉴权
│   ├── mq/                   # RabbitMQ 封装（发布 / 消费 / 重试 / 死信）
│   ├── notify/               # 通知存取与消费逻辑
│   ├── router/               # 全站路由登记
│   └── handler/              # 各业务接口
├── frontend/                 # Vue 3 前端
│   └── src/
│       ├── views/            # 页面
│       ├── components/       # 公共组件
│       └── views/css/        # 页面样式（外置）
├── uploads/                  # 用户上传（头像 / 说说配图 / 相册媒体）—— 不进 Git
├── pictures/                 # 主页图片墙的照片 —— 不进 Git
└── .env.example              # 环境变量模板
```

## 🚀 本地运行

### 1. 准备依赖

需要本机有（用 Docker 起最省事）：

```bash
# MySQL（端口 3306）、Redis（6379）、RabbitMQ（5672 + 管理后台 15672）
docker run -d --name mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=123456 mysql:8
docker run -d --name redis -p 6379:6379 redis:7
docker run -d --name my-rabbit -p 5672:5672 -p 15672:15672 rabbitmq:management
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 然后按自己的数据库账号密码改一下 .env
```

关键变量：

| 变量 | 说明 |
| --- | --- |
| `DB_*` | MySQL 连接信息 |
| `REDIS_ADDR` | Redis 地址 |
| `RABBITMQ_URL` | RabbitMQ 地址 |
| `JWT_SECRET` | JWT 签名密钥（**上线务必换掉**） |
| `OWNER_ID` | 博主的用户 ID（发说说/发相册的权限给 TA） |
| `PICTURES_DIR` | 主页图片墙的照片目录（默认 `./pictures`） |
| `LOG_LEVEL` | 日志级别：`debug`（本地）/ `info`（线上） |
| `GIN_MODE` | `debug`（本地）/ `release`（线上） |

### 3. 启动后端

```bash
# 必须在仓库根目录跑（uploads、pictures 都是相对路径）
go run ./cmd/server
```

看到 `服务已启动` 和 `RabbitMQ 已连接，通知消费者已启动` 就 OK。

### 4. 启动前端

```bash
cd frontend
npm install
npm run dev
```

打开 http://localhost:5173（`/api`、`/uploads`、`/pictures` 由 Vite 代理到 8080）。

## 🗺 路线图

- [x] 说说 / 评论 / 点赞
- [x] 相册（图片 + 视频）
- [x] 留言板弹幕
- [x] 个人空间 + 热力图
- [x] 主页图片墙
- [x] RabbitMQ 异步通知（含重试 / 死信队列）
- [x] 列表分页（`?page=&size=`，返回 `has_more`）
- [ ] 部署上线：Docker + Nginx + HTTPS

## 📄 说明

- `.env`、`uploads/`、`pictures/` 都在 `.gitignore` 里，**不会提交**（分别是密钥、用户内容、个人照片）。
- 本项目是学习项目，代码里写满了「为什么这么写」的中文注释。
