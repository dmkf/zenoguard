#!/bin/bash

# ZenoGuard Docker 快速部署脚本

set -e

echo "=================================="
echo "  ZenoGuard Docker 部署脚本"
echo "=================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: 未找到Docker，请先安装Docker${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}错误: 未找到docker-compose，请先安装docker-compose${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker环境检查通过${NC}"
echo ""

# 进入项目目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/api"

echo "当前目录: $PWD"
echo ""

# 检查.env文件
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        echo -e "${YELLOW}未找到.env文件，将从.env.example创建...${NC}"
        cp .env.example .env
        echo -e "${GREEN}✓ 已创建.env文件${NC}"
        echo ""
        echo -e "${YELLOW}请编辑 .env 文件配置以下关键信息：${NC}"
        echo "  - APP_KEY (必须设置，运行: php artisan key:generate)"
        echo "  - DB_DATABASE=zenoguard"
        echo "  - DB_USERNAME=zenoguard"
        echo "  - DB_PASSWORD=your_secure_password"
        echo ""
        read -p "是否现在编辑 .env 文件？(y/n) " -n 1
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ${EDITOR:-nano} .env
        fi
    else
        echo -e "${RED}错误: 未找到.env和.env.example文件${NC}"
        exit 1
    fi
fi

# 生成APP_KEY如果为空
if grep -q "^APP_KEY=$" .env; then
    echo -e "${YELLOW}APP_KEY未设置，正在生成...${NC}"
    APP_KEY=$(openssl rand -base64 32 | tr -d "/+=" | cut -c1-32)
    sed -i "s/^APP_KEY=.*/APP_KEY=$APP_KEY/" .env
    echo -e "${GREEN}✓ APP_KEY已生成${NC}"
    echo ""
fi

echo -e "${GREEN}=================================="
echo "  开始构建和启动服务"
echo "==================================${NC}"
echo ""

# 停止现有容器
echo "停止现有容器..."
docker-compose down

# 构建并启动
echo "构建Docker镜像..."
docker-compose build

echo "启动服务..."
docker-compose up -d

echo ""
echo -e "${GREEN}=================================="
echo "  部署完成！"
echo "==================================${NC}"
echo ""

# 显示服务状态
echo "服务状态："
docker-compose ps

echo ""
echo -e "${YELLOW}等待服务启动...${NC}"
sleep 10

# 检查服务健康状态
echo ""
echo "服务健康检查："

# 检查API
if curl -sf http://localhost:8888/api/health > /dev/null 2>&1; then
    echo -e "  ✓ ${GREEN}API服务${NC} - http://localhost:8888"
else
    echo -e "  ⏳ ${YELLOW}API服务启动中...${NC} - http://localhost:8888"
fi

# 检查MySQL
if docker-compose exec mysql mysqladmin ping -h localhost --silent; then
    echo -e "  ✓ ${GREEN}MySQL${NC} - localhost:3306"
else
    echo -e "  ⏳ ${YELLOW}MySQL启动中...${NC}"
fi

# 检查Redis
if docker-compose exec redis redis-cli ping > /dev/null 2>&1; then
    echo -e "  ✓ ${GREEN}Redis${NC} - localhost:6379"
else
    echo -e "  ⏳ ${YELLOW}Redis启动中...${NC}"
fi

echo ""
echo -e "${YELLOW}=================================="
echo "  下一步操作"
echo "==================================${NC}"
echo ""
echo "1. 访问API文档: http://localhost:8888"
echo "2. 运行数据库迁移:"
echo "   docker-compose exec api php artisan migrate"
echo ""
echo "3. 查看日志:"
echo "   docker-compose logs -f api"
echo ""
echo "4. 停止服务:"
echo "   docker-compose down"
echo ""
echo "5. 重启服务:"
echo "   docker-compose restart"
echo ""
echo -e "${GREEN}部署成功！${NC}"
