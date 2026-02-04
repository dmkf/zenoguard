#!/bin/bash

# ZenoGuard 数据库备份脚本

set -e

# 配置
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

# 创建备份目录
mkdir -p "$BACKUP_DIR"

echo "开始备份数据库..."
echo "备份目录: $BACKUP_DIR"
echo "时间戳: $TIMESTAMP"
echo ""

# 从.env文件读取数据库配置
if [ -f .env ]; then
    source .env
else
    echo "错误: 未找到.env文件"
    exit 1
fi

# 数据库名称
DB_NAME=${DB_DATABASE:-zenoguard}

# 备份文件名
BACKUP_FILE="$BACKUP_DIR/backup_${TIMESTAMP}.sql"

# 执行备份
echo "备份数据库: $DB_NAME"
docker-compose exec -T mysql mysqld -u${DB_USERNAME:-zenoguard} -p${DB_PASSWORD} $DB_NAME > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "✓ 备份成功: $BACKUP_FILE"

    # 压缩备份文件
    gzip "$BACKUP_FILE"
    echo "✓ 已压缩: ${BACKUP_FILE}.gz"

    # 删除旧备份
    echo ""
    echo "清理 $RETENTION_DAYS 天前的旧备份..."
    find "$BACKUP_DIR" -name "backup_*.sql.gz" -type f -mtime +${RETENTION_DAYS} -delete
    echo "✓ 旧备份已清理"

    echo ""
    echo "备份完成！"
else
    echo "✗ 备份失败"
    exit 1
fi
echo ""
echo "当前备份文件:"
ls -lh "$BACKUP_DIR" | tail -5
