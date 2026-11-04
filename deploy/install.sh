#!/usr/bin/env bash
# 斗篷系统 V2 服务器安装 / 升级脚本（Debian 12/13、Ubuntu 22.04/24.04，amd64）。
#
# 在一个「发布目录」里执行（由 deploy/release.ps1 打包上传，或在服务器上用 deploy/build.sh 生成）：
#   release/
#     bin/        admin gateway worker migrate replay mockorigin
#     www/        admin/ merchant/        （前端构建产物）
#     migrations/ *.sql
#     deploy/     本目录（configs systemd nginx scripts）
#
# 首次安装：
#   sudo ./deploy/install.sh --admin-host admin.example.com --merchant-host merchant.example.com \
#        --entry-host entry.example.com --acme-email ops@example.com [--demo-origins]
# 之后升级只需：
#   sudo ./deploy/install.sh
# 主机名等参数会记在 /opt/cloak/env/install.conf，升级时自动沿用。
set -euo pipefail

PREFIX=/opt/cloak
USER_NAME=cloak
HERE=$(cd "$(dirname "$0")" && pwd)
RELEASE=$(cd "$HERE/.." && pwd)
CONF_FILE=$PREFIX/env/install.conf

ADMIN_HOST=""; MERCHANT_HOST=""; ENTRY_HOST=""; ACME_EMAIL=""; DEMO_ORIGINS=0; SKIP_PKGS=0; TLS_STAGING=""
[[ -f $CONF_FILE ]] && . "$CONF_FILE"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --admin-host) ADMIN_HOST=$2; shift 2;;
    --merchant-host) MERCHANT_HOST=$2; shift 2;;
    --entry-host) ENTRY_HOST=$2; shift 2;;
    --acme-email) ACME_EMAIL=$2; shift 2;;
    --tls-staging) TLS_STAGING=$2; shift 2;;   # true / false
    --demo-origins) DEMO_ORIGINS=1; shift;;
    --skip-packages) SKIP_PKGS=1; shift;;
    -h|--help) sed -n '2,20p' "$0"; exit 0;;
    *) echo "未知参数: $1"; exit 2;;
  esac
done
[[ $(id -u) -eq 0 ]] || { echo "请用 root 执行"; exit 1; }
for v in ADMIN_HOST MERCHANT_HOST ENTRY_HOST; do
  [[ -n "${!v}" ]] || { echo "缺少 --$(echo $v | tr 'A-Z_' 'a-z-')（首次安装必填）"; exit 2; }
done
ACME_EMAIL=${ACME_EMAIL:-ops@$ENTRY_HOST}
TLS_STAGING=${TLS_STAGING:-true}
for d in bin www/admin www/merchant migrations; do
  [[ -e "$RELEASE/$d" ]] || { echo "发布目录缺少 $d：$RELEASE"; exit 1; }
done

log() { echo -e "\n== $* =="; }

# ---------- 1. 系统软件 ----------
if [[ $SKIP_PKGS -eq 0 ]]; then
  log "系统软件"
  export DEBIAN_FRONTEND=noninteractive
  if ! command -v mysqld >/dev/null 2>&1; then
    # MySQL 8.4 LTS 官方源（Debian 自带的是 MariaDB）。2023 版签名密钥已于 2025-10 过期，用续期后的 2025 版
    command -v gpg >/dev/null 2>&1 || { apt-get update -qq; apt-get install -y -qq gnupg curl ca-certificates >/dev/null; }
    . /etc/os-release
    codename=${VERSION_CODENAME:-bookworm}
    curl -fsSL https://repo.mysql.com/RPM-GPG-KEY-mysql-2025 | gpg --batch --yes --dearmor -o /usr/share/keyrings/mysql.gpg
    echo "deb [signed-by=/usr/share/keyrings/mysql.gpg] https://repo.mysql.com/apt/${ID}/ ${codename} mysql-8.4-lts" > /etc/apt/sources.list.d/mysql.list
  fi
  apt-get update -qq
  apt-get install -y -qq curl gnupg ca-certificates rsync nginx redis-server >/dev/null
  if ! command -v mysqld >/dev/null 2>&1; then
    echo "mysql-community-server mysql-community-server/root-pass password" | debconf-set-selections
    echo "mysql-community-server mysql-community-server/re-root-pass password" | debconf-set-selections
    echo "mysql-community-server mysql-server/default-auth-override select Use Strong Password Encryption (RECOMMENDED)" | debconf-set-selections
    apt-get install -y -qq mysql-community-server >/dev/null
  fi
  systemctl enable --now mysql redis-server nginx >/dev/null 2>&1 || true
  timedatectl set-timezone Asia/Shanghai 2>/dev/null || true
fi

# ---------- 2. 用户与目录 ----------
log "用户与目录"
id -u $USER_NAME >/dev/null 2>&1 || useradd --system --home $PREFIX --shell /usr/sbin/nologin $USER_NAME
mkdir -p $PREFIX/{bin,configs,env,migrations,www,backups,data/{uploads,exports,certs,wal}}
chmod 750 $PREFIX/env

# ---------- 3. 程序、前端、迁移文件 ----------
log "安装程序文件"
install -m 755 "$RELEASE"/bin/* $PREFIX/bin/
install -m 755 "$HERE/scripts/cloak-backup.sh" $PREFIX/bin/cloak-backup.sh
rsync -a --delete "$RELEASE/migrations/" $PREFIX/migrations/
rsync -a --delete "$RELEASE/www/admin/" $PREFIX/www/admin/
rsync -a --delete "$RELEASE/www/merchant/" $PREFIX/www/merchant/
for f in admin gateway worker; do
  [[ -f $PREFIX/configs/$f.yaml ]] || cp "$HERE/configs/$f.yaml" $PREFIX/configs/$f.yaml
done

# ---------- 4. 环境变量（密钥只在首次生成） ----------
log "环境变量"
# 有限输入，避免 tr 被 head 关管道后触发 SIGPIPE（set -o pipefail 下会让脚本退出）
rand() { head -c 96 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | cut -c1-"$1"; }
if [[ ! -f $PREFIX/env/common.env ]]; then
  DB_PASS=$(rand 24)
  # 值加双引号：systemd EnvironmentFile 与 bash source 都能读，DSN 里的 & 不会被当作后台符
  cat > $PREFIX/env/common.env <<EOF
# 自动生成，admin / gateway / worker 共用。修改后 systemctl restart cloak-admin cloak-gateway cloak-worker
CLOAK_MYSQL_DSN="cloak:${DB_PASS}@tcp(127.0.0.1:3306)/cloak_v2?charset=utf8mb4&parseTime=true&loc=Local"
CLOAK_REDIS_ADDR="127.0.0.1:6379"
CLOAK_AUTH_JWT_SECRET="$(rand 48)"
CLOAK_AUTH_SECRET_KEY="$(rand 48)"
CLOAK_DB_PASSWORD="${DB_PASS}"
EOF
fi
cat > $PREFIX/env/admin.env <<EOF
CLOAK_APP_PUBLIC_URL="http://${ADMIN_HOST}"
CLOAK_NOTIFY_CONSOLE_URL="http://${MERCHANT_HOST}"
CLOAK_GATEWAY_ENTRY_HOST="${ENTRY_HOST}"
EOF
cat > $PREFIX/env/gateway.env <<EOF
CLOAK_GATEWAY_ENTRY_HOST="${ENTRY_HOST}"
CLOAK_GATEWAY_TLS_EMAIL="${ACME_EMAIL}"
CLOAK_GATEWAY_TLS_STAGING="${TLS_STAGING}"
EOF
cat > $PREFIX/env/worker.env <<EOF
CLOAK_NOTIFY_CONSOLE_URL="http://${MERCHANT_HOST}"
EOF
cat > $CONF_FILE <<EOF
ADMIN_HOST=$ADMIN_HOST
MERCHANT_HOST=$MERCHANT_HOST
ENTRY_HOST=$ENTRY_HOST
ACME_EMAIL=$ACME_EMAIL
TLS_STAGING=$TLS_STAGING
DEMO_ORIGINS=$DEMO_ORIGINS
EOF
chmod 640 $PREFIX/env/*.env $CONF_FILE
chown -R $USER_NAME:$USER_NAME $PREFIX

# ---------- 5. 数据库 ----------
log "数据库"
set -a; . $PREFIX/env/common.env; set +a
mysql -uroot <<SQL
CREATE DATABASE IF NOT EXISTS cloak_v2 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'cloak'@'localhost' IDENTIFIED BY '${CLOAK_DB_PASSWORD}';
ALTER USER 'cloak'@'localhost' IDENTIFIED BY '${CLOAK_DB_PASSWORD}';
GRANT ALL ON cloak_v2.* TO 'cloak'@'localhost';
FLUSH PRIVILEGES;
SQL
sudo -u $USER_NAME bash -c "set -a; . $PREFIX/env/common.env; . $PREFIX/env/admin.env; set +a; export CLOAK_MIGRATIONS_DIR=$PREFIX/migrations; cd $PREFIX && ./bin/migrate up && ./bin/migrate seed"

# ---------- 6. Nginx ----------
log "Nginx"
sed -e "s/ADMIN_HOST/${ADMIN_HOST}/g" -e "s/MERCHANT_HOST/${MERCHANT_HOST}/g" "$HERE/nginx/cloak.conf" > /etc/nginx/sites-available/cloak
install -m 644 "$HERE/nginx/cloak-console.conf" /etc/nginx/snippets/cloak-console.conf
ln -sf /etc/nginx/sites-available/cloak /etc/nginx/sites-enabled/cloak
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

# ---------- 7. systemd ----------
log "systemd"
cp "$HERE"/systemd/*.service "$HERE"/systemd/*.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable cloak-admin cloak-gateway cloak-worker cloak-backup.timer >/dev/null
systemctl restart cloak-admin cloak-gateway cloak-worker
systemctl start cloak-backup.timer
if [[ $DEMO_ORIGINS -eq 1 ]]; then
  systemctl enable --now cloak-mockorigin@real cloak-mockorigin@review >/dev/null
  systemctl restart cloak-mockorigin@real cloak-mockorigin@review
fi

# ---------- 8. 健康检查 ----------
log "健康检查"
ok=1
for i in $(seq 1 30); do
  a=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/health || true)
  g=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8090/_cloak/health || true)
  w=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9091/metrics || true)
  if [[ $a == 200 && $g == 200 && $w == 200 ]]; then ok=0; break; fi
  sleep 1
done
echo "admin=$a gateway=$g worker=$w"
[[ $ok -eq 0 ]] || { echo "服务未就绪，查看：journalctl -u cloak-admin -u cloak-gateway -u cloak-worker -n 50"; exit 1; }

cat <<EOF

安装完成。
  总后台    http://${ADMIN_HOST}      （admin / admin123，首次登录后请改密码）
  商户端    http://${MERCHANT_HOST}   （merchant / merchant123）
  托管入口  ${ENTRY_HOST}             （商户托管域名 CNAME 到它；本机 80 由 Nginx 转给网关，443 由网关直接监听）
  配置      $PREFIX/configs/*.yaml，密钥与地址在 $PREFIX/env/*.env
  日志      journalctl -u cloak-gateway -f
  备份      每日 03:30 → $PREFIX/backups（cloak-backup.timer）
EOF
