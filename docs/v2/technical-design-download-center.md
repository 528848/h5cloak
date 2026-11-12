# 下载中心技术方案

配套产品文档：`docs/v2/prd-download-center.md`。导出队列、CSV 生成和签名下载保持现状，本方案只把入口和列表收成独立页面。

## 1. 做法

不新建队列表。继续用 `exports`、`opssvc.CreateExport` 和 worker 的 `RunPendingExports`。变化在四处：

| 位置 | 变化 |
| --- | --- |
| 商户菜单 | 增加「下载中心」，去掉报表页里的导出页签 |
| 顶栏 | 下载图标进入 `/merchant/download`，数字来自进行中任务数 |
| 列表接口 | 增加类型、状态筛选，并返回范围摘要和是否过期 |
| 创建接口 | 同一商户进行中任务达到 3 个时返回 429 |

完成通知里的链接从 `/merchant/report?tab=export` 改为 `/merchant/download`。

## 2. 接口

仍挂在商户端，权限不变。

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| POST | `/merchant/export` | 创建任务。进行中达到 3 个时 429，文案「已有 3 个导出任务在处理，请到下载中心查看」 |
| GET | `/merchant/export/list` | 分页。查询参数增加 `kind`、`status` |
| GET | `/merchant/export/stats` | `{ active, ready, expired, failed }`，供概览卡和顶栏数字 |
| POST | `/merchant/export/:id/retry` | 仅失败任务。按原 kind 和 params 新建一条 pending |
| POST | `/merchant/export/:id/cancel` | 仅 pending。状态改为 `cancelled`，worker 不再领取 |
| GET | `/merchant/export/:id/link` | 保持。已过期返回 410 |
| GET | `/api/export/download/:token` | 保持 |

`status` 查询值：`pending`、`running`、`done`、`expired`、`failed`、`cancelled`。`expired` 不是库里的状态，查询时等于 `status=done` 且 `expires_at < now`。`done` 查询只返回未过期的行。列表每行增加：

| 字段 | 说明 |
| --- | --- |
| `expired` | 已完成且过了 `expiresAt` |
| `scopeText` | 由 params 拼出，例如「近 7 日 · 全部站点」。站点名在服务里用站点表补上，不把 ID 交给页面拼 |
| `downloadToken` | 仅未过期的 done 返回，逻辑与现在相同 |

取消用新状态 `cancelled`。`ClaimExport` 只领取 `pending`，因此取消后不会被 worker 拿走。已是 `running` 的取消请求返回 400。

重试不改旧行。新建行复制 `kind`、`params`、`merchant_id`，`user_id` 用当前操作人。

## 3. 统计

`GET /merchant/export/stats` 一次聚合：

- `active`：`pending` + `running`
- `ready`：`done` 且未过期
- `expired`：`done` 且已过期
- `failed`：`failed`

顶栏只使用 `active`。页面打开和每 5 秒轮询时，若 `active=0` 就停止轮询。

## 4. 页面

新页面 `frontend/src/views/merchant/download/index.vue`。菜单种子仍注册这条路由，但 `Visible: false`，侧边栏不展示。入口只有顶栏下载图标。

报表页删除 `export` 页签、导出表格和进入该页签的跳转。`doExport` 成功后只提示，并用 `router` 可点的「去下载中心」。

顶栏在 `header-right.vue` 增加下载图标，商户端显示，总后台不显示。点击 `router.push('/merchant/download')`。`el-badge` 的值来自 `export/stats` 的 `active`，为 0 时隐藏。进入商户布局时拉一次，之后与下载中心页共用 5 秒轮询；离开下载中心后顶栏改为 30 秒一次，避免全站都 5 秒打接口。

页面结构按 `docs/前端UI规范.md` 的列表页：

- `summary-cards` 四张卡，`clickable`，`active-key` 绑定当前状态筛选。
- 搜索卡片：类型下拉、状态下拉、创建时间范围。按钮用 `btn-items` 的搜索和重置。
- 表格卡片：`ele-pro-table`，操作列固定右侧，按钮为 `el-button link`。状态用 `el-tag`。取消和重试用 `ElMessageBox.confirm`，`type: warning`，`draggable: true`。
- 失败原因用现有 `error-text`。任务编号、行数用 `mono`。

范围列直接显示接口返回的 `scopeText`。

## 5. 报表页调整

`frontend/src/views/merchant/report/index.vue` 删除导出页签相关列、数据源和 `tab=export` 的初始值。导出按钮保留。成功文案改为「已加入下载中心」。

通知链接在 `opssvc.runExport` 里改路径。已发出的旧通知仍指向报表页，不回写历史数据。报表页收到 `tab=export` 时重定向到 `/merchant/download`，避免旧通知点开是空白页签。

## 6. 测试

- 第 4 个进行中任务返回 429，已完成和失败的不计入这 3 个。
- `expired` 筛选不返回未过期文件；`done` 筛选不返回过期文件。
- 取消 pending 后 worker 不再生成该文件。取消 running 返回 400。
- 重试产生新 ID，旧失败记录仍在。
- 顶栏数字与 `active` 一致。报表页不再渲染导出表格。

不改 CSV 列、签名算法和 24 小时过期时间。
