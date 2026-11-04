#!/usr/bin/env bash
# 测试服务器一键部署：Ubuntu 22.04 / Debian 12。
# 用法（在仓库根目录，已安装 Go 1.23）：
#   sudo ./deploy/install.sh            # 构建 + 安装 + 启动
#   sudo ./deploy/install.sh --no-build # 只重装配置与 unit
set -euo pipefail

PREFIX=/opt/cloak
USER_NAME=cloak
BUILD=1
[[ "${1:-}" == "--no-build" ]] && BUILD=0

cd "$(dirname "$0")/.."
ROOT=$(pwd)

if ! id -u "$USER_NAME" >/dev/null 2>&1; then
  useradd --system --home "$PREFIX" --shell /usr/sbin/nologin "$USER_NAME"
fi
mkdir -p "$PREFIX"/{bin,configs,env,.local/certs,.local/logs,uploads,migrations}

if [[ $BUILD -eq 1 ]]; then
  echo "== build =="
  (cd "$ROOT/backend-go" && \
    CGO_ENABLED=0 GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o "$PREFIX/bin/admin"   ./cmd/admin && \
    CGO_ENABLED=0 GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o "$PREFIX/bin/gateway" ./cmd/gateway && \
    CGO_ENABLED=0 GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o "$PREFIX/bin/worker"  ./cmd/worker && \
    CGO_ENABLED=0 GOFLAGS=-mod=mod go build -trimpath -ldflags "-s -w" -o "$PREFIX/bin/migrate" ./cmd/migrate)
fi

echo "== configs =="
# 不覆盖已有配置，便于保留本机改动
for f in admin gateway; do
  [[ -f "$PREFIX/configs/$f.yaml" ]] || cp "$ROOT/deploy/configs/$f.yaml" "$PREFIX/configs/$f.yaml"
done
cp "$ROOT/backend-go/configs/worker.yaml" "$PREFIX/configs/worker.yaml" 2>/dev/null || true
rsync -a --delete "$ROOT/backend-go/migrations/" "$PREFIX/migrations/"
for f in admin gateway worker; do
  [[ -f "$PREFIX/env/$f.env" ]] || touch "$PREFIX/env/$f.env"
done
chown -R "$USER_NAME:$USER_NAME" "$PREFIX"
chmod 600 "$PREFIX"/env/*.env

echo "== systemd =="
cp "$ROOT"/deploy/systemd/*.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable cloak-admin cloak-gateway cloak-worker >/dev/null

cat <<EOF

安装完成。接下来：
1. 编辑 $PREFIX/configs/admin.yaml 与 gateway.yaml（或 $PREFIX/env/*.env），填好 MySQL、Redis、JWT、ACME 邮箱、entry_host。
2. 建库并迁移：
     mysql -uroot -p -e "CREATE DATABASE cloak_v2 CHARACTER SET utf8mb4; CREATE USER 'cloak'@'localhost' IDENTIFIED BY '...'; GRANT ALL ON cloak_v2.* TO 'cloak'@'localhost';"
     sudo -u $USER_NAME bash -c "cd $PREFIX && ./bin/migrate up && ./bin/migrate seed"
3. 启动：systemctl start cloak-admin cloak-gateway cloak-worker
4. DNS：entry.<域名> A 记录指到本机；商户托管域名 CNAME 到 entry.<域名>。
5. 验证：curl -I https://<托管域名>/ 应看到证书签发并返回审核内容；带分销码访问返回真实内容。
日志：journalctl -u cloak-gateway -f
EOF
