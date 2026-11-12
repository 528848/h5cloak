# 归因码 技术设计

## 1. 文档信息

| 项 | 内容 |
| --- | --- |
| 配套 | `docs/v2/prd-attribution-code.md` |
| 本阶段 | 只统一术语为「归因码」与补文档；不改表结构、类型取值、权限、判定引擎与交接机制 |
| 改动面 | 商户端 `code` 视图、总后台 `admin/code` 视图、菜单种子 `seed.go` 的菜单名 |

## 2. 现状代码映射

| 能力 | 代码位置 | 本方案 |
| --- | --- | --- |
| 归因码实体 | `internal/domain/attribution/model.go` `Code`（表 `promo_codes`），`CodeType` = `distributor` / `campaign` | 不改 |
| 展示状态（有效/未生效/已过期/已停用） | `Code.EffectiveStatus(now)` | 不改 |
| 带码判定 → 真实/审核 | `internal/engine/engine.go` `Decide` → `attributed` / `unattributed`；参数提取校验 `internal/engine/params` | 不改 |
| 访客令牌签发/验签 | `internal/engine/token`，`engine.issue`；Cookie 名来自站点 `Cookie.Name`（默认 `_ck_vid`） | 不改 |
| 交接凭证 `_ck` | `engine.issueHandoff` / `handoffByToken`，参数名 `engine.HandoffParam = "_ck"` | 不改 |
| 服务端/管理接口 | `internal/app/attrsvc`、`internal/transport/http/admin/attribution_handler.go`、`internal/adapter/mysql/attribution_repo.go` | 不改 |
| 商户端归因码界面 | `frontend/src/views/merchant/code/**` | 改文案 |
| 总后台全平台归因码界面 | `frontend/src/views/admin/code/index.vue` | 改文案 |
| 菜单名 | `internal/app/seed/seed.go` `m.code*` / `a.code*` 的 `Name` | 改菜单名 |

## 3. 判定链路（既有，用于文档对齐，不改）

`Decide` 顺序：

1. 站点是否服务中 → 否则按无归因动作返回审核 / block / not_found。
2. 路径规则命中（bypass / always_real / always_review / block）。
3. 校验访客令牌 Cookie（存在且有效 → 后续可判回访）。
4. 提取 URL 参数并校验（含归因码校验 `ValidateCodes`）。
5. 分支：首访命中 > 回访（有令牌） > 交接 token（`_ck`） > 交接匹配 > 无归因。

归因码的「有效期内→真实、过期→审核」落在第 4、5 步：`params.Validate` 命中有效归因码 → `attributed()` 置 `ContentReal` 并签发令牌；归因码停用 / 过期 / 未生效导致不命中 → `unattributed()` 按站点无归因动作置 `ContentReview`（默认）。

交接：首访/回访在内置浏览器内命中时 `issueHandoff` 生成 `_ck`，`decisionData` 把 `_ck` 追加到 `handoffUrl`，网关/SDK 注入引导条；系统浏览器带 `_ck` 访问走 `handoffByToken` 接续同一归因。

## 4. 术语改动清单（仅展示文案）

### 4.1 菜单种子 `internal/app/seed/seed.go`

| Key | 原 Name | 新 Name |
| --- | --- | --- |
| `m.code` | 分销码 | 归因码 |
| `m.code.add` | 分销码登记 | 归因码登记 |
| `m.code.edit` | 分销码编辑 | 归因码编辑 |
| `m.code.status` | 分销码状态 | 归因码状态 |
| `m.code.remove` | 分销码删除 | 归因码删除 |
| `m.code.export` | 分销码导出 | 归因码导出 |
| `a.code` | 全平台分销码 | 全平台归因码 |
| `a.code.status` | 分销码状态 | 归因码状态 |

`Path` / `Perms` / `Component` / `Key` 全部不变。seed 以 Key upsert，重新播种即更新菜单名；已授权关系不受影响。

### 4.2 商户端 `frontend/src/views/merchant/code/**`

- `components/code-edit.vue`：标题「登记/编辑分销码」→「登记/编辑归因码」，注释。
- `components/code-generate.vue`：注释、结果「已生成 N 个码」→「N 个归因码」。
- `components/code-import.vue`：标题「导入分销码」→「导入归因码」，注释。
- `components/code-search.vue`：注释。
- `index.vue`：汇总卡「全部码」→「全部归因码」，确认框「码「x」」→「归因码「x」」、「N 个码」→「N 个归因码」。
- `constants.js`：类型标签保留「分销码 / 活动码」（归因码的类型），取值不变。

### 4.3 总后台 `frontend/src/views/admin/code/index.vue`

- 列「分销码」→「归因码」，搜索占位「分销码 / 名称」→「归因码 / 名称」，确认「强制停用分销码「x」」→「…归因码「x」」。

## 5. 兼容性

- 数据：`promo_codes` 表、`codeType` 值、`attribution_params` / 规则表均不动。
- 接口：请求/响应字段 `codeType`、`code`、统计字段不变。
- 权限：`attribution:code:*`、`platform:code:*` 不变。
- 部署：仅需重新构建前端 + 重新 seed（菜单名更新）。无迁移。

## 6. 测试

1. 重新 seed 后，商户端与总后台菜单显示「归因码」/「全平台归因码」。
2. 归因码列表、登记、批量生成、导入、导出流程功能不变，类型仍为分销码 / 活动码。
3. 引擎相关单测（`internal/engine/...`）保持通过，本方案不改引擎。
4. 带有效归因码访问同一 URL 返回真实、过期返回审核的回归用例不变。

## 7. 可选增强（非本期默认）

若后续需要在交接链接上保留可读归因码用于商户自有埋点：在 `decisionData` 构造 `handoffUrl` 时，除 `_ck` 外附加 `d.Code.Code`（仅作展示/埋点，引擎仍以 `_ck` 为归因权威，不据该参数新建归因）。需单独评审后开启。
