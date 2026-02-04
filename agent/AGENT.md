# ZenoGuard Agent 使用说明

## 快速开始

### 1. 构建 Agent

```bash
cd /Users/xiong/works/zenoGuard/agent

# 构建 Linux AMD64 版本
./build-quick.sh

# 构建 macOS 版本
./build-quick.sh darwin
```

### 2. 在服务器上创建主机

```bash
# 登录获取 token
curl -X POST http://127.0.0.1:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 创建主机（替换 YOUR_HOSTNAME）
curl -X POST http://127.0.0.1:8000/api/hosts \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "YOUR_HOSTNAME",
    "remark": "本地开发机器",
    "report_interval": 60,
    "is_active": true
  }'
```

### 3. 启动 Agent

**方式一：使用启动脚本（推荐）**
```bash
cd /Users/xiong/works/zenoGuard/agent

# 使用默认配置启动
./start-agent.sh

# 或自定义配置
SERVER_URL=http://your-server:8000 \
TOKEN=your-host-token \
./start-agent.sh
```

**方式二：直接运行**
```bash
./bin/zenoguard-agent \
  -server http://127.0.0.1:8000 \
  -token YOUR_HOST_TOKEN \
  -log /tmp/zenoguard-logs/agent.log
```

**方式三：后台运行**
```bash
nohup ./bin/zenoguard-agent \
  -server http://127.0.0.1:8000 \
  -token YOUR_HOST_TOKEN \
  -log /tmp/zenoguard-logs/agent.log \
  > /dev/null 2>&1 &
```

## 当前配置

- **服务器**: http://127.0.0.1:8000
- **主机名**: HIH.local
- **Token**: 42c284fc60718ad80261ccb0fdfed500968ca0d0a53cd8bb94cf3d469be579a5
- **报告间隔**: 60 秒

## 测试 Agent

手动发送测试数据：

```bash
cd /Users/xiong/works/zenoGuard/agent
go run /tmp/test-report.go
```

## 查看 Agent 状态

```bash
# 查看进程
ps aux | grep zenoguard-agent

# 查看日志
tail -f /tmp/zenoguard-logs/agent.log

# 查看服务器数据
curl http://127.0.0.1:8000/api/hosts \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

## 停止 Agent

```bash
# 前台运行: 按 Ctrl+C

# 后台运行:
pkill -f zenoguard-agent
```

## 故障排除

### 1. 权限错误
如果遇到 `/etc/zenoguard/config.json` 权限错误，使用命令行参数：
```bash
./bin/zenoguard-agent -server URL -token TOKEN
```

### 2. 连接失败
- 检查服务器是否运行: `curl http://127.0.0.1:8000/api/hosts`
- 验证 token 是否正确
- 检查防火墙设置

### 3. 数据未上报
- 查看 agent 日志: `tail -f /tmp/zenoguard-logs/agent.log`
- 确认主机已激活: 登录管理后台查看主机状态
- 测试网络连接

## 生产环境部署

### 1. 复制到服务器

```bash
# 复制二进制文件
scp bin/zenoguard-agent-linux-amd64 user@server:/usr/local/bin/zenoguard-agent

# SSH 到服务器
ssh user@server

# 设置权限
chmod +x /usr/local/bin/zenoguard-agent
```

### 2. 创建 systemd 服务

```bash
sudo cat > /etc/systemd/system/zenoguard-agent.service << 'EOF'
[Unit]
Description=ZenoGuard Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/zenoguard-agent -server https://your-server.com -token YOUR_TOKEN -log /var/log/zenoguard/agent.log
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable zenoguard-agent
sudo systemctl start zenoguard-agent
sudo systemctl status zenoguard-agent
```

### 3. 查看日志

```bash
journalctl -u zenoguard-agent -f
```

## 监控数据说明

Agent 会定期采集以下数据：

- **CPU 使用率**: 系统负载和进程 CPU 占用
- **内存使用率**: 内存和交换空间使用情况
- **磁盘使用率**: 各分区磁盘空间使用
- **网络流量**: 接收和发送的字节数
- **系统负载**: 1、5、15 分钟平均负载
- **运行时间**: 系统启动时间
- **SSH 登录**: 当前登录用户
- **公网 IP**: 服务器的外网 IP
- **IP 地理位置**: 根据 IP 解析的地理位置

数据上报间隔可在服务器端配置，默认 60 秒。
