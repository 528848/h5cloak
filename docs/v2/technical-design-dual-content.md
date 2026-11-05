# 同域双内容与 V1 接入补齐 技术设计

## 1. 文档信息

| 项 | 内容 |
| --- | --- |
| 配套 | `docs/v2/prd-dual-content.md` |
| 本阶段 | 只定方案，不改代码 |
| 改动面 | WordPress 插件 `cloak-attribution`、商户端接入指南。判定接口和网关反代保持现有契约 |

## 2. 现有代码怎么分工

| 路径 | 作用 | 本方案 |
| --- | --- | --- |
| `backend-go/internal/transport/http/openapi/handler.go` `decision` | v1 请求补 `page` / `action`；v2 返回 `content`、`cookie`、`renderHint` | 不改字段含义 |
| `openapi/auth.go` `authV1` | `X-API-Key` 与站点密钥明文比对，前缀 12 位走 `ByPrefix` | 不改。文档写明要用 `sk_` |
| `openapi/handler.go` `renderHint` | 审核且来源是模板时给 `type=html`；模板是 URL 或审核源站时给 `type=url` | 插件「平台模板」消费它。静态包 `type=static` 仍由商户自己的文件或托管接入处理，插件不下载静态包 |
| `wordpress-plugin/cloak-api` | V1。`dispatch_page` 按 `review_mode` / `real_mode` 做 302 或 `readfile` | 保留给已上线站点。新文档不再把它当推荐安装包 |
| `wordpress-plugin/cloak-attribution` | V2。`Cloak_Attr_Gate::handle` 只在审核时输出，放行直接 return | 在这里补齐模式 |
| 网关 `proxy` | 托管域名按 Host 回源或返回模板，地址不变 | 不改。场景一只用现有站点配置 |

## 3. 插件输出器

把现在的 `render_review` 收成一个输出器，审核页和真实页共用。

```text
render(side, settings, decision) -> 结束请求或继续 WordPress
```

`side` 为 `review` 或 `real`。配置键与 V1 对齐，并多两个同域模式：

| 模式值 | 行为 | 对应现有函数 |
| --- | --- | --- |
| `current` | 只允许真实页。函数直接 return | 今天放行时的路径 |
| `api_html` | 只允许审核页。`renderHint.type=html` 则输出 HTML；`type=url` 则服务器拉取后输出；都没有则输出站点名的简单页 | `render_review` 的默认分支 |
| `wp_page` | `render_wp_page`，当前 URL，200 | 已有 |
| `html_file` | 路径可读则 `readfile`，200，`no-store` | 已有 `REVIEW_HTML_FILE` |
| `proxy_url` | 服务器 GET 该 URL，输出正文，200 | 已有 `REVIEW_PROXY_URL` |
| `wp_page_redirect` | `get_permalink` 后 `wp_safe_redirect` 302。已在目标页上则不再跳 | 从 `cloak-api` 的 `redirect_wp_page` 移入 |
| `external_redirect` | https 且主机在白名单才 `wp_redirect` 302 | 从 `redirect_external` 移入 |
| `html_redirect` | 302 到填写的公开 URL。当前 URI 已包含该路径则不再跳 | 从 `redirect_html` 移入 |

判定结果：

| `content` | 输出器 |
| --- | --- |
| `real`、`bypass` | `render('real', ...)` |
| `block` | 403 页，结束。不走真实页模式 |
| `not_found` | 404 页，结束 |
| 其他，含 `review` | `render('review', ...)` |

`current` 配在审核页上时，保存配置拒绝。`api_html` 配在真实页上时同样拒绝。

Cookie 仍在输出之前按接口返回的 `cookie` 写入，跳转和直接输出都写。这样 302 之后的下一跳还能带上访客令牌。

同域输出统一加：

```text
Cache-Control: no-store
X-Robots-Tag: noindex, nofollow
X-Cloak-Content: review 或 real
```

302 只加 `X-Cloak-Content`，不把审核页缓存指示加到跳转目标上。目标页自己的缓存由那个页面负责。

## 4. 配置与迁移

`Cloak_Attr_Settings` 增加真实页字段，缺省值保证旧配置仍是「审核页按原 `review_mode`，真实页 `current`」：

```text
real_mode = current
real_page_id = 0
real_html_path = ''
real_proxy_url = ''
real_redirect_page_id = 0
real_external_url = ''
real_html_url = ''
external_allowed_hosts = ''
```

审核页原有四个值不改名：`api_html`、`wp_page`、`html_file`、`proxy_url`。新增三个跳转值，避免和「当前地址渲染」混用同一个 `wp_page`。

从 `cloak_api_settings` 迁移时补上真实页，不再丢掉 `real_mode`：

迁移按审核页、真实页各自的 `{side}_mode` 逐一映射（`map_v1_side`），保留 V1 的 302 行为：

| V1 `{side}_mode` | V2 模式 | 携带字段 |
| --- | --- | --- |
| `wp_page` | `wp_page_redirect`（V1 本就是 302） | `{side}_page_id` |
| `external_url` | `external_redirect` | `{side}_external_url` |
| `html_redirect` | `html_redirect` | `{side}_html_url` |
| `html_output` | `html_file`（V1 唯一的同域 200） | `{side}_html_path` |
| 空 / 未配置 | 审核页 `api_html`，真实页 `current` | — |

`external_allowed_hosts` 从 V1 同名字段整体拷贝。`ck_` 不迁入密钥框。迁移仅在 V2 尚无配置时执行。

已经迁过、只存了审核页的站点不重复迁移。后台显示一句说明：真实页目前是继续当前页面，要恢复 V1 的跳转可在真实页里改。

密钥仍只认 `sk_`。`ck_` 不迁入密钥框，设置页继续提示到站点配置重置密钥。

## 5. 后台表单

设置 → Cloak 归因，分成两块，审核页在上，真实页在下。

审核页选项顺序：平台模板、站内页面（地址不变）、服务器 HTML 文件、拉取外部地址、站内页面（跳转）、外部链接（跳转）、HTML 地址（跳转）。

真实页选项顺序：继续当前页面、站内页面（地址不变）、服务器 HTML 文件、拉取外部地址、站内页面（跳转）、外部链接（跳转）、HTML 地址（跳转）。

每个跳转选项的说明用同一句：「浏览器地址会变成目标地址」。不跳转的选项说明用：「地址栏保持访客打开的链接」。

外部链接白名单、排除路径、失败策略、仅首页、跳过已登录用户，沿用现有字段。排除路径命中时不调用判定，直接当前页面，与 V1「匹配则不执行斗篷」一致。

## 6. 托管接入

不新增接口。场景一使用已经存在的配置：

1. 站点类型为托管或混合。
2. 域名类型为托管，状态已绑定。
3. 真实源站 URL，审核内容为模板、审核源站或静态包。
4. 网关 `ByHost` 找到站点后按 `content` 回源或返回模板。

服务端接入的自有域名不走这条回源。文档里把两类域名分开写。

## 7. 接入指南页面

改 `frontend/src/views/merchant/api-guide/index.vue` 里现在的「同域双内容」卡片，内容以 `integration-guide-dual-content.md` 为准。

页面结构：

- 一张选择表。
- 五个场景卡片。场景四的跳转步骤默认折叠，标题带「地址会变」。
- 场景二的插件步骤链到现有 WordPress 卡片，不复制两份密钥说明。
- 场景三继续引用已有的「场景一：页面请求判定」代码，只在前面加「输出哪一套」的步骤，避免第二份签名示例。

签名函数、错误码、V1 兼容调用这几块保持不动。

## 8. 测试

插件内用现有 PHP 结构做分支测试有困难时，至少用手动清单覆盖：

- 审核 `html_file`：请求首页，200，正文是文件内容，地址不变。
- 审核 `wp_page`：200，正文是所选页面，地址仍是首页。
- 审核 `wp_page_redirect`：302，Location 为该页面固定链接。
- 真实 `current`：200，正文是首页本身。
- 真实 `external_redirect`：非 https 或不在白名单时不跳。
- `content=review` 且模式为 `api_html`、`renderHint.type=html`：输出接口里的 HTML。
- 排除路径 `/cart`：不请求网关，输出购物车原文。

网关单测和 v1 鉴权单测保持通过，本方案不改它们的断言。
