#!/bin/bash

# ZenoGuard Agent 启动脚本
# 用于启动本地开发环境的 agent

set -e

# 配置
SERVER_URL="${SERVER_URL:-http://127.0.0.1:8000}"
TOKEN="${TOKEN:-42c284fc60718ad80261ccb0fdfed500968ca0d0a53cd8bb94cf3d469be579a5}"
LOG_FILE="${LOG_FILE:-/tmp/zenoguard-logs/agent.log}"
AGENT_BIN="${AGENT_BIN:-./bin/zenoguard-agent}"

# 创建日志目录
mkdir -p $(dirname "$LOG_FILE")

echo "=========================================="
echo "ZenoGuard Agent 启动脚本"
echo "=========================================="
echo "服务器: $SERVER_URL"
echo "Token: ${TOKEN:0:20}..."
echo "日志: $LOG_FILE"
echo "=========================================="

# 检查 agent 二进制文件
if [ ! -f "$AGENT_BIN" ]; then
    echo "错误: 找不到 agent 二进制文件: $AGENT_BIN"
    echo "请先运行: ./build-quick.sh"
    exit 1
fi

# 检查是否已经运行
if pgrep -f "zenoguard-agent" > /dev/null; then
    echo "警告: Agent 似乎已经在运行"
    echo "如果需要重启，请先运行: pkill -f zenoguard-agent"
    echo ""
fi

# 启动 agent（前台运行，便于调试）
echo "启动 Agent..."
echo "按 Ctrl+C 停止"
echo ""

exec "$AGENT_BIN" \
    -server "$SERVER_URL" \
    -token "$TOKEN" \
    -log "$LOG_FILE" \
    -log-level info
