#!/bin/sh
# MySQL 定时备份脚本（在 alpine 容器内运行）。
# 依赖 mariadb-client（apk add mariadb-client）；每 BACKUP_INTERVAL dump 一次，保留 KEEP_DAYS 天。
# 用法：由 db-backup 服务 entrypoint 调用。手动测试：
#   docker run --rm -v "$PWD/deploy/scripts/backup.sh:/backup.sh" alpine:3.20 /bin/sh /backup.sh
set -eu

INTERVAL="${BACKUP_INTERVAL:-24h}"
KEEP="${KEEP_DAYS:-7}"
DB="${MYSQL_DATABASE:-gva}"
HOST="${MYSQL_HOST:-mysql}"
PWD_="${MYSQL_PASSWORD:-root}"

# 把 1h30m / 24h 转成 sleep 可识别的秒数（仅支持 h/m/s 简单形式）
to_seconds() {
    s=0
    val="$1"
    while [ -n "$val" ]; do
        num=$(echo "$val" | sed -E 's/^([0-9]+).*/\1/')
        unit=$(echo "$val" | sed -E 's/^[0-9]+([hms]).*/\1/')
        case "$unit" in
            h) s=$((s + num * 3600)) ;;
            m) s=$((s + num * 60)) ;;
            s) s=$((s + num)) ;;
            *) s=$((s + num)) ;;
        esac
        val=$(echo "$val" | sed -E 's/^[0-9]+[hms]//')
    done
    echo "$s"
}

# 安装 mariadb-client（alpine 镜像默认无）
apk add --no-cache mariadb-client >/dev/null

echo "[backup] 启动：每 ${INTERVAL} 备份 ${DB}，保留 ${KEEP} 天"

while true; do
    ts=$(date +%Y%m%d-%H%M%S)
    out="/backups/${DB}-${ts}.sql.gz"
    echo "[backup] dump → ${out}"
    mysqldump -h"$HOST" -uroot -p"$PWD_" --single-transaction "$DB" | gzip > "$out"
    # 清理过期备份
    find /backups -name "${DB}-*.sql.gz" -mtime +${KEEP} -delete 2>/dev/null || true
    sleep "$(to_seconds "$INTERVAL")"
done
