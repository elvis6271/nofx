# 🚀 NOFX 快速部署到 Linux 服务器

## 方式一：一键部署脚本（推荐）

### 1. 准备工作

确保你的本地机器可以 SSH 连接到服务器：

```bash
# 测试 SSH 连接
ssh root@your-server-ip

# 或使用自定义端口
ssh -p 22 root@your-server-ip
```

### 2. 运行部署脚本

在本地项目目录下运行：

```bash
# 基本用法
./deploy-to-server.sh <服务器IP> <用户名>

# 示例1: 使用 root 用户，默认 22 端口
./deploy-to-server.sh 192.168.1.100 root

# 示例2: 使用普通用户
./deploy-to-server.sh 192.168.1.100 ubuntu

# 示例3: 使用自定义端口
./deploy-to-server.sh 192.168.1.100 root 2222
```

脚本会自动：
- ✅ 检查 SSH 连接
- ✅ 检查并安装 Docker（如需要）
- ✅ 同步代码到服务器
- ✅ 配置环境变量
- ✅ 构建并启动服务
- ✅ 配置防火墙（可选）

### 3. 访问服务

部署完成后：
- 前端: `http://your-server-ip:3000`
- 后端: `http://your-server-ip:8080`

---

## 方式二：手动部署

### 1. 连接到服务器

```bash
ssh root@your-server-ip
```

### 2. 安装 Docker

#### Ubuntu/Debian

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | bash

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 添加当前用户到 docker 组
sudo usermod -aG docker $USER

# 重新登录使组权限生效
exit
ssh root@your-server-ip
```

#### CentOS/RHEL

```bash
# 安装 Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker
```

### 3. 上传代码

在本地机器上：

```bash
# 方式1: 使用 rsync（推荐）
rsync -avz --progress \
    --exclude '.git' \
    --exclude 'node_modules' \
    --exclude 'data' \
    ./ root@your-server-ip:/root/nofx/

# 方式2: 使用 scp
scp -r ./ root@your-server-ip:/root/nofx/

# 方式3: 使用 Git（如果服务器可以访问 GitHub）
# 在服务器上执行：
git clone https://github.com/your-username/nofx.git
cd nofx
git checkout feature/custom-rsi-chandelier
```

### 4. 配置环境变量

在服务器上：

```bash
cd /root/nofx

# 复制环境变量模板
cp .env.example .env

# 编辑环境变量
nano .env
```

修改以下配置：

```env
# 端口配置
NOFX_BACKEND_PORT=8080
NOFX_FRONTEND_PORT=3000

# 时区
NOFX_TIMEZONE=Asia/Shanghai

# 数据库路径（可选）
NOFX_DATA_DIR=./data
```

### 5. 启动服务

```bash
# 构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f

# 查看服务状态
docker compose ps
```

### 6. 配置防火墙

#### Ubuntu/Debian (UFW)

```bash
sudo ufw allow 3000/tcp
sudo ufw allow 8080/tcp
sudo ufw status
```

#### CentOS/RHEL (firewalld)

```bash
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
sudo firewall-cmd --list-ports
```

---

## 常用命令

### 查看日志

```bash
# 查看所有日志
docker compose logs -f

# 查看后端日志
docker compose logs -f nofx

# 查看前端日志
docker compose logs -f nofx-frontend

# 查看最近 100 行日志
docker compose logs --tail 100
```

### 重启服务

```bash
# 重启所有服务
docker compose restart

# 重启后端
docker compose restart nofx

# 重启前端
docker compose restart nofx-frontend
```

### 停止服务

```bash
# 停止服务
docker compose down

# 停止并删除数据卷
docker compose down -v
```

### 更新代码

```bash
# 方式1: 从本地推送更新
# 在本地执行
./deploy-to-server.sh your-server-ip root

# 方式2: 在服务器上拉取更新
cd /root/nofx
git pull
docker compose up -d --build
```

### 查看资源使用

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
docker system df

# 清理未使用的镜像
docker system prune -a
```

---

## 配置 HTTPS（可选）

### 使用 Nginx + Let's Encrypt

1. 安装 Nginx

```bash
# Ubuntu/Debian
sudo apt install -y nginx

# CentOS/RHEL
sudo yum install -y nginx
```

2. 配置 Nginx

```bash
sudo nano /etc/nginx/sites-available/nofx
```

添加配置：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端
    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # 后端 API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

3. 启用配置

```bash
sudo ln -s /etc/nginx/sites-available/nofx /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

4. 安装 SSL 证书

```bash
# 安装 Certbot
sudo apt install -y certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期
sudo certbot renew --dry-run
```

---

## 故障排查

### 服务无法启动

```bash
# 查看详细日志
docker compose logs

# 检查端口占用
sudo netstat -tulpn | grep -E '3000|8080'

# 检查 Docker 状态
sudo systemctl status docker
```

### 无法访问服务

```bash
# 检查防火墙
sudo ufw status
sudo firewall-cmd --list-ports

# 检查服务是否运行
docker compose ps

# 测试端口连通性
curl http://localhost:3000
curl http://localhost:8080/api/health
```

### 内存不足

```bash
# 查看内存使用
free -h

# 限制容器内存
# 编辑 docker-compose.yml
services:
  nofx:
    deploy:
      resources:
        limits:
          memory: 2G
```

---

## 生产环境建议

1. **使用域名和 HTTPS**
   - 配置域名解析
   - 使用 Let's Encrypt 免费 SSL 证书

2. **配置自动备份**
   ```bash
   # 创建备份脚本
   cat > /root/backup-nofx.sh << 'EOF'
   #!/bin/bash
   DATE=$(date +%Y%m%d_%H%M%S)
   tar -czf /root/backups/nofx_$DATE.tar.gz /root/nofx/data
   # 保留最近 7 天的备份
   find /root/backups -name "nofx_*.tar.gz" -mtime +7 -delete
   EOF
   
   chmod +x /root/backup-nofx.sh
   
   # 添加到 crontab（每天凌晨 2 点备份）
   (crontab -l 2>/dev/null; echo "0 2 * * * /root/backup-nofx.sh") | crontab -
   ```

3. **监控服务状态**
   ```bash
   # 创建监控脚本
   cat > /root/monitor-nofx.sh << 'EOF'
   #!/bin/bash
   if ! docker compose ps | grep -q "Up"; then
       echo "NOFX service is down, restarting..."
       cd /root/nofx && docker compose restart
   fi
   EOF
   
   chmod +x /root/monitor-nofx.sh
   
   # 每 5 分钟检查一次
   (crontab -l 2>/dev/null; echo "*/5 * * * * /root/monitor-nofx.sh") | crontab -
   ```

4. **配置日志轮转**
   ```bash
   # Docker 日志配置
   # 编辑 /etc/docker/daemon.json
   {
     "log-driver": "json-file",
     "log-opts": {
       "max-size": "10m",
       "max-file": "3"
     }
   }
   
   sudo systemctl restart docker
   ```

---

## 需要帮助？

- 查看完整文档: [LINUX_DEPLOY_GUIDE.md](./LINUX_DEPLOY_GUIDE.md)
- 查看 Docker 文档: [docs/getting-started/docker-deploy.zh-CN.md](./docs/getting-started/docker-deploy.zh-CN.md)
- GitHub Issues: https://github.com/your-username/nofx/issues
