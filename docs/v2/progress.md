# V2 开发进度

本文件是进度的唯一真源。每个迭代开始与结束时更新。执行方案见 `execution-plan.md`。

## 迭代状态

| 迭代 | 名称 | 状态 | 开始 | 完成 | 标签 |
| --- | --- | --- | --- | --- | --- |
| I0 | 工程基线 | 待验收 | 2026-11-03 | 2026-11-04 | `v2-i0` |
| I1 | 认证与权限 | 未开始 | | | |
| I2 | 商户与额度 | 未开始 | | | |
| I3 | 站点、域名、源站、审核内容 | 未开始 | | | |
| I4 | 归因规则与分销码 | 未开始 | | | |
| I5 | 配置快照与判定引擎 | 未开始 | | | |
| I6 | 托管反向代理与证书 | 未开始 | | | |
| I7 | 决策日志与归因管道 | 未开始 | | | |
| I8 | 服务端判定接口 v2 与 V1 兼容 | 未开始 | | | |
| I9 | 跨浏览器交接 | 未开始 | | | |
| I10 | WordPress/WooCommerce 插件 v2 与 JS SDK | 未开始 | | | |
| I11 | 看板、报表、告警、Webhook | 未开始 | | | |
| I12 | 总后台监管与系统配置 | 未开始 | | | |
| I13 | 迁移工具与影子对比 | 未开始 | | | |
| I14 | 部署与切换 | 未开始 | | | |
| I15 | 可选增强层 | 未开始 | | | |

## 当前迭代：I0 工程基线

- [x] git 仓库初始化与 `.gitignore`
- [x] `backend-go` 骨架：`cmd/{admin,gateway,worker,scheduler,migrate}`、配置加载、zap 日志、请求 ID、若依格式统一响应、恢复与访问日志中间件、Prometheus 指标、优雅停机
- [x] golang-migrate 接入（`migrate up/down/status/create`）
- [x] 配置文件 `configs/*.yaml`，环境变量 `CLOAK_*` 覆盖
- [x] `scripts/dev/*.ps1`：MySQL（3307）、Redis（6380）、四个服务、前端、停止、门禁
- [x] 单测：配置加载、管理 API 路由、网关处理器
- [x] OpenAPI 骨架、golangci 配置
- [x] 本地冒烟：四个服务启动，健康检查、版本接口、404、指标端口均正常
- [x] 前端 `.env.development` 增加 Go 后端地址说明

### I0 验收步骤

1. `powershell -File scripts/dev/start-mysql.ps1`、`start-redis.ps1`：首次运行自动初始化，输出端口与 pid。
2. `powershell -File scripts/dev/check.ps1`：输出 `CHECK PASSED`。
3. `powershell -File scripts/dev/start-service.ps1 -Name admin -Background`（gateway、worker、scheduler 同理）。
4. 浏览器或 curl 访问：
   - `http://127.0.0.1:8080/api/version` 返回 `{"code":200,...}`
   - `http://127.0.0.1:8080/api/nope` 返回 404 且 body 为 `{"code":404,"msg":"接口不存在"}`
   - `http://127.0.0.1:8090/_cloak/health` 返回 `ok`
   - `http://127.0.0.1:9080/metrics` 有 `admin_http_requests_total`
5. `powershell -File scripts/dev/stop-all.ps1` 全部停止。

## 待决策

| 编号 | 事项 | 影响迭代 | 状态 |
| --- | --- | --- | --- |
| D1 | 超级管理员初始账号名（V1 为 `admin`，建议沿用） | I1 | 待回答 |
| D2 | 商户默认额度：判定次数/月、站点数、域名数、分销码数、子账号数、日志保留天数（建议 100 万 / 5 / 20 / 1000 / 10 / 90） | I2 | 待回答 |
| D3 | 功能开关默认值：托管接入、服务端接入、JS SDK、转化回传、Webhook（建议全部开启，增强层关闭） | I2 | 待回答 |

## 已知缺陷与技术债

| 编号 | 描述 | 计划 |
| --- | --- | --- |
| T1 | 便携版 Go/MySQL/Redis 复用自 `D:\pay-virtual\.runtime`，路径写在 `scripts/dev/common.ps1` | 后续换独立安装只改该文件 |
| T2 | 本地无 golangci-lint，门禁中跳过 | 安装后自动启用 |
| T3 | 本地无 ClickHouse，日志仓库先用 MySQL 分区实现 | I7 在测试服务器接 ClickHouse |

## 环境与资源

| 资源 | 状态 | 需要在哪个迭代前 |
| --- | --- | --- |
| 本地 Go 1.23.12 / MySQL 8.4 / Redis 7.4 / Node 22 | 就绪 | - |
| Linux 测试服务器 | 未提供 | I6 |
| 测试域名 ×2 与 DNS 控制权 | 未提供 | I3（仅校验）/ I6（证书与代理） |
| 对象存储（R2/S3） | 未提供 | I6 |
| GeoLite2 账号 | 未提供 | I7 |
| SMTP / Telegram Bot | 未提供 | I11 |
| WordPress + WooCommerce 测试站 | 未提供 | I10 |
| V1 数据副本 | 未提供 | I13 |
