#!/usr/bin/env bash
# 在 Linux 上（CI 或服务器）生成发布目录：需要 Go 1.23 与 Node 22。
# 用法：./deploy/build.sh [输出目录，默认 ./release]
set -euo pipefail
cd "$(dirname "$0")/.."
OUT=${1:-release}
rm -rf "$OUT"; mkdir -p "$OUT"/{bin,www,migrations}

VERSION=$(git describe --tags --always 2>/dev/null || echo dev)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS="-s -w -X cloak/internal/pkg/version.Version=$VERSION -X cloak/internal/pkg/version.Commit=$COMMIT -X cloak/internal/pkg/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

(cd backend-go && for c in admin gateway worker migrate replay mockorigin; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=mod go build -trimpath -ldflags "$LDFLAGS" -o "../$OUT/bin/$c" "./cmd/$c"
done)
cp backend-go/migrations/*.sql "$OUT/migrations/"

(cd frontend && npm ci --silent && npm run build >/dev/null)
cp -r frontend/dist/admin "$OUT/www/admin"
cp -r frontend/dist/merchant "$OUT/www/merchant"
cp -r deploy "$OUT/deploy"
echo "$VERSION ($COMMIT)" > "$OUT/VERSION"
echo "release ready: $OUT ($VERSION $COMMIT)"
