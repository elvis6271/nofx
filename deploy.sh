#!/bin/bash

# NOFX 交易系统快速部署脚本
# 适用于 Linux 服务器

set -e  # 遇到错误立即退出

echo "🚀 开始部署 NOFX 交易系统..."
echo "================================"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查是否为首次部署
if [ -d "nofx" ]; then
    echo -e "${YELLOW}⚠️  检测到现有部署${NC}"
    read -p "是否清理并重新部署？(y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}⏹️  停止旧容器...${NC}"
        cd nofx
        docker compose down || true
        cd ..
        
        echo -e "${YELLOW}🗑️  删除旧代码...${NC}"
        rm -rf nofx
    else
        echo -e "${GREEN}✅ 保留现有部署，执行更新流程${NC}"
        cd nofx
        
        echo -e "${YELLOW}⏹️  停止容器...${NC}"
        docker compose down
        
        echo -e "${YELLOW}📥 拉取最新代码...${NC}"
        git pull origin dev
        
        echo -e "${YELLOW}🔨 重新构建并启动...${NC}"
        docker compose up -d --build
        
        echo -e "${GREEN}✨ 更新完成！${NC}"
        echo -e "${GREEN}🌐 访问地址: http://localhost:8080${NC}"
        
        echo -e "${YELLOW}📋 最近日志：${NC}"
        docker logs nofx-trading --tail 20
        
        exit 0
    fi
fi

# 首次部署流程
echo -e "${GREEN}📥 克隆代码...${NC}"
git clone -b dev https://github.com/elvis6271/nofx.git

echo -e "${GREEN}📂 进入项目目录...${NC}"
cd nofx

# 检查 config.json
if [ ! -f "config.json" ]; then
    echo -e "${YELLOW}⚙️  创建配置文件...${NC}"
    cp config.json.example config.json
    
    echo -e "${RED}❗ 重要：请编辑 config.json 配置文件${NC}"
    echo -e "${YELLOW}需要配置的关键项：${NC}"
    echo "  1. jwt_secret - 使用 openssl rand -base64 64 生成"
    echo "  2. ai500_api_url - 信号源地址"
    echo "  3. oi_top_api_url - 信号源地址"
    echo ""
    read -p "是否现在编辑配置文件？(y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        nano config.json
    else
        echo -e "${RED}⚠️  请稍后手动编辑 config.json 并重新运行此脚本${NC}"
        exit 1
    fi
fi

# 验证 config.json 格式
echo -e "${GREEN}✅ 验证配置文件...${NC}"
if ! python3 -m json.tool config.json > /dev/null 2>&1; then
    echo -e "${RED}❌ config.json 格式错误，请检查${NC}"
    exit 1
fi

# 构建并启动
echo -e "${GREEN}🔨 构建并启动容器（这可能需要几分钟）...${NC}"
docker compose up -d --build

# 等待启动
echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
sleep 15

# 检查容器状态
echo -e "${GREEN}✅ 检查服务状态...${NC}"
if docker ps | grep -q nofx-trading; then
    echo -e "${GREEN}✨ 部署成功！${NC}"
    echo -e "${GREEN}🌐 访问地址: http://localhost:8080${NC}"
    
    echo -e "${YELLOW}📋 最近日志：${NC}"
    docker logs nofx-trading --tail 20
    
    echo ""
    echo -e "${GREEN}常用命令：${NC}"
    echo "  查看日志: docker logs nofx-trading -f"
    echo "  重启服务: docker compose restart"
    echo "  停止服务: docker compose down"
    echo "  更新代码: git pull origin dev && docker compose up -d --build"
else
    echo -e "${RED}❌ 部署失败，请查看日志：${NC}"
    docker logs nofx-trading --tail 50
    exit 1
fi
