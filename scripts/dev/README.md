# 本地开发脚本

全部为 PowerShell，在仓库任意位置执行。路径与端口集中在 `common.ps1`。

| 脚本 | 作用 |
| --- | --- |
| `start-mysql.ps1` | 启动专用 MySQL（127.0.0.1:3307，库 `cloak_v2`，账号 `cloak/cloak`），首次自动初始化 |
| `start-redis.ps1` | 启动专用 Redis（127.0.0.1:6380） |
| `start-service.ps1 -Name admin` | 前台启动 Go 服务；加 `-Background` 后台运行并写日志到 `.local/logs` |
| `start-frontend.ps1 -Portal admin` | 启动前端（admin 5173 / merchant 5174） |
| `stop-all.ps1` | 停止以上全部进程 |
| `check.ps1` | vet + test + gofmt（+ golangci-lint 如已安装）；加 `-Frontend` 同时跑前端 lint 与构建 |

数据库迁移在 `backend-go` 目录下执行 `go run ./cmd/migrate up`。

端口一览：

| 服务 | 端口 | 指标端口 |
| --- | --- | --- |
| admin | 8080 | 9080 |
| gateway | 8090 | 9090 |
| worker | - | 9091 |
| scheduler | - | 9092 |
| MySQL | 3307 | - |
| Redis | 6380 | - |
| 前端 admin / merchant | 5173 / 5174 | - |
