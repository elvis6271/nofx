# NOFX 交易系统完整部署指南

本文档提供从本地开发到服务器部署的完整流程。

---

## 📋 目录

1. [本地清理与准备](#1-本地清理与准备)
2. [提交代码到 GitHub Fork](#2-提交代码到-github-fork)
3. [服务器环境准备](#3-服务器环境准备)
4. [服务器部署](#4-服务器部署)
5. [验证与测试](#5-验证与测试)
6. [常见问题排查](#6-常见问题排查)

---

## 1. 本地清理与准备

### 1.1 停止本地 Docker 容器

```bash
cd /Users/elvis/Documents/thumbgeekbot/nofx
docker compose down
```

### 1.2 清理 Docker 资源（可选，彻底清理）

```bash
# 删除所有停止的容器
docker container prune -f

# 删除未使用的镜像
docker image prune -a -f

# 删除未使用的卷
docker volume prune -f

# 删除未使用的网络
docker network prune -f
```

### 1.3 检查本地代码状态

```bash
# 查看当前分支
git branch

# 查看未提交的修改
git status

# 查看最近的提交记录
git log --oneline -5
```

---

## 2. 提交代码到 GitHub Fork

### 2.1 确认远程仓库配置

```bash
# 查看远程仓库
git remote -v

# 应该看到：
# origin    https://github.com/elvis6271/nofx.git (fetch)
# origin    https://github.com/elvis6271/nofx.git (push)
# upstream  https://github.com/NoFxAiOS/nofx.git (fetch)
# upstream  https://github.com/NoFxAiOS/nofx.git (push)
```

### 2.2 提交所有修改（如果有未提交的）

```bash
# 查看修改的文件
git status

# 添加所有修改
git add .

# 提交修改
git commit -m "feat: 添加新币做空策略和 Heikin Ashi 指标支持"

# 如果没有修改，跳过此步骤
```

### 2.3 推送到 GitHub Fork

```bash
# 推送当前分支到 origin（您的 Fork）
git push origin feature/custom-rsi-chandelier

# 如果是第一次推送，使用：
git push -u origin feature/custom-rsi-chandelier
```

### 2.4 合并到 dev 分支（推荐）

```bash
# 切换到 dev 分支
git checkout dev

# 拉取最新代码
git pull origin dev

# 合并 feature 分支
git merge feature/custom-rsi-chandelier

# 推送到 GitHub
git push origin dev
```

---

## 3. 服务器环境准备

### 3.1 SSH 连接到服务器

```bash
ssh your_username@your_server_ip
```

### 3.2 安装必要软件（如果尚未安装）

```bash
# 更新包管理器
sudo apt update

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Docker Compose
sudo apt install docker-compose-plugin -y

# 将当前用户添加到 docker 组（避免每次使用 sudo）
sudo usermod -aG docker $USER

# 重新登录以使组权限生效
exit
ssh your_username@your_server_ip

# 验证安装
docker --version
docker compose version
```

### 3.3 安装 Git（如果尚未安装）

```bash
sudo apt install git -y
git --version
```

---

## 4. 服务器部署

### 4.1 清理旧部署（如果存在）

```bash
# 进入项目目录（如果存在）
cd ~/nofx  # 或您的实际路径

# 停止并删除容器
docker compose down

# 删除旧的项目目录（谨慎操作！）
cd ~
rm -rf nofx

# 或者只删除容器和镜像，保留代码
docker compose down --rmi all --volumes
```

### 4.2 克隆代码

```bash
# 克隆您的 Fork（使用 dev 分支）
git clone -b dev https://github.com/elvis6271/nofx.git

# 进入项目目录
cd nofx

# 验证分支
git branch
```

### 4.3 创建配置文件

```bash
# 复制示例配置
cp config.json.example config.json

# 编辑配置文件
nano config.json
```

**需要修改的关键配置**：

```json
{
  "beta_mode": false,
  "registration_enabled": true,
  "leverage": {
    "btc_eth_leverage": 5,
    "altcoin_leverage": 5
  },
  "use_default_coins": false,
  "default_coins": [
    "BTCUSDT",
    "ETHUSDT",
    "SOLUSDT"
  ],
  "ai500_api_url": "http://your_signal_source:5001/api/ai500",
  "oi_top_api_url": "http://your_signal_source:5001/api/oi-top",
  "api_server_port": 8080,
  "max_daily_loss": 10.0,
  "max_drawdown": 20.0,
  "stop_trading_minutes": 60,
  "jwt_secret": "YOUR_RANDOM_SECRET_KEY_HERE",
  "log": {
    "level": "info"
  }
}
```

**生成安全的 JWT Secret**：

```bash
openssl rand -base64 64
```

### 4.4 构建并启动服务

```bash
# 构建并启动（后台运行）
docker compose up -d --build

# 查看构建日志
docker compose logs -f nofx
```

---

## 5. 验证与测试

### 5.1 检查容器状态

```bash
# 查看运行中的容器
docker ps

# 应该看到 nofx-trading 容器在运行
```

### 5.2 查看日志

```bash
# 实时查看日志
docker logs nofx-trading -f

# 查看最近 100 行日志
docker logs nofx-trading --tail 100
```

### 5.3 测试 API 接口

```bash
# 健康检查
curl http://localhost:8080/api/health

# 应该返回：{"status":"ok"}
```

---

## 6. 常见问题排查

### 6.1 config.json 是目录错误

**问题**：`read config.json: is a directory`

**解决方案**：

```bash
# 删除错误的目录
rm -rf config.json

# 重新创建文件
cp config.json.example config.json
nano config.json

# 重启容器
docker compose down
docker compose up -d
```

### 6.2 更新代码

**在服务器上拉取最新代码**：

```bash
cd ~/nofx

# 停止容器
docker compose down

# 拉取最新代码
git pull origin dev

# 重新构建并启动
docker compose up -d --build

# 查看日志
docker logs nofx-trading -f
```

---

## 🎯 完整部署流程总结

### **本地操作**

```bash
# 1. 进入项目目录
cd /Users/elvis/Documents/thumbgeekbot/nofx

# 2. 停止本地容器
docker compose down

# 3. 推送代码
git push origin feature/custom-rsi-chandelier
```

### **服务器操作**

```bash
# 1. SSH 连接
ssh your_username@your_server_ip

# 2. 克隆代码
git clone -b dev https://github.com/elvis6271/nofx.git
cd nofx

# 3. 创建配置
cp config.json.example config.json
nano config.json

# 4. 部署
docker compose up -d --build

# 5. 验证
docker logs nofx-trading -f
```
