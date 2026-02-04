# ZenoGuard Docker 部署 - 快速开始

## 📦 Docker部署说明

本项目已配置完整的Docker部署方案，支持一键部署到生产服务器。

## 🚀 快速部署（3步）

### 1. 准备配置

```bash
# 克隆项目
git clone <repository-url>
cd zenoGuard/server

# 复制环境配置
cp api/.env.example api/.env

# 编辑配置文件（必须配置的项）
nano api/.env
```

**必须配置项：**
```bash
# 应用密钥（生成命令：php artisan key:generate）
APP_KEY=your_generated_key_here

# 数据库密码
DB_PASSWORD=your_secure_password

# LLM API（如果使用）
LLM_API_URL=https://api.openai.com/v1/chat/completions
LLM_API_KEY=sk-xxxxxxxxx
```

### 2. 一键部署

```bash
# 运行部署脚本
./deploy-docker.sh
```

### 3. 访问应用

- **API服务**: http://your-server:8888
- **管理后台**: http://your-server:8888/hosts

## 📁 文件说明

### Docker配置文件
- `Dockerfile` - API服务镜像定义
- `docker-compose.yml` - 生产环境配置
- `docker-compose.dev.yml` - 开发环境配置
- `.dockerignore` - Docker构建忽略文件
- `docker/supervisord.conf` - Laravel定时任务配置

### 部署脚本
- `deploy-docker.sh` - 一键部署脚本
- `backup.sh` - 数据库备份脚本

### 文档
- `DOCKER_DEPLOY.md` - 完整部署文档
- `README_DOCKER.md` - 本文档

## 🔧 常用命令

### 服务管理
```bash
# 启动所有服务
docker-compose up -d

# 停止所有服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f api
```

### 数据库操作
```bash
# 进入MySQL
docker-compose exec mysql mysql -uroot -p

# 运行迁移
docker-compose exec api php artisan migrate

# 数据库备份
./backup.sh
```

### 日志查看
```bash
# API日志
docker-compose logs -f api

# 定时任务日志
docker-compose exec api tail -f /var/log/supervisor/scheduler.out.log

# 所有日志
docker-compose logs
```

## 🌐 部署到新服务器

### 1. 服务器环境要求
- Linux系统（推荐：Ubuntu 20.04+ / CentOS 8+）
- Docker 20.10+
- Docker Compose 2.0+
- 至少2GB内存
- 至少10GB磁盘空间

### 2. 部署步骤

```bash
# SSH登录服务器
ssh user@your-server-ip

# 安装Docker（如果没有）
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
sudo user -aG docker $USER

# 安装Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-uname -m" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 克隆项目
git clone <repository-url>
cd zenoGuard/server

# 运行部署脚本
chmod +x deploy-docker.sh
./deploy-docker.sh
```

### 3. 配置防火墙

```bash
# 开放必要端口
sudo ufw allow 8888/tcp  # API端口
sudo ufw allow 22/tcp      # SSH端口
sudo ufw enable
```

### 4. 配置反向代理（可选）

使用Nginx配置HTTPS访问，参考 `DOCKER_DEPLOY.md`。

## 📊 监控和维护

### 自动备份
设置crontab定时备份数据库：

```bash
# 添加到crontab
crontab -e

# 每天凌晨3点自动备份
0 3 * * * /path/to/zenoGuard/server/backup.sh
```

### 日志管理

日志位置：
- API日志：`docker-compose logs api`
- 定时任务日志：`/var/log/supervisor/scheduler.out.log`

### 健康检查

```bash
# 检查API健康状态
curl http://localhost:8888/api/health

# 检查服务状态
docker-compose ps
```

## 🔒 安全建议

1. ✅ 修改所有默认密码
2. ✅ 配置防火墙限制访问
3. ✅ 使用HTTPS（配置Nginx反向代理）
4. ✅ 定期更新依赖
5. ✅ 定期备份数据库
6. ✅ 监控日志文件

## 📞 技术支持

详细文档请查看：`DOCKER_DEPLOY.md`

## 🐛 故障排查

### 问题1: 端口被占用
```bash
# 检查端口占用
sudo netstat -tunlp | grep 8888

# 停止占用端口的进程
sudo kill <PID>
```

### 问题2: 容器启动失败
```bash
# 查看详细日志
docker-compose logs api

# 重新构建
docker-compose down
docker-compose build
docker-compose up -d
```

### 问题3: Agent无法连接
1. 检查网络连通性
2. 验证Token是否正确
3. 查看Agent日志：`tail -f /var/log/zenoguard/agent.log`

## 🔄 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose down
docker-compose build
docker-compose up -d
```

## 📦 镜像导出（离线部署）

```bash
# 导出镜像
docker save zenoguard-api | gzip > zenoguard-api.tar.gz

# 在目标服务器导入
gunzip < zenoguard-api.tar.gz | docker load
```
