# ============================================================
# 前端（Vue）的 Docker 镜像配方
#
# 两段：
#   第一段用 Node 把 Vue 打包成纯静态文件（HTML/JS/CSS）
#   第二段用 Nginx 把这些静态文件对外提供服务，并反向代理后端接口
# ============================================================

# ---------- 第一段：打包 ----------
FROM node:24-alpine AS builder

WORKDIR /app

# 先拷依赖清单，依赖没变就能复用缓存
COPY package.json package-lock.json ./

# 安装依赖：
#   ci          严格按照 package-lock.json 安装（版本完全一致，线上更可靠）
#   --registry  走 npmmirror 国内镜像，比 npm 官方源快很多
RUN npm ci --registry=https://registry.npmmirror.com

# 拷入源码和配置
COPY . .

# 执行打包：产物在 /app/dist 目录（一堆静态文件）
RUN npm run build

# ---------- 第二段：运行 ----------
FROM nginx:alpine

# 把打包结果放进 Nginx 的网站根目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 用我们自己的配置替换 Nginx 默认配置（里面有反向代理规则）
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 容器监听 80 端口
EXPOSE 80
