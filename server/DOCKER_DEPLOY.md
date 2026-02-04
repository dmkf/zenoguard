# ZenoGuard Docker 部署指南

## 项目说明

ZenoGuard是一个服务器监控系统，包含前后端两个部分：
- **后端 API**: Laravel 10 + PHP 8.1 + MySQL 8.0
- **前端 Admin**: Vue 3 + TypeScript + Element Plus
- **Agent**: Go 语言（需单独部署到监控服务器）

## 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 至少 2GB 内存
- 至少 10GB 磁盘空间

## 快速开始

### 1. 准备配置文件

复制环境变量模板并根据实际情况修改：

```bash
cd /path/to/zenoGuard/server
cp api/.env.example api/.env
```

### 2. 编辑配置文件

编辑 `api/.env` 文件，配置以下关键信息：

```bash
# 应用密钥（必须设置！）
# 生成方法：php artisan key:generate
APP_KEY=base64:xxxxxxxxxxxxx

# 数据库配置
DB_DATABASE=zenoguard
DB_USERNAME=zenoguard
DB_PASSWORD=your_secure_password_here

# LLM API 配置（可选）
LLM_API_URL=https://api.openai.com/v1/chat/completions
LLM_API_KEY=sk-xxxxxxxxxxxxx
LLM_MODEL_NAME=gpt-3.5-turbo

# API 访问地址
APP_URL=http://your-server-ip:8888
```

### 3. 启动服务

```bash
cd /path/to/zenoGuard/server

# 构建并启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f api

# 查看服务状态
docker-compose ps
```

### 4. 访问应用

- **API服务**: http://your-server-ip:8888
- **数据库**: localhost:3306
- **Redis**: localhost:6379

### 5. 初始化数据库

首次启动需要运行数据库迁移：

```bash
# 进入API容器
docker-compose exec api bash

# 运行迁移
php artisan migrate

# 生成新的应用密钥（如果.env中没有设置）
php artisan key:generate

# 创建管理员用户（可选）
php artisan tinker --execute "
\$user = new \App\Models\User();
\$user->name = 'Admin';
\$user->email = 'admin@example.com';
\$user->password = Hash::make('your_password');
\$user->save();
"
```

## 配置说明

### 端口配置

在 `docker-compose.yml` 中修改端口映射：

```yaml
services:
  api:
    ports:
      - "8888:8888"  # 修改左侧端口可更改对外端口
```

### 数据库配置

MySQL数据会持久化到Docker volume `mysql_data`，重启容器数据不会丢失。

重置数据库（危险操作！）：
```bash
docker-compose down -v
docker-compose up -d
```

### LLM分析配置

1. 登录管理后台
2. 进入"LLM配置"页面
3. 配置API URL、API Key和模型名称

支持的LLM服务：
- OpenAI (GPT-3.5/GPT-4)
- Azure OpenAI
- 其他兼容OpenAI API的服务

## Agent 部署

Agent需要部署到被监控的服务器上：

```bash
# 1. 构建Linux agent
cd agent
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o zenoguard-agent-linux-amd64 ./cmd/agent

# 2. 上传到监控服务器
scp zenoguard-agent-linux-amd64 user@monitor-server:/tmp/

# 3. SSH登录监控服务器安装
ssh user@monitor-server
sudo cp /tmp/zenoguard-agent-linux-amd64 /usr/local/bin/zenoguard-agent
sudo chmod +x /usr/local/bin/zenoguard-agent

# 4. 配置agent
mkdir -p /etc/zenoguard
sudo nano /etc/zenoguard/config.json
```

配置文件格式：
```json
{
  "server_url": "http://your-api-server:8888",
  "token": "your-host-token-from-web-ui"
}
```

```bash
# 5. 启动agent
sudo /usr/local/bin/zenoguard-agent -daemon
```

## 常用命令

### 查看日志
```bash
# API日志
docker-compose logs -f api

# MySQL日志
docker-compose logs -f mysql

# 所有服务日志
docker-compose logs -f
```

### 进入容器
```bash
# 进入API容器
docker-compose exec api bash

# 进入MySQL
docker-compose exec mysql mysql -uroot -p
```

### 重启服务
```bash
# 重启所有服务
docker-compose restart

# 重启单个服务
docker-compose restart api
```

### 停止服务
```bash
# 停止所有服务
docker-compose down

# 停止并删除所有数据卷（危险！）
docker-compose down -v
```

### 更新代码

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up -d --build

# 查看构建日志
docker-compose logs -f api
```

## 数据库备份

### 备份
```bash
# 导出数据库
docker-compose exec mysql mysqld -uroot -p${DB_PASSWORD} zenoguard > backup_$(date +%Y%m%d).sql

# 导出所有数据库（包括系统数据库）
docker-compose exec mysql mysqld -uroot -p${DB_PASSWORD} --all-databases > full_backup_$(date +%Y%m%d).sql
```

### 恢复
```bash
# 导入数据库
docker-compose exec -T mysql mysql -uroot -p${DB_PASSWORD} zenoguard < backup_20250131.sql
```

## 监控和维护

### 健康检查
```bash
# 检查API健康状态
curl http://localhost:8888/api/health

# 查看服务状态
docker-compose ps
```

### 定时任务

Laravel定时任务（LLM分析）通过Supervisor自动运行，无需额外配置。

查看定时任务日志：
```bash
docker-compose exec api tail -f /var/log/supervisor/scheduler.out.log
```

### 性能优化

1. **调整PHP内存限制**：修改 `php.ini` 配置
2. **调整MySQL配置**：修改 `my.cnf` 配置
3. **启用OPcache**：已默认启用
4. **配置Redis缓存**：已在docker-compose.yml中配置

## 故障排查

### 容器无法启动
```bash
# 查看详细日志
docker-compose logs api

# 检查端口占用
netstat -tunlp | grep 8888

# 检查磁盘空间
df -h
```

### 数据库连接失败
```bash
# 检查MySQL是否就绪
docker-compose ps

# 进入API容器测试连接
docker-compose exec api php artisan tinker --execute "
try {
    \DB::connection()->getPdo();
    echo 'Database connection OK';
} catch (Exception \$e) {
    echo 'Database connection FAILED: ' . \$e->getMessage();
}
"
```

### Agent无法上报数据

1. 检查网络连通性
2. 验证Token是否正确
3. 查看Agent日志：`tail -f /var/log/zenoguard/agent.log`
4. 确认API服务是否可访问

### 定时任务不执行

```bash
# 检查scheduler进程
docker-compose exec api supervisorctl status

# 手动测试定时任务
docker-compose exec api php artisan schedule:run
```

## 安全建议

1. **修改默认密码**：修改数据库密码、管理员密码
2. **限制网络访问**：使用防火墙限制只允许必要端口
3. **使用HTTPS**：配置反向代理（Nginx）启用HTTPS
4. **定期备份**：设置定时备份数据库
5. **更新依赖**：定期运行 `composer update` 并重新构建镜像

## 反向代理配置（Nginx）

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://localhost:8888;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://localhost:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 技术支持

如有问题请查看：
- 项目仓库
- Issue Tracker
- 技术文档
