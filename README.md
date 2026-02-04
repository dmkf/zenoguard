# 智巡Guard 安装指南

## 系统要求

### Server端
- PHP 8.1+
- MySQL 8.0+
- Nginx 1.18+
- Composer 2.0+
- Node.js 16+ (构建管理后台)

### Agent端
- Linux操作系统 (Debian/Ubuntu/CentOS/Amazon Linux)
- AMD64或ARM64架构

---

## Server端安装

### 1. 安装依赖

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install php8.1 php8.1-fpm php8.1-mysql php8.1-xml php8.1-mbstring php8.1-curl nginx mysql-server composer nodejs npm
```

#### CentOS/RHEL
```bash
sudo yum install php php-fpm php-mysqlnd php-xml php-mbstring php-curl nginx mariadb-server composer nodejs npm
```

### 2. 克隆项目
```bash
cd /var/www
git clone https://github.com/dmkf/zenoguard.git
cd zenoGuard/server/api
```

### 3. 安装PHP依赖
```bash
composer install --optimize-autoloader --no-dev
```

### 4. 配置数据库

创建数据库：
```sql
CREATE DATABASE zenoguard CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'zenoguard'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON zenoguard.* TO 'zenoguard'@'localhost';
FLUSH PRIVILEGES;
```

### 5. 配置Laravel
```bash
cp .env.example .env
php artisan key:generate
```

编辑 `.env` 文件：
```env
APP_NAME="ZenoGuard"
APP_ENV=production
APP_KEY=base64:...
APP_DEBUG=false
APP_URL=https://monitor.example.com

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=zenoguard
DB_USERNAME=zenoguard
DB_PASSWORD=your_password

LOG_CHANNEL=daily
LOG_DAYS=7
```

### 6. 运行数据库迁移
```bash
php artisan migrate
```

### 7. 创建默认管理员
```bash
php artisan db:seed --class=UserSeeder
```

默认账号：`admin` / `admin123`

### 8. 构建管理后台
```bash
cd ../admin
npm install
npm run build
```

### 9. 配置SSL证书

使用Let's Encrypt：
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d monitor.example.com
```

或使用自签名证书（测试环境）：
```bash
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/zenoguard.key \
  -out /etc/nginx/ssl/zenoguard.crt
```

### 10. 配置Nginx
```bash
sudo cp deploy/nginx.conf /etc/nginx/sites-available/zenoguard
sudo ln -s /etc/nginx/sites-available/zenoguard /etc/nginx/sites-enabled/
sudo nano /etc/nginx/sites-available/zenoguard  # 修改server_name
sudo nginx -t
sudo systemctl reload nginx
```

### 11. 设置文件权限
```bash
sudo chown -R www-data:www-data /var/www/zenoGuard/server/api
sudo chmod -R 755 /var/www/zenoGuard/server/api
sudo chmod -R 775 /var/www/zenoGuard/server/api/storage
```

### 12. 配置Laravel任务调度（可选）
```bash
crontab -e
```

添加：
```
* * * * * cd /var/www/zenoGuard/server/api && php artisan schedule:run >> /dev/null 2>&1
```

---

## Agent端安装

### 1. 下载Agent二进制文件

从编译产物下载对应架构的二进制文件：
- AMD64: `zenoguard-agent-amd64`
- ARM64: `zenoguard-agent-arm64`

```bash
wget https://example.com/downloads/zenoguard-agent-amd64
chmod +x zenoguard-agent-amd64
sudo mv zenoguard-agent-amd64 /usr/local/bin/zenoguard-agent
```

### 2. 配置Agent

首次运行需要配置服务器地址和Token：

```bash
sudo zenoguard-agent -config -server https://monitor.example.com -token YOUR_TOKEN_HERE
```

Token从管理后台的"主机管理"中获取。

### 3. 以Daemon方式运行

```bash
sudo zenoguard-agent -daemon
```

### 4. 管理Agent

查看状态：
```bash
zenoguard-agent -status
```

停止运行：
```bash
zenoguard-agent -stop
```

查看日志：
```bash
tail -f /var/log/zenoguard/agent.log
```

### 5. 设置开机自启（systemd）

创建服务文件 `/etc/systemd/system/zenoguard-agent.service`：
```ini
[Unit]
Description=ZenoGuard Agent
After=network.target

[Service]
Type=forking
ExecStart=/usr/local/bin/zenoguard-agent -daemon
Restart=on-failure
RestartSec=60

[Install]
WantedBy=multi-user.target
```

启用服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable zenoguard-agent
sudo systemctl start zenoguard-agent
```

---

## 验证安装

### Server端
1. 访问 `https://monitor.example.com`
2. 使用 `admin` / `admin123` 登录
3. 检查仪表盘显示正常

### Agent端
1. 检查进程运行：`ps aux | grep zenoguard`
2. 检查日志：`tail -f /var/log/zenoguard/agent.log`
3. 在管理后台查看主机列表，确认主机在线

---

## 故障排查

### Agent连接失败
1. 检查Token是否正确
2. 检查网络连通性：`curl https://monitor.example.com/api/agent/report`
3. 检查防火墙规则
4. 查看Agent日志：`/var/log/zenoguard/agent.log`

### Server端错误
1. 检查Nginx错误日志：`/var/log/nginx/zenoguard-error.log`
2. 检查Laravel日志：`storage/logs/zenoguard.log`
3. 检查数据库连接
4. 检查PHP-FPM状态：`systemctl status php8.1-fpm`

### LLM分析失败
1. 检查API密钥是否正确
2. 检查API地址是否可访问
3. 在管理后台测试LLM连接
4. 查看Laravel日志中的错误信息
