# 🚀 NOFX Linux 服务器完整部署指南

本指南将帮助你将 NOFX 完整部署到 Linux 服务器（Ubuntu/Debian/CentOS）。

---

## 📋 目录

1. [服务器要求](#服务器要求)
2. [准备工作](#准备工作)
3. [安装依赖](#安装依赖)
4. [部署步骤](#部署步骤)
5. [配置反向代理](#配置反向代理)
6. [配置 HTTPS](#配置-https)
7. [系统服务配置](#系统服务配置)
8. [监控与维护](#监控与维护)
9. [故障排查](#故障排查)

---

## 📊 服务器要求

### 最低配置
- **CPU**: 2 核
- **内存**: 4GB RAM
- **硬盘**: 20GB 可用空间
- **系统**: Ubuntu 20.04+ / Debian 11+ / CentOS 8+
- **网络**: 公网 IP（可选，用于远程访问）

### 推荐配置
- **CPU**: 4 核或更多
- **内存**: 8GB RAM 或更多
- **硬盘**: 50GB SSD
- **带宽**: 10Mbps 或更高

---

## 🔧 准备工作

### 1. 连接到服务器

```bash
# 使用 SSH 连接到服务器
ssh root@your-server-ip

# 或使用普通用户
ssh username@your-server-ip
```

### 2. 更新系统

```bash
# Ubuntu/Debian
sudo apt update && sudo apt upgrade -y

# CentOS/RHEL
sudo yum update -y
```

### 3. 创建部署用户（推荐）

```bash
# 创建专用用户
sudo useradd -m -s /bin/bash nofx

# 设置密码
sudo passwd nofx

# 添加 sudo 权限
sudo usermod -aG sudo nofx

# 切换到 nofx 用户
su - nofx
```

---

## 🛠️ 安装依赖

### 1. 安装 Docker

#### Ubuntu/Debian

```bash
# 卸载旧版本
sudo apt-get remove docker docker-engine docker.io containerd runc

# 安装依赖
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# 添加 Docker 官方 GPG 密钥
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# 设置 Docker 仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker Engine
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 验证安装
docker --version
docker compose version
```

#### CentOS/RHEL

```bash
# 卸载旧版本
sudo yum remove docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine

# 安装依赖
sudo yum install -y yum-utils

# 添加 Docker 仓库
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# 安装 Docker Engine
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 验证安装
docker --version
docker compose version
```

### 2. 配置 Docker 权限

```bash
# 将当前用户添加到 docker 组
sudo usermod -aG docker $USER

# 重新登录或运行以下命令使权限生效
newgrp docker

# 测试 Docker（无需 sudo）
docker run hello-world
```

### 3. 安装 Git

```bash
# Ubuntu/Debian
sudo apt-get install -y git

# CentOS/RHEL
sudo yum install -y git

# 验证安装
git --version
```

---

## 🚀 部署步骤

### 1. 克隆项目

```bash
# 进入工作目录
cd ~

# 克隆项目（替换为你的仓库地址）
git clone https://github.com/your-username/nofx.git

# 进入项目目录
cd nofx
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量
nano .env
```

**编辑 `.env` 文件：**

```bash
# 端口配置
NOFX_BACKEND_PORT=9000      # 后端端口（避免与其他服务冲突）
NOFX_FRONTEND_PORT=3000     # 前端端口

# 时区设置
NOFX_TIMEZONE=Asia/Shanghai

# 数据库加密密钥（必须设置，用于加密敏感数据）
DATA_ENCRYPTION_KEY=your-32-character-encryption-key-here

# JWT 密钥（必须设置，用于用户认证）
JWT_SECRET=your-64-character-jwt-secret-key-here
```

**生成安全密钥：**

```bash
# 生成 DATA_ENCRYPTION_KEY（32字符）
openssl rand -base64 32

# 生成 JWT_SECRET（64字符）
openssl rand -base64 64
```

### 3. 配置 config.json

```bash
# 复制配置模板
cp config.json.example config.json

# 编辑配置文件
nano config.json
```

**基本配置示例：**

```json
{
  "beta_mode": false,
  "api_server_port": 8080,
  "use_default_coins": true,
  "default_coins": [],
  "coin_pool_api_url": "http://localhost:5001/api/ai500",
  "oi_top_api_url": "http://localhost:5001/api/oi-top",
  "max_daily_loss": 5.0,
  "max_drawdown": 10.0,
  "stop_trading_minutes": 30,
  "leverage": {
    "btc_eth_leverage": 3,
    "altcoin_leverage": 2
  },
  "jwt_secret": "your-jwt-secret-from-env",
  "data_k_line_time": "5m",
  "log": {
    "level": "info",
    "telegram": {
      "enabled": false,
      "bot_token": "",
      "chat_id": 0,
      "min_level": "error"
    }
  }
}
```

### 4. 配置 coin-api（如果需要）

```bash
# 进入 coin-api 目录
cd ~/nofx-coin-api

# 如果没有，需要克隆
git clone https://github.com/your-username/nofx-coin-api.git
cd nofx-coin-api

# 启动 coin-api
docker-compose up -d
```

### 5. 启动 NOFX 服务

```bash
# 返回 nofx 目录
cd ~/nofx

# 构建并启动服务
docker-compose up -d --build

# 查看启动日志
docker-compose logs -f
```

### 6. 验证部署

```bash
# 检查容器状态
docker-compose ps

# 应该看到类似输出：
# NAME            STATUS              PORTS
# nofx-trading    Up (healthy)        0.0.0.0:9000->8080/tcp
# nofx-frontend   Up (healthy)        0.0.0.0:3000->80/tcp

# 测试后端 API
curl http://localhost:9000/api/health

# 测试前端
curl http://localhost:3000
```

---

## 🌐 配置反向代理（Nginx）

### 1. 安装 Nginx

```bash
# Ubuntu/Debian
sudo apt-get install -y nginx

# CentOS/RHEL
sudo yum install -y nginx

# 启动 Nginx
sudo systemctl start nginx
sudo systemctl enable nginx
```

### 2. 配置 Nginx

```bash
# 创建配置文件
sudo nano /etc/nginx/sites-available/nofx
```

**Nginx 配置内容：**

```nginx
# HTTP 配置（稍后升级到 HTTPS）
server {
    listen 80;
    server_name your-domain.com;  # 替换为你的域名或服务器 IP

    # 前端
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # 后端 API
    location /api/ {
        proxy_pass http://localhost:9000/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # 增加超时时间（AI 决策可能需要较长时间）
        proxy_read_timeout 300s;
        proxy_connect_timeout 300s;
        proxy_send_timeout 300s;
    }

    # 日志
    access_log /var/log/nginx/nofx_access.log;
    error_log /var/log/nginx/nofx_error.log;
}
```

### 3. 启用配置

```bash
# Ubuntu/Debian
sudo ln -s /etc/nginx/sites-available/nofx /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重启 Nginx
sudo systemctl restart nginx
```

### 4. 配置防火墙

```bash
# Ubuntu/Debian (UFW)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

---

## 🔒 配置 HTTPS（Let's Encrypt）

### 1. 安装 Certbot

```bash
# Ubuntu/Debian
sudo apt-get install -y certbot python3-certbot-nginx

# CentOS/RHEL
sudo yum install -y certbot python3-certbot-nginx
```

### 2. 获取 SSL 证书

```bash
# 自动配置 HTTPS
sudo certbot --nginx -d your-domain.com

# 按照提示操作：
# 1. 输入邮箱
# 2. 同意服务条款
# 3. 选择是否重定向 HTTP 到 HTTPS（推荐选择 2）
```

### 3. 测试自动续期

```bash
# 测试续期（不会真正续期）
sudo certbot renew --dry-run

# 查看续期计划
sudo systemctl status certbot.timer
```

### 4. 验证 HTTPS

```bash
# 访问你的域名
https://your-domain.com

# 检查证书
curl -I https://your-domain.com
```

---

## 🔄 系统服务配置（开机自启）

### 1. 创建 systemd 服务

```bash
# 创建服务文件
sudo nano /etc/systemd/system/nofx.service
```

**服务配置内容：**

```ini
[Unit]
Description=NOFX AI Trading System
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/home/nofx/nofx
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
User=nofx
Group=nofx

[Install]
WantedBy=multi-user.target
```

### 2. 启用服务

```bash
# 重新加载 systemd
sudo systemctl daemon-reload

# 启用服务（开机自启）
sudo systemctl enable nofx

# 启动服务
sudo systemctl start nofx

# 查看状态
sudo systemctl status nofx
```

### 3. 管理服务

```bash
# 启动
sudo systemctl start nofx

# 停止
sudo systemctl stop nofx

# 重启
sudo systemctl restart nofx

# 查看日志
sudo journalctl -u nofx -f
```

---

## 📊 监控与维护

### 1. 查看日志

```bash
# 实时查看 Docker 日志
cd ~/nofx
docker-compose logs -f

# 查看 Nginx 日志
sudo tail -f /var/log/nginx/nofx_access.log
sudo tail -f /var/log/nginx/nofx_error.log

# 查看系统日志
sudo journalctl -u nofx -f
```

### 2. 监控资源使用

```bash
# 查看容器资源使用
docker stats

# 查看系统资源
htop  # 需要安装: sudo apt-get install htop

# 查看磁盘使用
df -h
du -sh ~/nofx/*
```

### 3. 定期备份

```bash
# 创建备份脚本
nano ~/backup-nofx.sh
```

**备份脚本内容：**

```bash
#!/bin/bash

# 配置
BACKUP_DIR="/home/nofx/backups"
NOFX_DIR="/home/nofx/nofx"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据
tar -czf $BACKUP_DIR/nofx_backup_$DATE.tar.gz \
    $NOFX_DIR/config.json \
    $NOFX_DIR/config.db \
    $NOFX_DIR/decision_logs \
    $NOFX_DIR/.env

# 删除 7 天前的备份
find $BACKUP_DIR -name "nofx_backup_*.tar.gz" -mtime +7 -delete

echo "备份完成: nofx_backup_$DATE.tar.gz"
```

**设置定时备份：**

```bash
# 添加执行权限
chmod +x ~/backup-nofx.sh

# 编辑 crontab
crontab -e

# 添加每天凌晨 2 点备份
0 2 * * * /home/nofx/backup-nofx.sh >> /home/nofx/backup.log 2>&1
```

### 4. 更新系统

```bash
# 拉取最新代码
cd ~/nofx
git pull

# 重新构建并启动
docker-compose up -d --build

# 查看日志确认更新成功
docker-compose logs -f
```

---

## 🐛 故障排查

### 1. 容器无法启动

```bash
# 查看详细日志
docker-compose logs backend
docker-compose logs frontend

# 检查配置文件
cat config.json | jq .  # 验证 JSON 格式

# 重新构建（清除缓存）
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### 2. 端口冲突

```bash
# 查看端口占用
sudo netstat -tlnp | grep :9000
sudo netstat -tlnp | grep :3000

# 或使用 lsof
sudo lsof -i :9000
sudo lsof -i :3000

# 修改端口（编辑 .env 文件）
nano .env
```

### 3. 网络连接问题

```bash
# 测试 Binance 连接
curl -I https://fapi.binance.com

# 检查 DNS
nslookup fapi.binance.com

# 如果在中国大陆，可能需要配置代理
# 编辑 docker-compose.yml 添加代理设置
```

### 4. 数据库问题

```bash
# 检查数据库文件
ls -lh ~/nofx/config.db

# 备份并重置数据库
cp ~/nofx/config.db ~/nofx/config.db.backup
rm ~/nofx/config.db
docker-compose restart
```

### 5. 内存不足

```bash
# 查看内存使用
free -h

# 清理 Docker 资源
docker system prune -a

# 限制容器内存（编辑 docker-compose.yml）
services:
  nofx:
    deploy:
      resources:
        limits:
          memory: 2G
```

---

## 📝 常用命令速查

```bash
# === Docker 管理 ===
docker-compose up -d --build      # 构建并启动
docker-compose down               # 停止并删除容器
docker-compose restart            # 重启服务
docker-compose logs -f            # 查看日志
docker-compose ps                 # 查看状态

# === 系统服务 ===
sudo systemctl start nofx         # 启动服务
sudo systemctl stop nofx          # 停止服务
sudo systemctl restart nofx       # 重启服务
sudo systemctl status nofx        # 查看状态

# === Nginx ===
sudo nginx -t                     # 测试配置
sudo systemctl restart nginx      # 重启 Nginx
sudo tail -f /var/log/nginx/nofx_error.log  # 查看错误日志

# === 更新 ===
cd ~/nofx && git pull && docker-compose up -d --build

# === 备份 ===
~/backup-nofx.sh                  # 手动备份

# === 清理 ===
docker system prune -a            # 清理 Docker 资源
```

---

## 🎯 安全检查清单

- [ ] 修改默认端口
- [ ] 设置强密码的 JWT_SECRET
- [ ] 设置强密码的 DATA_ENCRYPTION_KEY
- [ ] 配置防火墙规则
- [ ] 启用 HTTPS
- [ ] 定期备份数据
- [ ] 限制 SSH 访问（使用密钥认证）
- [ ] 配置日志轮转
- [ ] 监控系统资源
- [ ] 定期更新系统和 Docker

---

## 🆘 获取帮助

- **GitHub Issues**: [提交问题](https://github.com/your-username/nofx/issues)
- **Telegram 社区**: [NOFX Developer Community](https://t.me/nofx_dev_community)
- **文档**: 查看 [README.md](README.md)

---

🎉 **恭喜！你已经成功将 NOFX 部署到 Linux 服务器！**

现在你可以通过域名或 IP 地址访问你的交易系统了。

**访问地址：**
- HTTP: `http://your-domain.com` 或 `http://your-server-ip`
- HTTPS: `https://your-domain.com`（配置 SSL 后）

**下一步：**
1. 访问 Web 界面
2. 注册账户
3. 配置 AI 模型（DeepSeek/Qwen）
4. 配置交易所（Binance/Hyperliquid/Aster）
5. 创建交易员
6. 开始交易！
