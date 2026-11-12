# 商户专属 Telegram 机器人技术方案

配套产品文档：`docs/v2/prd-telegram-bot.md`（v1.1）。在现有告警发送上增加：每商户 Bot、长轮询入站问答、每日日报。

## 1. 总览

```
worker 进程
  ├─ botManager：每 30s 刷新 active 机器人；每 Bot 一条 getUpdates 长轮询
  │     update → telegramsvc.handleUpdate(merchantID, update)
  │        ├─ /start|/bind <code> → 绑定会话
  │        ├─ 自然语言 / 命令 → intent → Digest / Alerts → sendMessage
  │        └─ 未绑定 → 回绑定指引
  └─ dailyLoop：每分钟检查到点日报 → 发往启用会话
出站（admin / worker）：
  告警 / 测试 / 日报 → 优先商户 Token；无则平台 Bot + notify_settings.telegram_chat_id
admin 进程：
  配置接口（Token / 绑定码 / 会话 / 日报 / 测试）
```

要点：

- **每商户一个 Bot Token，AES-GCM 加密存库**（`secretbox` + `auth.secret_key`）。
- **入站用 getUpdates 长轮询，不用 webhook。** 不改 Nginx、不要公网回调；worker 能访问 `api.telegram.org` 即可。`last_update_id` 持久化，重启不重放。
- **问答只读**：复用 `opssvc.Digest` / `Alerts`。
- **worker 单实例**跑 botManager（多实例需 Redis leader，否则同一 Bot 会被两处 getUpdates 抢）。

## 2. 数据模型

迁移：`20261110000012_telegram_bots.{up,down}.sql`。`notify_settings` 不动。

```sql
telegram_bots (
  merchant_id PK,
  bot_token_enc, bot_user_id UNIQUE, bot_username,
  status, last_update_id,
  daily_report, daily_report_hour, daily_report_site, last_report_on,
  created_at, updated_at
)
telegram_chats (
  id PK, merchant_id, chat_id, UNIQUE(merchant_id, chat_id),
  chat_type, title, status, created_at
)
```

绑定码：Redis `tg:bind:<code> = merchantID`，TTL 10 分钟，`GETDEL` 一次性。无 Redis 时绑定接口返回明确错误。

## 3. 包与职责

| 位置 | 职责 |
| --- | --- |
| `domain/ops/telegram.go` | 模型与状态常量 |
| `adapter/mysql/ops_telegram_repo.go` | CRUD、offset、日报标记 |
| `pkg/notifier` | `NewTelegramWithToken` / `GetMe` / `GetUpdates` / `Send` |
| `app/telegramsvc` | 配置、绑定、意图、digest 文案、poll + daily |
| `app/opssvc/digest.go` | 只读概况汇总 |
| `app/opssvc/alert.go` | `sendMerchantTelegram` 优先自有 Bot |
| `transport/http/admin/telegram_handler.go` | 商户 API |
| `views/.../telegram-setup` | 可视化配置页 |
| `views/.../telegram` | 教程页 |

## 4. 商户 API

均需登录；权限见下。

| 方法 | 路径 | 权限 | 作用 |
| --- | --- | --- | --- |
| GET | `/merchant/telegram` | view | 配置视图（hasBot、username、尾号、chats、日报） |
| PUT | `/merchant/telegram/bot` | edit | `{token}` → getMe → 查 bot_user_id 占用 → 加密入库 |
| DELETE | `/merchant/telegram/bot` | edit | 删机器人行（会话保留） |
| POST | `/merchant/telegram/bind-code` | edit | 生成码 + deepLink + groupCommand |
| GET | `/merchant/telegram/chats` | view | 会话列表（轮询） |
| POST | `/merchant/telegram/chats/:id/toggle` | edit | 停用/启用 |
| DELETE | `/merchant/telegram/chats/:id` | edit | 解绑 |
| PUT | `/merchant/telegram/daily` | edit | `{enabled,hour,siteId}`；首期 UI 固定 siteId=0 |
| POST | `/merchant/telegram/test` | edit | 向所有启用会话发测试 |

保存 Token 时若 `bot_user_id` 已被其他 `merchant_id` 占用 → 400「该机器人已被其他商户使用」。

## 5. 长轮询入站

`telegramsvc.RunPolling(ctx)`（worker 启动）：

1. 每 30s `ActiveTelegramBots`；新增起 goroutine，移除/Token 变更则 cancel 重启。
2. 循环 `getUpdates(offset=last+1, timeout=50, allowed_updates=["message"])`。
3. 成功推进 `last_update_id`；401 → 置 `disabled` 并停该循环；其它错误指数退避至 30s。
4. `handleUpdate`：绑定指令 → 未绑定指引 → 限流（20/分钟/chat）→ `Answer` → `Send`。panic recover 不影响循环。

`/start <code>`、`/bind <code>`、`/start@bot <code>`、以及深链偶发的 `/start<code>`（无空格）均需识别。

绑定：`GETDEL` Redis → 校验 merchant 与当前 Bot 一致 → upsert chat（上限 **10** 个启用会话，超出提示解绑）→ 成功文案。

## 6. 意图与文案

`intent.go`：归一化 → 命令优先 → 时间词 × 指标词 → 站点名包含匹配。

| 意图 | 数据 |
| --- | --- |
| visits / conversion | `opssvc.Digest(merchant, site, ParseRange(...))` |
| alerts | `opssvc.Alerts(..., status=open)` |
| report | `BuildDigest`（昨日） |
| help / unknown | 固定清单 |

回复 HTML `parse_mode`，转义用户可控字段。

## 7. 日报

每分钟：`daily_report=1` 且当前小时（商户时区）== `daily_report_hour` 且 `last_report_on < 今天` → 拼 digest → 发全部 active chats → `MarkDailyReportSent`。无会话也标记已发，避免空转重试。

## 8. 告警出站

`sendMerchantTelegram`：有 active 自有 Bot → 解密 Token → 发所有启用会话（`handled=true`，即使无会话也不回退平台 Bot，状态 `skipped:no_chat`）。否则回退平台 Bot + Chat ID。`TestChannel("telegram")` 同逻辑。

## 9. 部署

- worker 需出网访问 `api.telegram.org`；**无需**公网 webhook / Nginx 改动。
- `auth.secret_key`（或 `jwt_secret`）必须配置，否则自有 Bot 功能关闭（日志提示）。
- 绑定需要 Redis。
- `notify.console_url` 用于日报里的看板链接。
- 平台 `CLOAK_NOTIFY_TELEGRAM_BOT_TOKEN` 保留作兜底。
- 多 worker 实例时需 leader 选举，避免双 getUpdates。

## 10. 安全

- Token 加密、接口只回显尾号 4 位；日志不打明文。
- 问答严格按绑定 merchant 过滤。
- 绑定码一次性 + TTL；Bot 与码 merchant 必须一致。
- 限流 20/分钟/chat；offset 去重。

## 11. 实施状态

| 步骤 | 状态 |
| --- | --- |
| 迁移 + 模型 + notifier 按 Token | 已完成 |
| 配置 API + 配置页 + 教程 | 已完成 |
| 长轮询 + 绑定 + 规则问答 | 已完成 |
| 日报定时 | 已完成 |
| 告警优先自有 Bot | 已完成 |
| Token 占用冲突 / 会话上限 / start 无空格 | 本轮补齐 |
| 冒烟与测试环境部署 | 本轮 |

## 12. 验收

对齐 PRD §8。单测覆盖：意图解析、`bindArg`、normalize。
