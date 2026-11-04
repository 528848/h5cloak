#!/usr/bin/env bash
# 每日备份：mysqldump 全库 + 配置与 env，gzip 后保留 KEEP_DAYS 天。
# 由 cloak-backup.timer 调用；也可手动执行。恢复见 docs/v2/runbook.md。
set -euo pipefail

BACKUP_DIR=${BACKUP_DIR:-/opt/cloak/backups}
KEEP_DAYS=${KEEP_DAYS:-7}
DB_NAME=${DB_NAME:-cloak_v2}
STAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"
# root 走 auth_socket，不需要密码
mysqldump --single-transaction --quick --routines --triggers --set-gtid-purged=OFF "$DB_NAME" | gzip -6 > "$BACKUP_DIR/$DB_NAME-$STAMP.sql.gz"
tar -czf "$BACKUP_DIR/config-$STAMP.tar.gz" -C /opt/cloak configs env 2>/dev/null || true

find "$BACKUP_DIR" -type f -mtime +"$KEEP_DAYS" -name '*.gz' -delete
echo "backup ok: $BACKUP_DIR/$DB_NAME-$STAMP.sql.gz ($(du -h "$BACKUP_DIR/$DB_NAME-$STAMP.sql.gz" | cut -f1))"
