#!/bin/bash

# NOFX Linux 服务器快速部署脚本
# 使用方法: ./deploy-to-server.sh <服务器IP> <用户名>
# 例如: ./deploy-to-server.sh 192.168.1.100 root

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查参数
if [ $# -lt 2 ]; then
    print_error "使用方法: $0 <服务器IP> <用户名> [端口]"
    echo "例如: $0 192.168.1.100 root"
    echo "或: $0 192.168.1.100 ubuntu 22"
    exit 1
fi

SERVER_IP=$1
SERVER_USER=$2
SERVER_PORT=${3:-22}
REMOTE_DIR="/home/$SERVER_USER/nofx"

print_info "开始部署 NOFX 到服务器..."
echo "服务器: $SERVER_USER@$SERVER_IP:$SERVER_PORT"
echo "目标目录: $REMOTE_DIR"
echo ""

# 1. 测试 SSH 连接
print_info "测试 SSH 连接..."
if ssh -p $SERVER_PORT -o ConnectTimeout=5 $SERVER_USER@$SERVER_IP "echo 'SSH 连接成功'" > /dev/null 2>&1; then
    print_success "SSH 连接正常"
else
    print_error "无法连接到服务器，请检查 IP、用户名和端口"
    exit 1
fi

# 2. 检查服务器上是否安装了 Docker
print_info "检查服务器 Docker 环境..."
if ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "command -v docker" > /dev/null 2>&1; then
    print_success "Docker 已安装"
else
    print_warning "服务器未安装 Docker"
    read -p "是否自动安装 Docker? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "安装 Docker..."
        ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "bash -s" << 'ENDSSH'
            # 检测系统类型
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                OS=$ID
            fi
            
            if [ "$OS" = "ubuntu" ] || [ "$OS" = "debian" ]; then
                # Ubuntu/Debian
                sudo apt-get update
                sudo apt-get install -y ca-certificates curl gnupg
                sudo install -m 0755 -d /etc/apt/keyrings
                curl -fsSL https://download.docker.com/linux/$OS/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
                sudo chmod a+r /etc/apt/keyrings/docker.gpg
                echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
                sudo apt-get update
                sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            elif [ "$OS" = "centos" ] || [ "$OS" = "rhel" ]; then
                # CentOS/RHEL
                sudo yum install -y yum-utils
                sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
                sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
                sudo systemctl start docker
                sudo systemctl enable docker
            fi
            
            # 添加当前用户到 docker 组
            sudo usermod -aG docker $USER
            
            echo "Docker 安装完成"
ENDSSH
        print_success "Docker 安装完成"
    else
        print_error "请手动安装 Docker 后再运行此脚本"
        exit 1
    fi
fi

# 3. 创建远程目录
print_info "创建远程目录..."
ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "mkdir -p $REMOTE_DIR"
print_success "目录创建完成"

# 4. 同步代码到服务器
print_info "同步代码到服务器..."
rsync -avz --progress \
    -e "ssh -p $SERVER_PORT" \
    --exclude '.git' \
    --exclude 'node_modules' \
    --exclude '.env' \
    --exclude 'data' \
    --exclude '*.log' \
    ./ $SERVER_USER@$SERVER_IP:$REMOTE_DIR/

print_success "代码同步完成"

# 5. 配置环境变量
print_info "配置环境变量..."
if [ ! -f .env ]; then
    print_warning "本地没有 .env 文件，将从 .env.example 创建"
    cp .env.example .env
fi

# 上传 .env 文件
scp -P $SERVER_PORT .env $SERVER_USER@$SERVER_IP:$REMOTE_DIR/.env
print_success "环境变量配置完成"

# 6. 在服务器上构建并启动
print_info "在服务器上构建并启动服务..."
ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "cd $REMOTE_DIR && docker compose up -d --build"

if [ $? -eq 0 ]; then
    print_success "服务启动成功！"
else
    print_error "服务启动失败，请查看日志"
    exit 1
fi

# 7. 等待服务启动
print_info "等待服务启动..."
sleep 5

# 8. 检查服务状态
print_info "检查服务状态..."
ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "cd $REMOTE_DIR && docker compose ps"

# 9. 显示访问信息
echo ""
print_success "🎉 部署完成！"
echo ""
echo "访问信息："
echo "  前端: http://$SERVER_IP:3000"
echo "  后端: http://$SERVER_IP:8080"
echo ""
echo "查看日志："
echo "  docker compose logs -f"
echo ""
echo "停止服务："
echo "  docker compose down"
echo ""
echo "重启服务："
echo "  docker compose restart"
echo ""

# 10. 询问是否配置防火墙
read -p "是否需要配置防火墙开放端口? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "配置防火墙..."
    ssh -p $SERVER_PORT $SERVER_USER@$SERVER_IP "bash -s" << 'ENDSSH'
        # 检测防火墙类型
        if command -v ufw > /dev/null 2>&1; then
            # Ubuntu/Debian (UFW)
            sudo ufw allow 3000/tcp
            sudo ufw allow 8080/tcp
            echo "UFW 防火墙规则已添加"
        elif command -v firewall-cmd > /dev/null 2>&1; then
            # CentOS/RHEL (firewalld)
            sudo firewall-cmd --permanent --add-port=3000/tcp
            sudo firewall-cmd --permanent --add-port=8080/tcp
            sudo firewall-cmd --reload
            echo "Firewalld 防火墙规则已添加"
        else
            echo "未检测到防火墙，跳过配置"
        fi
ENDSSH
    print_success "防火墙配置完成"
fi

print_success "所有步骤完成！"
