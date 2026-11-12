# 斗篷列表模块技术方案

配套产品文档：`docs/v2/prd-cloak.md`。本文只描述怎么改，不在这一步改代码。

## 1. 做法

斗篷是商户下的独立记录，不写入 `sites`。站点继续走现有快照和反向代理。斗篷只处理自己的短链。

网关在选站点之前看路径：`/c/{code}` 交给斗篷判定，返回 302 或一段 HTML。其它路径仍按 Host 找站点。这样托管域名上的页面不会被斗篷截走，斗篷短链也不会进站点归因。

和 V1 的两处差别：

| V1 | 本方案 |
| --- | --- |
| 一个域名上最新一条启用斗篷吃掉该域名的全部路径 | 只认 `/c/{code}`，一条短链对应一条斗篷 |
| 黑名单和白名单都跳到审核链接 | 本期不接名单。以后白名单进真实链接，黑名单进审核侧 |

## 2. 数据

新表 `cloaks`：

| 字段 | 说明 |
| --- | --- |
| `id` | 主键 |
| `merchant_id` | 商户 |
| `domain_id` | 关联域名，创建后不改 |
| `name` | 名称 |
| `status` | `enabled` / `disabled` |
| `real_urls` | JSON 数组，至少一条 |
| `review_mode` | `external` / `local_template` |
| `review_url` | 外链，可空 |
| `review_tag` | `google` / `facebook` / `tiktok` / `generic`，可空 |
| `code` | 短链编码，全表唯一 |
| `code_custom` | 是否商户自填 |
| `allowed_sources` | JSON 数组 |
| `mode` | `cloak` / `green` / `full_open` / `review` |
| `target_regions` | JSON 数组，大写国家代码 |
| `spider_whitelist` | JSON 数组 |
| `device_whitelist` | 空字符串或 `mobile` / `tablet` / `pc` / `mobile,tablet` |
| `block_empty_language` | 布尔 |
| `block_pc` | 布尔 |
| `block_proxy` | 布尔 |
| `intermediate_page` | `none` / `system_template` / `custom_template` |
| `intermediate_text` | 自定义中间页正文，可空 |
| `remark` | 可空 |
| `created_at` / `updated_at` | |

索引：`(merchant_id, status)`、`(domain_id, status)`、`code` 唯一。

新表 `cloak_access_logs`，只记短链访问，不进 `decision_logs`：

| 字段 | 说明 |
| --- | --- |
| `id` | 主键 |
| `merchant_id` | 可空，域名都对不上时为空 |
| `cloak_id` | 可空 |
| `code` | 请求里的编码 |
| `country` | 国家代码，未知为空 |
| `device` | `mobile` / `tablet` / `pc` / `bot` |
| `hit_rule` | 第 4 节的规则名 |
| `final_action` | `real` / `review` / `not_found` |
| `latency_ms` | |
| `created_at` | |

索引：`(cloak_id, created_at)`、`(merchant_id, created_at)`。清理任务按商户 `log_retention_days` 删旧行，和决策日志同一轮。

列表上的四项次数不落在 `cloaks` 上。查询时按 `final_action` 和商户时区的今日零点聚合。单页最多 20 条，用 `cloak_id IN (...)` 一次查出，避免逐行计数。

域名占用：`domains.site_id` 非空的域名不能被斗篷选用。斗篷保存时再查一次。站点绑定域名时，若该域名已有启用或停用的斗篷，拒绝绑定。两边都要查。

额度：沿用商户 `site_quota` 不合适。`merchants` 增加 `cloak_quota`，默认 20。超过返回 429，文案「斗篷数量已达上限」。

## 3. 短链

系统编码：6 位小写字母和数字，冲突重试 5 次。

自定义编码：校验主机名等于关联域名，路径是 `/c/{code}`，忽略大小写后把 code 存成小写。

对外短链：`https://{域名}/c/{code}`。列表和复制都用这个字符串，不另存一份 URL。

## 4. 判定

网关函数 `decideCloak(host, code, ip, ua, acceptLanguage, via, forwardedFor, referer)`。规则名写入 `hit_rule`。

1. 按 `code` 找斗篷。没有，或 Host 不是它的域名：`not_found` / `cloak:missing`。
2. 斗篷、域名、商户任一不可用：`not_found` / `cloak:disabled`。
3. `mode=green` 或 `review`：`review` / `mode:green` 或 `mode:review`。
4. `mode=full_open`：`real` / `mode:full_open`。
5. `mode=cloak` 时按这个顺序，命中即 `review`：
   - `block_pc` 且设备为 `pc`：`rule:block_pc`
   - `block_proxy` 且（有 `Via`，或 `X-Forwarded-For` 含逗号）：`rule:block_proxy`
   - `block_empty_language` 且 `Accept-Language` 为空：`rule:block_empty_language`
   - `target_regions` 非空：有国家且不在列表里则 `rule:region_mismatch`；国家未知则不拦截，`hit_rule` 仍可能是后面的规则；若最终放行，把 `country` 留空即可
   - 来源不是 `all`，且 Referer 主机名不包含所选来源关键字：`rule:source_mismatch`。关键字与 V1 相同：`facebook`、`google`、`tiktok`、`kwai`
   - UA 命中蜘蛛白名单：`rule:spider_whitelist`。匹配子串：`googlebot`、`bingbot`、`facebookexternalhit`、`tiktok`
   - 设备白名单非空且当前设备不在其中：`rule:device_whitelist`
6. 都没拦住：`real` / `rule:pass`。

设备分类：UA 含 `bot`、`spider`、`crawler` 为 `bot`；含 `ipad` 或 `tablet` 为 `tablet`；含 `mobile` 为 `mobile`；其余为 `pc`。bot 不属于移动端、平板或 PC。

国家用网关已有的 GeoIP。文件没加载时国家为空，投放地区不拦截。

`real` 的目标是 `real_urls[0]`。`review` 且 `review_mode=external` 的目标是 `review_url`。`review` 且 `local_template` 没有跳转目标，由网关直接写 HTML。

响应：

| 结果 | HTTP |
| --- | --- |
| `not_found` | 404，正文「Not Found」 |
| `real`，中间页为不使用 | 302 到真实链接 |
| `review` + 外链，中间页为不使用 | 302 到审核链接 |
| `review` + 本地模版 | 200 HTML，按 `review_tag` 选四段固定文案之一 |
| 需要中间页，且结果是跳转 | 200 HTML。页面上一个按钮，链接就是原本要 302 的地址 |

本地模版和中间页的 HTML 放在网关内的 `html/template`，不读商户上传的文件。自定义中间页正文做 HTML 转义。响应加 `X-Robots-Tag: noindex` 和 `Cache-Control: no-store`。

写 `cloak_access_logs` 失败只打日志，不影响响应。

斗篷数据放进现有站点快照不合适，量小且和站点无关。网关启动时和每 5 秒从 MySQL 拉启用斗篷到内存 map，键是 `code`。管理端保存、启停、删除后不要求等这 5 秒以外的通道。停用和删除在接口里同步把该 code 从本进程删掉做不到（网关是另一个进程），所以接受最多 5 秒延迟，并在 PRD 验收里把「立即」理解成这一窗口。接口返回文案仍写「已停用」。

## 5. 管理接口

挂在商户端 `/api/merchant/cloak`。商户范围用登录态，不能用查询参数改商户。

| 方法 | 路径 | 权限 | 作用 |
| --- | --- | --- | --- |
| GET | `/cloak/stats` | `cloak:cloak:list` | 全部、启用、停用、今日跳转次数 |
| GET | `/cloak/list` | `cloak:cloak:list` | 分页。参数 `keyword`、`domainId`、`mode`、`status`。每行带四项次数、短链、域名 |
| GET | `/cloak/:id` | `cloak:cloak:list` | 编辑回填 |
| GET | `/cloak/domain-options` | `cloak:cloak:add` | 可选域名：本商户、已绑定、`site_id` 为空、没有别的斗篷占用 |
| POST | `/cloak` | `cloak:cloak:add` | 新建 |
| PUT | `/cloak/:id` | `cloak:cloak:edit` | 修改。忽略传入的 `domainId` |
| POST | `/cloak/:id/copy` | `cloak:cloak:copy` | 复制，返回新 id |
| PUT | `/cloak/changeStatus` | `cloak:cloak:status` | 启停 |
| DELETE | `/cloak/:id` | `cloak:cloak:remove` | 删除斗篷行，不删访问记录 |

校验失败 400，文案直接说哪个字段。域名被站点占用 409。额度 429。

现有 `GET /merchant/site/list` 继续给定站点管理用。斗篷页不再调用它。

## 6. 菜单和前端

种子把现有 `/merchant/cloak` 从站点列表页改成斗篷列表，组件仍是 `merchant/cloak`，权限改成 `cloak:cloak:list`。按钮权限换成第 5 节的六个权限，角色仍是：查看给全部商户角色，删除只给商户管理员，其余写操作给管理员和操作员。

`/merchant/cloak/form` 不进菜单，和站点配置页一样在 `routes.js` 里注册，`meta.active` 指向 `/merchant/cloak`，避免再次出现「点了地址变了、页面不动」。

页面：

- `views/merchant/cloak/index.vue`：概览、搜索、表格。删掉现在对 `pageSites` 的调用。
- `views/merchant/cloak/form/index.vue`：分组卡片。基本信息、链接、访问条件、高级（蜘蛛、设备、限制、中间页）、备注。底部取消和保存。
- 列表里的复制、启停、删除沿用确认框。

今日次数的时区用商户 `timezone`。非法时区按 `Asia/Shanghai`。

## 7. 测试

- 规则表：四种模式，加上 PC、代理、空语言、地区命中和未命中、来源、蜘蛛、设备白名单。地区库缺失时不拦截。
- 网关：`/c/{code}` 返回 302 或 HTML；同域名的 `/` 仍走站点或 404，不进斗篷。
- 接口：越权 id 返回 404；站点域名不能选；复制后的 code 不同。
- 列表次数：两条 `real`、一条 `review`、一条今日以前的 `review`，四项数字正确。
