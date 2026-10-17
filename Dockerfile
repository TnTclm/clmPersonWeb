# ============================================================
# 后端（Go）的 Docker 镜像配方
#
# 思路：分两段（多阶段构建）
#   第一段用"带 Go 的完整环境"把源码编译成二进制
#   第二段只把编译结果搬到一个极小的系统里运行
# 好处：最终镜像里没有编译器、没有源码，体积小、更安全
# ============================================================

# ---------- 第一段：编译 ----------
# golang:1.26-alpine = 官方镜像，自带 Go 1.26 编译器，基于 alpine 精简系统
FROM golang:1.26-alpine AS builder

# 容器里的工作目录，后面的 COPY/RUN 都相对于它
WORKDIR /app

# 国内拉 Go 依赖走七牛云代理，否则直连 proxy.golang.org 会慢到超时
ENV GOPROXY=https://goproxy.cn,direct
# 如果镜像里的 Go 版本比 go.mod 要求的低，允许自动下载对应工具链
ENV GOTOOLCHAIN=auto

# 先只拷"依赖清单"两个文件
# 这样只要依赖没变，下面这条 go mod download 就能命中缓存，不用每次重新下载
COPY go.mod go.sum ./
# 下载全部第三方依赖（gin、gorm、redis 那些）
RUN go mod download

# 再拷全部源码进来（.dockerignore 会挡掉 .git、uploads 这些不需要的东西）
COPY . .

# 真正编译：
#   CGO_ENABLED=0  关闭 C 语言依赖 → 编出纯静态二进制，alpine 里不用装 libc
#   GOOS=linux     目标系统是 Linux（虽然构建本身也在 Linux 容器里，写上更明确）
#   -trimpath      去掉编译产物里的本机路径信息
#   -ldflags       去掉符号表和调试信息，二进制更小
#   -o /app/server 输出文件路径
#   ./cmd/server   要编译的入口包
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/server

# ---------- 第二段：运行 ----------
# 最终镜像：一个干净的 alpine（只有几 MB），只放编译好的程序
FROM alpine:latest

# tzdata          时区数据（不装的话 TZ=Asia/Shanghai 不生效，时间会是 UTC）
# ca-certificates HTTPS 证书（以后程序要访问外部 HTTPS 接口时用得上）
RUN apk add --no-cache tzdata ca-certificates

# 容器内的时区，保证日志和数据库时间是北京时间
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从第一段（builder）里把编译好的二进制"搬"过来
COPY --from=builder /app/server /app/server

# 声明容器会监听 8080（只是说明，真正的端口映射在 docker-compose.yml 里）
EXPOSE 8080

# 容器启动时执行什么命令
CMD ["/app/server"]
