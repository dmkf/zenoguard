# 智巡Guard 部署指南

## 生产环境部署

### Server端部署

#### 1. 系统准备

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装必要软件
sudo apt install -y php8.1 php8.1-fpm php8.1-mysql php8.1-xml php8.1-mbstring \
  php8.1-curl php8.1-bcmath nginx mysql-server composer git
```

#### 2. 数据库配置

```sql
-- 创建数据库
CREATE DATABASE zenoguard_prod CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户（使用强密码）
CREATE USER 'zenoguard'@'localhost' IDENTIFIED BY 'STRONG_PASSWORD_HERE';

-- 授权
GRANT ALL PRIVILEGES ON zenoguard_prod.* TO 'zenoguard'@'localhost';
FLUSH PRIVILEGES;
```

#### 3. 部署代码

```bash
# 克隆代码
cd /var/www
sudo git clone https://github.com/dmkf/zenoguard.git
cd zenoGuard/server/api

# 安装依赖（生产环境）
sudo composer install --optimize-autoloader --no-dev

# 设置权限
sudo chown -R www-data:www-data /var/www/zenoguard
sudo chmod -R 755 /var/www/zenoguard/server/api
sudo chmod -R 775 /var/www/zenoguard/server/api/storage
```

#### 4. 配置环境变量

```bash
sudo cp .env.example .env
sudo nano .env
```

生产环境配置：
```env
APP_NAME="ZenoGuard"
APP_ENV=production
APP_KEY=base64:生成的密钥
APP_DEBUG=false
APP_URL=https://monitor.example.com

LOG_CHANNEL=daily
LOG_DAYS=14
LOG_LEVEL=error

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=zenoguard_prod
DB_USERNAME=zenoguard
DB_PASSWORD=STRONG_PASSWORD_HERE

SESSION_DRIVER=file
SESSION_LIFETIME=120

# Sanclum配置
SANCTUM_STATEFUL_DOMAINS=monitor.example.com
```

生成应用密钥：
```bash
php artisan key:generate
```

#### 5. 运行迁移和种子

```bash
php artisan migrate --force
php artisan db:seed --force --class=UserSeeder
```

修改默认管理员密码：
```bash
php artisan tinker
>>> User::where('username', 'admin')->update(['password' => Hash::make('NEW_PASSWORD')]);
```

#### 6. 构建前端

```bash
cd ../admin
npm install
npm run build
```

#### 7. SSL证书配置

推荐使用Let's Encrypt：
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d monitor.example.com --email your-email@example.com --agree-tos
```

设置自动续期：
```bash
sudo crontab -e
# 添加
0 0 * * * certbot renew --quiet
```

#### 8. Nginx配置

```bash
sudo cp /var/www/zenoGuard/server/deploy/nginx.conf /etc/nginx/sites-available/zenoguard
sudo nano /etc/nginx/sites-available/zenoguard
```

修改配置：
- `server_name`: 改为实际域名
- SSL证书路径（如果使用自定义证书）

启用配置：
```bash
sudo ln -s /etc/nginx/sites-available/zenoguard /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### 9. PHP-FPM优化

编辑 `/etc/php/8.1/fpm/pool.d/www.conf`：
```ini
pm = dynamic
pm.max_children = 50
pm.start_servers = 5
pm.min_spare_servers = 5
pm.max_spare_servers = 35
pm.max_requests = 500
```

重启PHP-FPM：
```bash
sudo systemctl restart php8.1-fpm
```

#### 10. 设置日志轮转

Laravel已配置日志轮转，无需额外配置。

---

### Agent端部署

#### 1. 编译Agent

在开发机器上：
```bash
cd agent
./build/build.sh      # AMD64
./build/build-arm.sh  # ARM64
```

#### 2. 部署脚本

创建部署脚本 `deploy-agent.sh`：
```bash
#!/bin/bash
set -e

AGENT_BINARY="zenoguard-agent-amd64"
SERVER_URL="https://monitor.example.com"
TOKEN="YOUR_TOKEN_HERE"

# 检查是否以root运行
if [ "$EUID" -ne 0 ]; then
  echo "请使用root运行此脚本"
  exit 1
fi

# 停止现有服务
systemctl stop zenoguard-agent 2>/dev/null || true

# 复制二进制文件
cp $AGENT_BINARY /usr/local/bin/zenoguard-agent
chmod +x /usr/local/bin/zenoguard-agent

# 配置
/usr/local/bin/zenoguard-agent -config -server $SERVER_URL -token $TOKEN

# 创建systemd服务
cat > /etc/systemd/system/zenoguard-agent.service << EOF
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
EOF

# 启用服务
systemctl daemon-reload
systemctl enable zenoguard-agent
systemctl start zenoguard-agent

echo "Agent部署完成"
systemctl status zenoguard-agent
```

#### 3. 批量部署

使用Ansible批量部署：
```yaml
# playbook.yml
---
- hosts: all_servers
  become: yes
  tasks:
    - name: 复制Agent二进制文件
      copy:
        src: zenoguard-agent-amd64
        dest: /usr/local/bin/zenoguard-agent
        mode: '0755'

    - name: 配置Agent
      command: /usr/local/bin/zenoguard-agent -config -server {{ server_url }} -token {{ host_token }}

    - name: 创建systemd服务
      template:
        src: zenoguard-agent.service.j2
        dest: /etc/systemd/system/zenoguard-agent.service

    - name: 启动Agent
      systemd:
        name: zenoguard-agent
        state: started
        enabled: yes
```

---

## 安全加固

### 1. 防火墙配置

```bash
# UFW
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# iptables
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -j DROP
```

### 2. 数据库安全

```bash
# MySQL安全配置
sudo mysql_secure_installation

# 编辑MySQL配置
sudo nano /etc/mysql/mysql.conf.d/mysqld.cnf
# 添加
bind-address = 127.0.0.1
```

### 3. PHP安全

编辑 `/etc/php/8.1/fpm/php.ini`：
```ini
disable_functions = exec,passthru,shell_exec,system,proc_open,popen
expose_php = Off
allow_url_fopen = Off
allow_url_include = Off
```

### 4. 修改默认密码

```bash
# 登录MySQL
mysql -u root -p

# 修改root密码
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'STRONG_PASSWORD';

# 在应用中修改管理员密码
php artisan tinker
>>> User::find(1)->update(['password' => Hash::make('NEW_STRONG_PASSWORD')]);
```

### 5. 定期备份

创建备份脚本 `backup.sh`：
```bash
#!/bin/bash
BACKUP_DIR="/var/backups/zenoguard"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
mysqldump -u zenoguard -p'PASSWORD' zenoguard_prod | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# 备份配置文件
tar -czf $BACKUP_DIR/config_$DATE.tar.gz /var/www/zenoGuard/server/api/.env

# 删除30天前的备份
find $BACKUP_DIR -type f -mtime +30 -delete

echo "Backup completed: $DATE"
```

添加到crontab：
```
0 2 * * * /path/to/backup.sh
```

---

## 监控和维护

### 1. 日志监控

```bash
# 实时查看Laravel日志
tail -f /var/www/zenoGuard/server/api/storage/logs/zenoguard.log

# 查看Nginx访问日志
tail -f /var/log/nginx/zenoguard-access.log

# 查看Agent日志
tail -f /var/log/zenoguard/agent.log
```

### 2. 性能监控

安装监控工具：
```bash
# htop
sudo apt install htop

# nginx status
sudo apt install libnginx-mod-http-status

# PHP-FPM status
# 编辑 /etc/php/8.1/fpm/pool.d/www.conf
# 取消注释 pm.status_path = /status
```

### 3. 定期更新

```bash
# 系统更新
sudo apt update && sudo apt upgrade -y

# 代码更新
cd /var/www/zenoGuard
git pull origin main

# 依赖更新
cd server/api
composer update --no-dev

# 运行迁移
php artisan migrate --force

# 重启服务
sudo systemctl restart php8.1-fpm
sudo systemctl reload nginx
```

### 4. 数据库维护

```sql
-- 优化表
OPTIMIZE TABLE host_data;
OPTIMIZE TABLE hosts;

-- 清理旧数据（保留90天）
DELETE FROM host_data WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 重置自增ID
ALTER TABLE host_data AUTO_INCREMENT = 1;
```

---

## 高可用部署

### 1. 负载均衡

使用Nginx作为反向代理：
```nginx
upstream zenoguard_backend {
    least_conn;
    server server1.example.com:443;
    server server2.example.com:443;
    server server3.example.com:443;
}

server {
    listen 443 ssl;
    server_name monitor.example.com;

    location / {
        proxy_pass https://zenoguard_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 2. 数据库主从复制

主库配置 (`/etc/mysql/mysql.conf.d/mysqld.cnf`)：
```ini
[mysqld]
server-id = 1
log-bin = mysql-bin
binlog-format = ROW
```

从库配置：
```ini
[mysqld]
server-id = 2
relay-log = mysql-relay-bin
read-only = 1
```

### 3. Redis缓存

安装Redis：
```bash
sudo apt install redis-server
sudo systemctl enable redis-server
```

Laravel配置 `.env`：
```env
CACHE_DRIVER=redis
QUEUE_CONNECTION=redis
SESSION_DRIVER=redis

REDIS_HOST=127.0.0.1
REDIS_PASSWORD=null
REDIS_PORT=6379
```

---

## 故障恢复

### 1. 数据库恢复

```bash
# 解压备份
gunzip /var/backups/zenoguard/db_20260130_020000.sql.gz

# 恢复
mysql -u zenoguard -p zenoguard_prod < /var/backups/zenoGuard/db_20260130_020000.sql
```

### 2. 配置恢复

```bash
# 恢复环境配置
tar -xzf /var/backups/zenoguard/config_20260130_020000.tar.gz -C /
```

### 3. 服务重启

```bash
# 重启所有服务
sudo systemctl restart php8.1-fpm
sudo systemctl restart nginx
sudo systemctl restart mysql

# 重启Agent
sudo systemctl restart zenoguard-agent
```
