# V2 开发进度

本文件是进度的唯一真源。每个迭代开始与结束时更新。执行方案见 `execution-plan.md`。

## 迭代状态

| 迭代 | 名称 | 状态 | 开始 | 完成 | 标签 |
| --- | --- | --- | --- | --- | --- |
| I0 | 工程基线 | 已完成 | 2026-11-03 | 2026-11-04 | `v2-i0`（90963b9） |
| I1 | 认证与权限 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i1`（654b3ce） |
| I2 | 商户与额度 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i2`（b2dee42） |
| I3 | 站点、域名、源站、审核内容 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i3`（4ef7ec8） |
| I4 | 归因规则与分销码 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i4`（1278261） |
| I5 | 配置快照与判定引擎 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i5`（960cd10） |
| I6 | 托管反向代理与证书 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i6`（af4145d） |
| I7 | 决策日志与归因管道 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i7`（3706736） |
| I8 | 服务端判定接口 v2 与 V1 兼容 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i8`（66ed160） |
| I9 | 跨浏览器交接 | 已完成 | 2026-11-04 | 2026-11-04 | `v2-i9`（0d3d8e0） |
| I10 | WordPress/WooCommerce 插件 v2 与 JS SDK | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i10`（87628e6） |
| I11 | 看板、报表、告警、Webhook | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i11`（2c52ce0） |
| I12 | 总后台监管与系统配置 | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i12`（7d79f64） |
| I13 | 回放回归工具与 V1 清理（原「迁移工具与影子对比」，V1 未上线后收窄，见 D4） | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i13`（234b1af） |
| I14 | 部署与切换 | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i14`（f50668b） |
| I15 | 可选增强层 | 未开始 | | | |

## 当前迭代：I14 测试环境部署

V1 未上线（D4），本迭代按全新部署做，不做灰度切流。生产服务器和正式域名还没有，先在测试机跑通并写成手册。

- [x] `deploy/release.ps1`：Windows 上交叉编译、构建两个前端、打包上传、远程执行安装；`deploy/install.sh` 幂等，升级不覆盖已生成的密钥
- [x] 安装内容：MySQL 8.4 LTS、Redis、Nginx、cloak-admin / gateway / worker、每日备份 timer；可选演示源站
- [x] Nginx 按 Host 分流：两个控制台走静态文件加 `/api`，其它 Host 转网关；443 由网关自己监听并按绑定域名签发证书
- [x] 域名校验：CNAME 指向入口，或 A 记录直接指向入口 IP，都算绑定成功
- [x] 手册：`docs/v2/deploy-guide.md`、`docs/v2/runbook.md`（含 WordPress 整页缓存排除）
- [x] 测试机 `170.178.174.31` 已安装。公网验证：`http://demo.170-178-174-31.sslip.io/` 无参数为审核页，`?ref=DEMO1` 为真实页，带 Cookie 访问 `/products` 仍为真实页，不带 Cookie 的内页为审核页。Let's Encrypt staging 证书已签出（浏览器不信任，切正式证书见手册第 4 节）
- [ ] 正式证书、正式域名、生产机观察期、压测达到 PRD 第 11 章。有生产机后再做

### I14 验收步骤

1. 打开 http://admin.170-178-174-31.sslip.io 用 `admin` / `admin123` 登录，能看到平台首页和网关节点。
2. 打开 http://merchant.170-178-174-31.sslip.io 用 `merchant` / `merchant123` 登录，站点列表里有「演示站点」。
3. 浏览器访问 http://demo.170-178-174-31.sslip.io/ 应是审核页；加上 `?ref=DEMO1` 应是真实页；同一浏览器再打开 `/products` 仍是真实页。
4. 服务器上 `systemctl is-active cloak-admin cloak-gateway cloak-worker cloak-backup.timer` 都是 `active`。
5. `docs/v2/deploy-guide.md` 与 `docs/v2/runbook.md` 能独立完成一台新机器的安装和一次备份恢复。

## 上一迭代记录：I13 回放回归工具与 V1 清理

V1 没有上线、没有需要保留的数据（D4），原计划的「V1 → V2 数据迁移」「与 V1 影子对比」「迁移手册」没有对象，不再做。保留并落地的是对后续仍有用的两件事：

- [x] V1 清理：删除 Laravel 后端 `backend/`、本地 `_laravel_skeleton/` 与 `tools/composer.phar`；删除商户端「斗篷列表 / 访问日志 / 黑白名单 / API Key」和总后台「全部斗篷 / 斗篷日志」六个没有后端接口的 V1 页面及其 API 封装；`migrate seed` 新增 `retireMenus`，把这些菜单连同按钮与角色绑定一起从库里删掉（幂等）
- [x] 回放回归工具 `cmd/replay`：`export` 从 `decision_logs` 导出用例（JSONL），`run` 回放到网关并输出差异报告（.md / .json）。日志里没有参数值与访客令牌，导出时按归因结果重建参数并标注 `reconstructed`，原请求被全局开关 / 额度 / 站点停用改写过的记在 `overrides`
- [x] 差异分类：`match`、`needs_token`（回访 / 交接依赖令牌）、`global_override`、`quota_override`、`site_not_serving`、`param_reconstructed`、`host_unbound`、`transport_error`、`unexplained`；前七类为可解释差异，`unexplained` > 0 时退出码 1，可接 CI
- [x] 干跑：回放请求带 `X-Cloak-Replay` 头（只在 `X-Cloak-Debug: decide` 下生效），网关不计额度、不写决策日志、不计入节点统计；`-record` 恢复正常记录。`-loop` / `-rate` / `-concurrency` 可作压测发生器（I14 压测复用）
- [x] 单测：参数重建、差异分类、假网关端到端、JSONL 读写；本地回放演示站点 60 条 ×5 轮全部 `match`，唯一差异是紧急审核期间的请求，归类 `global_override` 并说明「现已解除」
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I13 验收步骤

1. 商户端左侧不再有「斗篷列表 / 访问日志 / 黑白名单 / API Key」，总后台不再有「全部斗篷 / 斗篷日志」；仓库里没有 `backend/` 目录。
2. `cd backend-go && go run ./cmd/replay export -out corpus.jsonl -site <有流量的站点> -limit 1000`，输出「exported N cases」。
3. `go run ./cmd/replay run -in corpus.jsonl -report replay-report.md`，报告里绝大多数为 `match`；打开报告能看到每类差异的样例、原因和判定轨迹。期间商户看板的访问数不变（干跑不记日志）。
4. 总后台打开「紧急全平台审核」再跑一次，差异全部归为 `global_override`，退出码 0；关掉后恢复。
5. 改一个站点的某个路径规则（如把 `/lp` 设为始终审核）再跑，出现 `unexplained` 且退出码 1，报告的轨迹列显示 `path_rule=...`。改回后再跑恢复。
6. `go run ./cmd/replay run -in corpus.jsonl -loop 50 -concurrency 16 -report load.json`，报告含吞吐与 P50 / P95 / P99。
7. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 上一迭代记录：I12 总后台监管与系统配置

- [x] 迁移 `20261104000008`：`sites.platform_hold`，`system_settings` 单行配置（紧急审核、四个接入开关、日志保留上限、限流、Webhook 重试、广告 Referer、内置浏览器特征、默认路径规则）
- [x] 全局开关走 Redis `cfg:global`：admin 保存后立即发布并推 `cfg:changed` 的 `global` 消息；网关订阅即时重载，另每 4 秒轮询兜底。紧急审核与关闭的接入只把「放行」改为「审核」并清访客令牌，bypass / 拦截 / 维护页不动，trace 记 `global => emergency_review | entry_off`
- [x] 网关进程内热更新：广告 Referer 表、内置浏览器特征、开放接口两级限流；admin 进程热更新新建站点的默认路径规则（已有站点保留自己的副本）；worker 读取日志保留上限与 Webhook 重试次数
- [x] 强制停用：总后台停用站点时置 `platform_hold`，商户再启用返回 403「该站点已被平台强制停用，请联系平台恢复」，总后台恢复后清除；商户端与总后台站点列表显示「平台停用」标签
- [x] 模拟登录：总后台「商户账号 → 更多 → 模拟登录」签发 60 秒一次性票据并打开商户端 `/login?ticket=`；商户端登录页兑换票据，本地已有会话会被票据覆盖。操作日志记 `POST /api/admin/merchant/:id/impersonate`，登录日志记「模拟登录」
- [x] 总后台新页面：系统配置（含网关节点卡）、默认模板增删改、全平台分销码查询与强制停用、管理员、角色（菜单勾选）、菜单（树形，可改名称/图标/排序/显示）
- [x] 权限：运营管理员可看系统配置、模板、分销码，可模拟登录；修改系统配置、模板、管理员、角色、菜单只有超级管理员。保护：不能删自己、不能停用或删最后一个启用的超级管理员、不能删内置角色、不能删仍被站点使用的模板
- [x] 商户端与总后台「API 调用日志」改读 `api_logs`（请求 ID、商户、站点、接口、状态码、错误码、耗时、IP）
- [x] 单测：`globalcfg.Apply` / `Overlay`；冒烟：紧急审核 0.22 秒生效、JS 入口同样返回审核、关闭后恢复放行；强制停用后判定为审核、商户启用 403、恢复后放行；模拟登录票据只能兑换一次
- [x] 浏览器：系统配置页、模板、分销码停用与恢复、管理员、角色、菜单树、API 调用日志、模拟登录直达商户首页、商户启用被平台停用的站点收到 403 提示
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I12 验收步骤

1. 总后台「系统配置」打开「紧急全平台审核」并保存。5 秒内用带归因参数的链接访问任一站点，应看到审核内容；网关日志出现 `global config applied {"emergency": true}`。关掉后再访问恢复真实内容。
2. 同一页关闭「JS SDK」接入，JS SDK 接入的页面应改为审核；打开后恢复。限流数字改小后用开放接口连发请求应收到 429，无需重启网关。
3. 「全部站点」对某个启用中的站点点「强制停用」。用该站点域名访问应得到审核或拦截内容；进入商户端，该站点显示「平台停用」标签，点「启用」提示「该站点已被平台强制停用，请联系平台恢复」。总后台点「恢复启用」后商户端恢复正常。
4. 「商户账号 → 更多 → 模拟登录」，浏览器新开商户端并直接进入商户首页。总后台操作日志里有 `POST /api/admin/merchant/<id>/impersonate`，商户登录日志有一条「模拟登录」。把链接再打开一次应提示票据无效。
5. 「全平台分销码」搜索任意商户的码并强制停用，该码对应的访问应不再被归因；恢复后再次归因。
6. 「管理员」新建一个运营管理员账号登录：能看系统配置但保存按钮不出现，看不到管理员 / 角色 / 菜单。尝试删除当前账号或内置角色应被拒绝。
7. 「默认模板」新建一个模板，商户端审核内容选择模板时能看到；删除正在使用的模板应被拒绝。
8. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 上一迭代记录：I11 看板、报表、告警、Webhook

- [x] 迁移 `20261104000007`：`metrics_hourly`、`code_stats_daily`、`path_stats_daily`、`alert_rules`、`alerts`、`notifications`、`notify_settings`、`webhook_endpoints`、`webhook_deliveries`、`exports`
- [x] worker 每分钟重算近 3 小时的小时指标和当天的分销码/路径统计，启动时回补 48 小时；网关每 15 秒写节点心跳（QPS、P99、回源错误率、版本）
- [x] 商户工作台：站点与时间范围、额度条、访问/放行/审核/拦截/校验失败/转化、按小时或按日趋势（内容、触点）、渠道与分销码排行、置信度、内置浏览器、国家、校验失败原因、接入方式、域名与源站健康
- [x] 报表：渠道、分销码、活动码、路径、转化；聚合与原始日志对账；异步 CSV 导出，24 小时签名链接下载，篡改链接 403
- [x] 告警：8 类默认规则（可改、可恢复默认），源站/域名/证书问题消失后自动关闭；站内、邮件、Telegram、Webhook。平台未配置 SMTP 或 Bot 时对应通道记 `skipped:not_configured`，测试按钮禁用
- [x] Webhook：`v1=` + HMAC-SHA256(secret, `时间戳.正文`)，失败按 1 分/5 分/30 分/2 小时/6 小时重试，最多 5 次，可手动重发；页面附 Node.js / PHP / Python 验签示例
- [x] 总后台工作台：全平台流量、网关节点、平台规模、接入方式、不可达源站、未恢复告警
- [x] 顶栏铃铛读站内通知，未读数与「告警与通知 → 通知设置」里的最近通知一致
- [x] 当前小时的实时日志按「替换同一小时已聚合的数」并入看板，避免重复累加
- [x] 单测：`applyLiveHour` / `patchTrend`、Webhook 签名
- [x] 浏览器：商户看板（演示站点 89 次访问与日志一致、近 7 日按日、触点切换）、报表对账一致、路径与转化、导出完成可下载、告警规则与站内测试通知、Webhook 测试投递成功、总后台节点卡与不可达源站
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I11 验收步骤

1. 商户端打开「商户首页」，选「演示：运营看板」（或自己的有流量站点）和「今日」：访问、放行、审核与「报表 → 对账」的数字一致；点「触点」趋势改为首访/回访/交接，点「近 7 日」副标题变为「按日」。
2. 「报表」依次看渠道、分销码、活动码、路径、转化；点「对账」应提示与原始日志一致（刚产生的流量最多差一分钟）。点「导出当前报表」，到「导出记录」刷新，状态变为已完成后下载 CSV。
3. 把某个站点的真实源站改成一个连不上的地址并发布，一分钟内「告警与通知」出现「源站不可达」，顶栏铃铛有未读；把源站改回可访问的地址，下一分钟告警变为已恢复。
4. 「Webhook」新增一个能收到 POST 的地址，点「测试」。用页面上的 Node/PHP/Python 示例校验 `X-Cloak-Signature`。把接收端改成返回 500 后点「重发」，投递记录应为失败，再改回 200 重发应为成功。
5. 总后台「平台首页」能看到今日流量、至少一个网关节点（QPS / P99 / 回源错误率 / 版本）和未恢复告警。节点刚重启、最近 60 秒没有请求时 QPS 为 0。
6. 邮件和 Telegram 需要在 `configs/admin.yaml`、`configs/worker.yaml` 的 `notify` 里填 SMTP 和 Bot。未配置时测试按钮是灰的，告警仍会写站内通知和 Webhook。
7. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 上一迭代记录：I10 WordPress/WooCommerce 插件 v2 与 JS SDK

- [x] 公钥索引 `pk:{publishableKey}` 随快照发布/删除；网关 `Store.ByPublishableKey`、`Loader.ResolvePublishableKey`
- [x] `POST /api/open/v2/js/capture`（无需签名）：公钥 + `Origin` 校验（必须是站点已绑定域名，本地开发放行 localhost/127.0.0.1）、CORS 预检、每站点限流、同一引擎判定（entry `js`）、同一日志管道；返回 content/触点/置信度/渠道/码/内置浏览器/令牌与 cookie 元数据/`handoffUrl`/`handoffBanner`，不含 renderHint
- [x] JS SDK `GET /_cloak/sdk.js`（任意 Host，缓存 1 小时）：`data-key` 公钥、`data-endpoint`/`data-cookie`/`data-banner="off"`；上报后切换 `[data-cloak="real|review"]` 区块、写访客 Cookie、注入引导条（x-safari-/intent://，`_ck_hb` 关闭记忆）、`cloak:decision` 事件（冒泡到 window）、`window.Cloak.getVisitorToken()/show()`；3 秒超时按审核处理
- [x] WordPress 插件 v2 `wordpress-plugin/cloak-attribution/`（保留 v1 `cloak-api/`）：`Cloak_Attr_Signer` v2 签名；`Cloak_Attr_Client` 每请求调 `v2/decision`、按返回写 Cookie、回访判定 transient 缓存（仅「已有令牌且无参数」，带参数始终实时）；`Cloak_Attr_Gate` 在 `template_redirect` 优先级 1 判定，审核内容在当前 URL 以 200 输出（平台模板 HTML / 站内页面保持 URL / 服务器 HTML 文件 / 服务端拉取外部地址，均 `noindex`）；`block`→403、`not_found`→404；`wp_footer` 引导条注入；跳过后台/AJAX/Cron/REST/Feed/已登录（可选）/排除路径（`/path/*`）/审核页自身；`Cloak_Attr_Woo` 全部商店页面或仅指定商品 ID/分类 slug（购物车与结账按车内商品）；设置页（网关地址、密钥、超时、降级、缓存、输出方式、引导条、排除、Woo 范围）与「测试连接」（ping + 签名判定，按错误码给中文提示）；`CLOAK_API_BASE`/`CLOAK_SECRET` 常量覆盖；激活时从 v1 `cloak_api_settings` 迁移（模式映射、排除合并，`ck_` Key 不迁移并提示填 `sk_`）
- [x] 商户端「接入指南」加 WordPress 插件四步 + `wp-config.php` 片段、JS SDK 引入示例（自动带入站点公钥）；`openapi.yaml` 加 `/_cloak/sdk.js` 与 `v2/js/capture`
- [x] 单测：`originAllowed`（绑定域名/端口/localhost/子域伪装/空/非法）、js/capture 预检/未知公钥/Origin 拒绝/缺字段/带码首访/回访/无归因、SDK 脚本响应；PHP 文件 `php -l` 通过
- [x] 端到端冒烟 9 组：sdk.js 可取；预检 204 带 ACAO；带码首访 real + cookie + 微信 UA 下 handoffUrl 与 banner；回访 return；localhost Origin 无归因 review；错 Origin 403 `origin_not_allowed`、未知公钥 401、缺字段 400；插件 `Cloak_Attr_Signer` 直签 `v2/decision` 200 real `hostKnown=true`；判定日志 entry 出现 `js`
- [x] 浏览器验证（本地静态页 + 网关 SDK）：带码访问真实区块显示、Cookie 写入、`cloak:decision` 事件带 attributed/channel；微信 UA 下底部引导条 `href` 含 `_ck`；清 Cookie 无参数访问显示审核区块
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I10 验收步骤

1. 商户端「接入指南」选站点，JS SDK 段落复制示例到一个静态 HTML（域名须绑定到该站点，或本地用 localhost），带 `?ref=<已登记码>` 打开：真实区块显示、`_ck_vid` Cookie 写入；换无痕窗口不带参数打开：审核区块显示。
2. 用微信/Facebook UA 带码打开同一页：底部出现「在浏览器中打开」引导条，链接含 `_ck=`；点 × 后刷新不再出现。
3. 把示例放到未绑定域名下打开：接口 403 `origin_not_allowed`，页面按审核区块显示。
4. WordPress 测试站安装 `wordpress-plugin/cloak-attribution`，设置页填网关地址与 `sk_` 密钥，「测试连接」显示连接成功；勾选启用后：带码访问任意页面看到真实页面并种 Cookie，无 Cookie 无参数访问同一 URL 看到审核内容（状态 200、地址不变、`X-Cloak-Content: review`）；后台与登录页不受影响。
5. WooCommerce：范围选「只判定下列商品 / 分类」，填一个商品 ID：该商品页判定，其他商品页直接真实内容；把该商品加入购物车后购物车页也判定。
6. 停用网关后访问：按「网关不可用时」设置显示审核或真实内容，页面不报错。
7. 装过 v1 `cloak-api` 的站点激活 v2 插件：设置页已带入网关地址、审核模式与排除路径，顶部提示需填写 `sk_` 密钥。
8. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 上一迭代记录：I9 跨浏览器交接

I5/I6 已落地的部分：内置浏览器 UA 识别与设备族、广告 Referer 表、置信度接入判定、交接令牌签发/`_ck` 消费/次数与过期、同设备匹配候选登记/窗口匹配/一次性消费、引导条注入与站内链接延续、`Accept-Encoding: identity` 回源。本迭代补齐：

- [x] 内置浏览器内的回访（带 Cookie 翻页）同样签发交接令牌，访客在第二页点「用浏览器打开」也能接上
- [x] 交接记录携带来源请求 ID（令牌与匹配候选都存 `rid`），`Decision.HandoffFromReqID` → 日志 `handoff_from_request_id`（迁移 `20261104000006`），详情页「交接来源」可直接跳到那次放行
- [x] 站点开关「只匹配系统浏览器」（`handoff_match_system_only`，默认开），关闭后允许 App 内请求参与匹配；站点配置页新增开关
- [x] 引导条关闭后写会话 Cookie `_ck_hb=1`，之后不再注入引导条，链接延续不受影响
- [x] 注入 HTML 后按客户端 `Accept-Encoding` 由网关 gzip（源站已被要求 identity），`Vary: Accept-Encoding`
- [x] UA 样本库补到 20 条（Facebook/Instagram/TikTok/Snapchat/Line/Twitter/LinkedIn/微信/Kwai/Google App，Chrome iOS、Samsung Chrome 等系统浏览器为反例）
- [x] 单测：回访签发令牌、来源请求 ID 传递、只匹配系统浏览器开关、引导条关闭、gzip
- [x] `docs/v2/handoff-verification.md`：真实手机五组验证脚本（令牌交接、翻页后交接、同设备匹配、系统浏览器不注入、置信度）与记录模板
- [x] 本地端到端冒烟 10 组：App 首访有引导条；令牌交接成功且来源指向首访；翻页后令牌来源指向翻页请求；关闭引导条后不注入但链接仍带 `_ck`；令牌第 4 次失效；同 /24 同设备族匹配成功、第二次不匹配、不同设备族不匹配、App 内请求默认不匹配、关闭开关后匹配；系统浏览器直接访问不注入；置信度 App+fbclid 高、Safari+fbclid 中
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I9 验收步骤

1. 本地：按 I6 步骤起站点与模拟源站，用 Facebook UA（`[FBAN/FBIOS;FBAV/430.0.0]`）带码访问网关，页面底部有引导条，`href` 含 `_ck=`。
2. 用 Safari UA 打开该 `_ck` 链接：真实内容；决策日志为「交接 · 令牌」，详情「交接来源」点进去是第 1 步那条。
3. 用第 1 步的 Cookie、Facebook UA 访问 `/second`：仍有引导条；带 `Cookie: _ck_hb=1` 再访问：没有引导条。
4. 用 Facebook UA 带码访问（`X-Forwarded-For: 203.0.113.50`，需网关 `trusted_proxies` 含 127.0.0.1），再用 Safari UA 不带参数访问 `/lp`（`X-Forwarded-For: 203.0.113.77`）：真实内容，日志「交接 · 匹配」、置信度低；再来一次：审核内容。
5. 站点配置页关闭「只匹配系统浏览器」保存后，用 Facebook UA 重复第 4 步的第二次请求：可以匹配。
6. 真实手机：按 `docs/v2/handoff-verification.md` 在 Facebook / Instagram / TikTok 三个 App 内完成五组测试并填记录模板（需要公网部署）。
7. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 上一迭代记录：I8 服务端判定接口 v2 与 V1 兼容

- [x] 迁移 `20261104000005`：`sites.secret_enc`（AES-256-GCM 加密的服务端密钥，主密钥 `auth.secret_key`，admin 与 gateway 必须一致）、`conversions`（`(merchant_id, order_no, status)` 唯一）、`visitor_emails`
- [x] 鉴权 `transport/http/openapi`：v2 `X-Cloak-Key`（前缀 12 位）+ `X-Cloak-Timestamp`（±300 秒）+ `X-Cloak-Nonce`（8–64 位，Redis 5 分钟防重放）+ `X-Cloak-Signature`（`HMAC-SHA256(secret, ts\nnonce\nMETHOD\npath\nsha256(body))`）；V1 `X-API-Key` / `Bearer` 常量时间比对；站点停用、密钥无加密副本、IP 白名单；两级限流每密钥 50/秒、每站点 2000/分
- [x] 接口（运行在网关进程内，Host 不是商户站点域名时命中 `/api/open/`）：`v2/ping`、`v2/decision`（含 cookie 指令、`handoffUrl`、审核模板 `renderHint`、`hostKnown`）、`v2/attribution/capture`、`v2/attribution/resolve`（读 `attr:` 缓存）、`v2/conversions`（入流，202）、`v2/promo-codes` 列表/登记/批量/修改/删除、`v2/decision-logs`、V1 `decision`（返回 `page`/`action`/`hitRule`，兼容 `host/path/query` 字段）
- [x] 开放接口判定走同一引擎、同一日志管道（entry `api_v2`/`api_v1`），计量与额度策略同托管入口；每次调用写 `api_logs`
- [x] worker 转化消费：幂等写入、关联访客归因、分销码转化数 +1、邮箱哈希 → 访客映射，同邮箱多访客时把后来者归因 `merged_into_visitor_id` 指向最早访客
- [x] 密钥前缀索引 `sk:{prefix}` 随快照发布，网关 `Store.ByPrefix`
- [x] 文档 `backend-go/docs/openapi/openapi.yaml`；商户端「接入指南」页：选站点与语言后示例自动带入网关地址、密钥前缀、绑定域名；签名函数 + 四种场景（页面判定 / 只做归因 / 下单查归因 / 支付回传）× PHP/Node/Go/Python，V1 兼容示例，错误码表
- [x] 单测：签名正确/过期/篡改/错密钥/未知前缀/缺头/nonce 长度、V1 Key 与 Bearer、IP 白名单、站点停用、无加密副本
- [x] 端到端冒烟 11 组：v2 判定首访/回访/无归因 renderHint；错误签名、过期、重放、未知 key 均 401；V1 返回 `page`；capture→resolve 查到渠道与码；转化回传后码转化数 1、重复回传不重复计数；开放接口登记码后停用立即影响判定；decision-logs 可查；IP 白名单 403
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I8 验收步骤

1. 商户端「接入指南」选择站点与语言，复制签名函数与场景一代码到你的服务端（或用 `backend-go/docs/openapi/openapi.yaml` 生成客户端）。密钥用站点配置页「重置服务端密钥」拿到的 `sk_...`。
2. 调 `POST /api/open/v2/decision`，`url` 带已登记分销码：`data.content=real`、`touchType=first`、有 `cookie`；把 `cookie.value` 作为 `visitorToken` 再调一次不带参数的 URL：`touchType=return`。不带参数也不带令牌：`content=review`，`renderHint.type=html`。
3. 把签名改错一位 → 401 `bad_signature`；时间戳减 600 → 401 `timestamp_expired`；同一 nonce 发两次 → 第二次 401 `nonce_replayed`。
4. 旧方式 `POST /api/open/decision` + `X-API-Key: sk_...`，body 用 `host/path/query`：返回 `data.page=real|review`。
5. `attribution/capture` 后用返回令牌调 `attribution/resolve`：`found=true`，`channel` 与 `code` 正确。
6. `conversions` 回传同一 `orderNo` 两次：均 202，「分销码」页该码转化数只加 1。
7. 用开放接口 `promo-codes` 登记一个码再 PUT `status=disabled`，随后判定该码返回 `review` / `disabled`。
8. 站点密钥 IP 白名单填一个不含本机的网段后调用 → 403 `ip_not_allowed`。
9. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 更早迭代记录：I7 决策日志与归因管道

- [x] 迁移 `20261104000004`：`decision_logs`（`request_id` 唯一，回放幂等）、`visitors`、`attributions`（`(site_id, visitor_id)` 唯一）、`usage_counters`、`api_logs`
- [x] 网关 `internal/logpipe`：环形缓冲（1 万）→ 每 200ms 或 500 条批量 `XADD` 到 `decision_logs` / `attribution_writes`；Redis 写失败切到本地 WAL（`.local/wal/*.wal`，超 256MB 丢最旧），每 3 秒探测恢复后回放并删除文件；写入永不阻塞请求
- [x] 事件字段按技术方案 11.3：查询参数名、触点、内容、置信度、内置浏览器、设备族、交接方式、渠道、码、命中参数、校验失败、访客/归因 ID、IP 前缀与哈希（不存明文）、国家、UA/Referer 摘要、路径规则、响应来源与源站状态、兜底、耗时、降级、判定链路
- [x] 计量 `internal/quota`：每次非 bypass 判定 `INCR usage:{merchant}:{yyyymm}`；超额按商户策略 `alert`（照常）/ `review_all`（真实→审核）/ `deny`（403），用量本地缓存 5 秒不增加延迟；快照携带商户额度，额度变更即时发布
- [x] worker `internal/worker/streams`：消费组 `workers`，批量写决策日志、逐条应用归因（访客 upsert、归因首次字段只写一次、最近字段每次更新、触点/交接计数、分销码访问/首访/回访统计，回访不带码时按归因最近码累计）、回填 `attr:{site}:{vid}` 缓存、月度 `usage_counters`；失败不 ACK 下次重试；启动先处理未确认积压
- [x] 保留清理 `worker/tasks`：每 6 小时按商户 `log_retention_days` 分批删除
- [x] GeoIP `adapter/geoip`：有 `GeoLite2-Country.mmdb` 时解析国家，没有则留空
- [x] 查询接口：商户端/总后台 `decision-log/list`（时间、站点、路径、触点、内容、渠道、码、失败原因、国家、内置浏览器、访客、请求 ID）、`/:requestId` 详情（链路、校验失败、归因、访客）、`/stats`（24 小时统计 + 当月用量）
- [x] 前端：决策日志页（概览卡片可点筛选、搜索卡片含「更多条件」、列表、详情抽屉：Hero 摘要条 → 判定链路时间线 → 请求信息 → 归因 → 访客，可跳到首访请求）；总后台复用同一页面多一列商户
- [x] 端到端冒烟：首访 ~500ms 内可查且链路完整；回访后归因触点 2、访客 2 次、码统计首访 1 回访 1；按失败原因/触点/访客筛选；总后台看到商户名、跨门户 403；Redis 写暂停 8 秒期间网关照常返回真实内容、14 条进 WAL，恢复后全部补齐；额度 1 + `review_all` 带码也返回审核且链路标 `quota`，`deny` 返回 403，恢复后正常
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I7 验收步骤

1. 启动 mock 源站、admin、gateway、worker。建站绑自有域名、配两个源站、登记码，用 Facebook UA 带码访问网关一次，记下响应头 `X-Request-Id`。
2. 30 秒内在商户端「决策日志」看到该请求：真实内容 / 首访 / 置信度高 / 内置浏览器 facebook；点详情能看到判定链路、识别到的参数、归因首次渠道、访客首次设备。
3. 带 Cookie 再访问一次：日志为回访，详情里归因触点 2、访客访问 2 次；「分销码」页该码访问 2、首访 1、回访 1。
4. 用「更多条件 → 失败原因 = 未登记」筛选，能筛出 `ref=NOPE` 那条。
5. 总后台「决策日志」能看到商户列；商户端只看到自己的。
6. `redis-cli CLIENT PAUSE 8000 WRITE` 后再访问网关：仍返回正确内容，`/_cloak/status` 的 `logpipe.walled` 增加；UNPAUSE 后 10 秒内日志出现在后台。
7. 总后台把演示商户判定额度改为 1、策略「全部审核」：带码访问返回审核内容，详情链路出现 `quota=over:review_all`；策略「拒绝」返回 403；恢复额度后正常。
8. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 更早迭代记录：I6 托管反向代理与证书

- [x] `internal/proxy`：按判定回源真实站/审核站或直出模板/静态包/403/404/维护页；每源站 Transport（连接池、拨号与响应头超时、对源站不开 HTTP/2、不解压）
- [x] 请求改写：Host 换回源 Host、追加 `X-Forwarded-*`/`X-Real-IP`、追加 `X-Cloak-Content/Touch/Visitor/Request-Id`、删除客户端伪造的 `X-Cloak-*`、源站 URL 路径前缀拼接
- [x] 响应改写：删 `Server`/`X-Powered-By`/`Via`，`Location` 指向源站时改回商户域名，源站 `Set-Cookie` 的 `Domain=源站` 去掉；真实与审核响应统一 `Cache-Control: private, no-store` + `Vary: Cookie`；审核内容加 `X-Robots-Tag: noindex`
- [x] 流式转发（SSE 即时 flush）、WebSocket Hijack 双向拷贝、请求体上限 413、幂等请求建连失败重试一次
- [x] 源站失败兜底：真实站不可达按 `fail_mode`（审核内容 / 维护页 503 / 502），审核站不可达退回模板；未配源站同样处理；`bypass` 不写 Cookie
- [x] 内置浏览器放行且响应为 HTML：向源站请求 `identity`，在 `</body>` 前注入引导条（iOS `x-safari-https://`，Android `intent://`），对同域 `<a href>`/`<form action>` 追加 `_ck`，外域与锚点不动；系统浏览器不注入
- [x] 静态包直出：按路径取文件，不存在回 `entry`（SPA），`..` 不越界，`Cache-Control: no-store`
- [x] TLS `pkg/tlsconf`：`selfsigned`（本地 CA 持久化到 `.local/certs/selfsigned/ca.crt`，按 SNI 签发）与 `acme`（certmagic 按需签发，HTTP-01/TLS-ALPN-01，staging 可切）；两种模式都只对快照中已绑定域名放行，未绑定 SNI 拒绝握手；`httpserver` 增加 HTTPS 监听；可选明文跳 HTTPS
- [x] 网关：`X-Cloak-Debug: decide` 请求头返回判定 JSON 便于排障；`gateway_proxy_responses_total{kind,fallback}` 指标；记录回源失败日志
- [x] 站点 API：`handoffBanner`/`handoffLinkRewrite`/`handoffMatchEnabled` 未传时取默认（开/关/开），修正此前 API 建站引导条默认关的问题
- [x] 本地工具：`cmd/mockorigin`（真实/审核模拟源站，回显头、302、Set-Cookie、SSE）、`scripts/dev/start-mock-origins.ps1`；`stop-all.ps1` 一并清理
- [x] 部署：`deploy/install.sh`（构建、安装到 `/opt/cloak`、systemd unit、迁移提示）、三份 unit（网关 `CAP_NET_BIND_SERVICE` 绑 80/443）、测试服务器配置模板、`deploy/README.md`
- [x] 单测：proxy 85.1%（头改写、Location、Cookie Domain、三种审核来源、兜底模式、413、HTML 注入、SSE、WebSocket）、tlsconf 66.7%、gateway 83.3%
- [x] 端到端冒烟 16 项（真实/审核模拟源站 + 网关 + 自签 HTTPS）全部通过
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I6 验收步骤

1. `scripts/dev/start-mock-origins.ps1`，再启动 admin 与 gateway。商户端建站、加自有域名并校验、真实源站填 `http://127.0.0.1:9101`、审核源站 `http://127.0.0.1:9102`、审核内容选「审核源站」、登记分销码 `PX1`。
2. `curl -s -H "Host: <域名>" http://127.0.0.1:8090/lp` 返回 **REVIEW SITE** 页面，响应头有 `X-Robots-Tag: noindex`，没有 `Server`。
3. `curl -si -H "Host: <域名>" "http://127.0.0.1:8090/lp?ref=PX1"` 返回 **REAL SITE** 页面与 `Set-Cookie: _ck_vid=...`；页面里回显的请求头含 `X-Cloak-Content: real`、`X-Forwarded-For`，Host 为回源 Host。
4. 带该 Cookie 访问 `/second` 仍是 REAL SITE；不带 Cookie 是 REVIEW SITE。
5. `/redirect` 的 `Location` 已改为商户域名；`/setcookie` 返回的源站 Cookie 不带 `Domain=`。
6. 用 Facebook 内置浏览器 UA 带码访问：页面底部有「在浏览器中打开」引导条，站内链接带 `_ck=`；用该 `_ck` 以 Safari UA 访问 `/second` 返回 REAL SITE。
7. 把真实源站改成不可达地址：带码访问按「失败兜底」返回审核内容或 503 维护页；审核源站也不可达时返回平台模板。
8. `curl -sk --resolve <域名>:8443:127.0.0.1 https://<域名>:8443/lp` 返回 200；把域名换成未绑定的，握手失败。
9. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。
10. 公网部分（需要服务器与域名）：按 `deploy/README.md` 部署后，在手机浏览器完成「带码首访真实内容、同窗口再访真实内容、新无痕窗口审核内容、证书自动签发」。

## 更早迭代记录：I5 配置快照与判定引擎

- [x] `internal/engine`：纯函数引擎，输入 `RequestContext` + 站点快照，输出 `Decision`；Redis 能力通过 `Store` 接口注入，自带 `MemStore`
- [x] 快照 `engine/snapshot`：JSON 结构 + `Compile()` 预编译（参数别名表、码表、正则、密钥、路径规则排序、域名索引）；网关侧 `Store` 懒加载、Host 索引、负缓存、按 revision 增量替换
- [x] 访客令牌 `engine/token`：`v1.<payload>.<hmac>`，密钥版本、宽限期验签、轮换后换发、跨站点拒绝
- [x] 路径规则、参数提取（别名、去重、类型排序）、校验（登记/停用/过期/未生效/路径/格式/来源/参数停用/点击复用）、组合规则、置信度、渠道推断
- [x] `Decide()` 顺序：站点状态 → 路径规则 → 令牌 → 参数 → 首访 / 回访 / 交接 token / 交接匹配 / 无归因；回访按 sliding/fixed 续期；内置浏览器首访签发交接 token 并登记匹配候选
- [x] UA 识别 `engine/ua`：Facebook/Instagram/TikTok/Snapchat/Twitter/LinkedIn/Line/微信/Kwai 等内置浏览器，iOS/Android 设备族
- [x] 快照构建与发布 `app/snapshotsvc`：站点/域名/源站/审核内容/路径规则/参数（站点覆盖内置）/规则/分销码 → `cfg:site:{id}`、`host:{domain}`、`cfg:changed`
- [x] 管理端所有站点级写操作（站点、源站、审核内容、路径规则、域名、参数、规则、分销码、商户状态、内置字典）成功后异步发布；admin 启动时全量刷新
- [x] 网关：Redis 订阅 `cfg:changed` 增量替换；按 Host 找站点 → 判定 → 写 Cookie → 返回判定 JSON（I6 替换为回源）；`/_cloak/status` 上报各站点 revision；可信代理 XFF
- [x] 访客令牌密钥轮换接口（宽限期可配，0 等于强制全部重新归因）；判定测试接口（dry run）与快照状态接口
- [x] 前端：商户端「判定测试」页（模拟访问 + 配置快照状态 + 结果/校验失败/判定过程/Cookie）；站点配置页新增访客令牌密钥轮换
- [x] 单测：`go test ./internal/engine/...` 合计覆盖 88.3%（engine 88.6 / params 77.7 / pathrule 100 / snapshot 93.5 / token 90.9 / ua 96.8），网关 92.6%
- [x] 端到端冒烟 16 项：登记分销码后网关 343ms 内从 review 变为 real；停用码后新访客 review、老 Cookie 仍放行；路径规则；内置浏览器 + 点击参数 → high + 交接 token；新浏览器带 `_ck` → handoff；密钥轮换宽限期内老 Cookie 放行并换发；站点停用；删站后 404
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I5 验收步骤

1. `go test ./internal/engine/... -cover`：全部通过，engine 包覆盖率 ≥ 85%。
2. 启动 admin 与 gateway（`scripts/dev/start-service.ps1 -Name gateway -Background`），`curl http://127.0.0.1:8090/_cloak/status` 能看到已加载站点的 revision。
3. 商户端为站点添加自有域名并校验，用 `curl -H "Host: 该域名" http://127.0.0.1:8090/lp?ref=XXX` 访问：未登记码返回 `content=review`、`validationErrors[0].reason=not_registered`。
4. 在「分销码」登记 `XXX` 后 5 秒内再次请求，返回 `content=real`、`touchType=first` 且有 `Set-Cookie: _ck_vid=v1....`；带该 Cookie 访问无参数路径返回 `touchType=return`。
5. 停用该码后新访客变为 `review`（原因 `disabled`），带老 Cookie 仍为 `real`。
6. 商户端「判定测试」页：选站点、填地址 `https://域名/lp?ref=XXX`、选「iPhone Facebook 内置浏览器」，结果为真实内容 / 首访 / 置信度高，判定过程逐步可见；配置快照卡片显示「一致」。
7. 站点配置页「轮换（宽限 72 小时）」后老 Cookie 仍放行且响应带新 Cookie；「强制全部重新归因」后老 Cookie 变为 review。
8. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 更早迭代记录：I4 归因规则与分销码

- [x] 迁移 `20261104000003`：`attribution_params`（内置行 `site_id=0`，站点行覆盖同名内置行）、`site_attribution_rules`、`promo_codes`（站点内唯一）
- [x] 内置字典 19 个：fbclid、gclid、gbraid、wbraid、ttclid、twclid、msclkid、kwai_click_id、kwaiclickid、kpid、sclid、li_fat_id、utm_* 五个、`ref`（别名 r/aff/affiliate/referral）、`cid`（别名 campaign/promo），各带长度与正则
- [x] 站点参数：新增自定义、覆盖内置、删除覆盖恢复内置；别名不能与同作用域其它参数名或别名冲突；正则编译校验
- [x] 站点规则：组合要求、是否校验分销码登记、是否校验点击参数格式、允许来源、点击参数复用上限与时间窗、超限动作、码停用后处理方式
- [x] 分销码/活动码：登记、编辑、批量启停、批量删除、批量生成（前缀 + 去易混淆字符随机，500 个 20ms）、文本/CSV 导入（逐行给出冲突原因）、CSV 导出、统计、渠道标签、额度校验
- [x] 链接生成：按站点域名与参数名拼接，分销码须已登记，停用或过期的码给出警告
- [x] 总后台内置字典维护：新增、编辑、删除；商户端立即可见
- [x] 菜单：商户端归因规则 / 分销码 / 链接生成器，总后台参数字典
- [x] 前端：归因规则页（站点选择 + 规则分区卡 + 参数字典表）、分销码页（概览卡、多选批量操作、生成/导入/导出弹窗）、链接生成器（左表单右结果）、总后台参数字典页；参数编辑弹窗两端共用
- [x] 接口冒烟 35 项通过；浏览器验证覆盖与恢复内置、批量生成、生成链接、总后台编辑弹窗
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I4 验收步骤

1. 商户端 `http://127.0.0.1:5174/merchant/attribution`：参数字典列出 19 个内置参数；对 `utm_source` 点「覆盖」打开「单独足以放行」保存后，来源列显示「已覆盖」；点「恢复内置」后恢复为「内置」。
2. 「新增参数」填 `mycode`、别名 `aff` 保存，应提示「aff」已被参数 ref 使用。
3. 修改组合要求为「至少一个点击参数或分销码」、允许来源加 `tiktok` 保存，刷新后仍保留。
4. `http://127.0.0.1:5174/merchant/code`：「批量生成」前缀 `TK`、数量 1000，几百毫秒内完成；导入粘贴含重复与非法行的文本，结果里逐行列出原因；勾选多行停用后状态变为「已停用」；「导出」下载 CSV。
5. 对任一码点「生成链接」，进入链接生成器后域名与码已带入，点「生成链接」得到 `https://域名/?ref=码&...`；对停用的码生成会出现警告。
6. 总后台 `http://127.0.0.1:5173/admin/param`：新增内置参数后，商户端参数字典立即出现；删除后消失。
7. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 更早迭代记录：I3 站点、域名、源站、审核内容

- [x] 迁移 `20261104000002`：站点、源站、审核内容、模板、静态包、域名、路径规则；内置商城与企业官网两个模板
- [x] 站点：创建时一次性返回服务端密钥，详情只保留前缀；公钥、Cookie、归因窗口、无归因动作、失败兜底、跨浏览器接力、密钥 IP 白名单
- [x] 域名：托管域名 CNAME/TXT 真实查询，自有域名只确认归属；重复域名与 IP 拒绝；列表带站点名与商户名
- [x] 源站保存与即时检测，worker 每 2 分钟巡检；审核内容支持模板、审核源站、zip 静态包（本地目录，小于 20MB）
- [x] 路径规则整表保存，新建站点写入默认静态资源与隐私政策规则
- [x] 菜单：商户端站点管理/域名列表，总后台全部站点/全部域名；跨门户访问返回 403
- [x] 前端：商户端站点列表与独立配置页（分组卡片）、域名页与解析指引；总后台站点与域名列表，可强制停用站点
- [x] 浏览器验证：创建站点并弹出一次性密钥；源站保存后检测 example.com 为可达；模板预览带入站点名；添加域名后指引为 `entry.cloak.local`，校验未解析域名提示「未配置 CNAME，且域名无法解析」；总后台能看到商户名、站点名和该结果
- [x] `scripts/dev/check.ps1 -Frontend` 通过

### I3 验收步骤

1. 商户端 `http://127.0.0.1:5174/merchant/site`：概览卡片与列表一致；「新建站点」进入配置页，创建后弹出服务端密钥，关闭后只剩前缀。
2. 配置页保存真实源站 `https://example.com` 后点「检测」，状态变为可达。选「合规商城」模板，填写站点名后「预览」能看到该名称。
3. `http://127.0.0.1:5174/merchant/domain`：添加托管域名，弹出 CNAME 指引，记录值为 `entry.cloak.local`。对未解析域名点「校验」，结果为未配置 CNAME。
4. 总后台 `http://127.0.0.1:5173/admin/site` 能看到该站点与商户名，「强制停用」后商户端状态变为停用。`/admin/domain` 能看到同一条校验结果。
5. 用总后台账号调 `/api/merchant/site/list` 返回 403；商户删掉站点后其域名一并消失。
6. `scripts/dev/check.ps1 -Frontend` 输出 `CHECK PASSED`。

## 更早迭代记录：I2 商户与额度

- [x] 总后台商户：分页筛选、概览统计、开通、编辑、启停/冻结、额度与功能开关、重置管理员密码、详情
- [x] 到期：到期后登录拒绝；清空到期时间后自动从 expired 恢复为启用
- [x] 停用/冻结踢出该商户全部会话；已登录请求再次校验商户状态
- [x] 商户端子账号：列表、新增、编辑、启停、重置密码、删除；额度上限；不能停用/删除自己；至少保留一个启用的商户管理员
- [x] 菜单种子：额度权限、账号管理菜单与按钮；`migrate seed` 幂等补齐
- [x] 看板占位：`/admin/dashboard`、`/merchant/dashboard` 返回 200，避免首页报错（完整指标在 I11）
- [x] 前端：商户列表按 UI 规范（概览卡片、搜索卡片、表格卡片、详情抽屉、分区卡片）；商户端账号管理页
- [x] 接口冒烟 27 项通过；浏览器验证总后台列表与详情、商户端账号列表与新增弹窗
- [x] `scripts/dev/check.ps1` 通过

### I2 验收步骤

1. 总后台 `http://127.0.0.1:5173/admin/merchant`：概览卡片数字与列表一致；点「已冻结」卡片应只筛出冻结商户。
2. 「开通商户」填写名称、管理员账号和密码后保存，弹出账号信息；重复名称被拒绝。
3. 点商户名称打开详情：额度摘要、资料、功能开关与后台设置一致；「额度与功能」修改后刷新即变。
4. 「更多 → 停用」后该商户管理员无法登录；重新启用后可登录。
5. 商户端 `http://127.0.0.1:5174/merchant/user`：能看到本人账号（无停用/删除按钮）；新增操作员后出现在列表；停用后该账号无法登录。
6. `scripts/dev/check.ps1` 输出 `CHECK PASSED`。

## 上一迭代记录：I1 认证与权限

- [x] 迁移 `20261104000001`：merchants、users、roles、menus、user_roles、role_menus、login_logs、operation_logs（含 V2 额度与功能开关字段）
- [x] 密码：argon2id 哈希，兼容 V1 bcrypt（登录成功后自动升级）；密码策略 8–64 位且含字母数字
- [x] JWT + Redis 会话：签发、校验、单会话注销、按用户踢出；Redis 不可用时降级为仅校验 JWT
- [x] 连续失败锁定：5 次锁 15 分钟（可配），Redis 计数
- [x] `/login`（带 `portal`）、`/captchaImage`、`/logout`、`/getInfo`、`/getRouters`
- [x] 门户归属校验：登录与每次请求都校验账号门户与请求门户一致
- [x] 权限中间件 `RequirePerm`、门户中间件 `RequirePortal`、操作日志中间件（写操作自动记录，密码字段脱敏，异步落库）
- [x] 个人中心：资料查看/修改、改密（校验旧密码与策略、踢出其它会话）、头像上传（本地目录 + `/uploads` 静态）
- [x] 字典接口 `/system/dict/data/type/:type`（内置 3 个基础字典）
- [x] 种子 `migrate seed`：5 个角色、admin/merchant 账号、演示商户（默认额度按 D2/D3）、现有前端页面对应的菜单与按钮
- [x] 前端：登录请求携带 `portal`；`VITE_API_URL` 切到 Go；两个门户登录、菜单、个人中心验证通过
- [x] 单测：密码、JWT/权限匹配、菜单树、操作日志模块推导与脱敏；门禁通过

### I1 验收步骤

1. `cd backend-go && go run ./cmd/migrate up && go run ./cmd/migrate seed`。
2. 启动 admin 服务与两个前端（`scripts/dev/start-service.ps1 -Name admin -Background`、`npm run dev:admin`、`npm run dev:merchant`）。
3. 总后台 `http://127.0.0.1:5173`：用 `merchant/merchant123` 登录应被拒绝并提示「该账号属于商户端」；用 `admin/admin123` 登录进入平台首页，左侧 6 个菜单。
4. 商户端 `http://127.0.0.1:5174`：`merchant/merchant123` 登录进入商户首页，左侧 7 个菜单。
5. 右上角个人中心：修改昵称、邮箱保存成功；修改密码时旧密码错误提示「旧密码错误」，新密码 `short` 提示长度不足。
6. 任意账号连续输错 5 次密码后提示锁定，第 6 次返回「请 N 分钟后再试」。
7. MySQL `login_logs` 有 success/failed/locked/logout 记录，`operation_logs` 有 `system.user.profile updatePwd` 等记录且 `request_summary` 中密码为 `***`。
8. `scripts/dev/check.ps1` 输出 `CHECK PASSED`。

### 已完成迭代摘要

- I0 工程基线：仓库、Go 骨架、本地脚本、门禁。验收步骤见 git 标签 `v2-i0` 的 progress.md。

## 待决策

| 编号 | 事项 | 影响迭代 | 状态 |
| --- | --- | --- | --- |
| D1 | 超级管理员初始账号名 | I1 | 已定：`admin` |
| D2 | 商户默认额度 | I2 | 已定：判定 100 万/月、站点 5、域名 20、分销码 1000、子账号 10、日志 90 天 |
| D3 | 功能开关默认值 | I2 | 已定：托管/服务端/JS SDK/转化回传/Webhook 开启，增强层关闭 |
| D4 | V1 未上线、数据无需保留 | I13 / I14 | 已定（2026-11-04）：不做 V1 数据迁移与影子对比，I13 收窄为回放回归工具 + V1 清理；I14 的「灰度切换」改为全新上线 |
| D5 | 是否保留 V1 兼容判定接口（`POST /api/open/decision` + `X-API-Key`、`api_v1` 入口） | I14 | 待定：V1 无存量插件，可在 I14 删除以减少接口面；保留则无额外成本 |

## 已知缺陷与技术债

| 编号 | 描述 | 计划 |
| --- | --- | --- |
| T1 | 便携版 Go/MySQL/Redis 复用自 `D:\pay-virtual\.runtime`，路径写在 `scripts/dev/common.ps1` | 后续换独立安装只改该文件 |
| T2 | 本地无 golangci-lint，门禁中跳过 | 安装后自动启用 |
| T3 | 本地无 ClickHouse，决策日志用 MySQL（`decision_logs` 单表 + 索引，按商户保留期删除）；`LogRepo` 是唯一写入/查询入口，换 ClickHouse 只改适配器 | 测试服务器到位后接 ClickHouse，日志量大时切换 |
| T13 | ~~小时聚合表尚未建，看板直接查日志~~ | I11 已建 `metrics_hourly` / `code_stats_daily` / `path_stats_daily`，worker 每分钟重算 |
| T14 | ~~商户端与总后台「API 调用日志」页仍是 V1 旧页，没有改成读 `api_logs`~~ | I12 已改读 `api_logs`，默认近 7 天，可按站点与接口筛选 |
| T15 | ~~I8 之前创建的站点没有密钥加密副本，开放接口会返回 `secret_unavailable`~~ | 只影响本地早期演示站点，重置一次密钥即可；无 V1 数据需要导入（D4） |
| T16 | ~~V1 `ck_` 前缀 API Key 尚未映射~~ | 无 V1 存量 Key（D4），不再需要；是否连兼容接口一起删见 D5 |
| T17 | 开放接口在网关进程内连 MySQL（分销码/日志接口）；网关多节点时每个节点都需 DB 访问 | 可接受；或 I14 把这两组接口移到 admin 域名 |
| T4 | 管理 API CORS 当前放行任意 Origin，生产由 Nginx 收紧 | I14 部署时处理 |
| T5 | 头像与审核静态包都存在本地目录，多节点部署需切对象存储 | I6 接入对象存储后切换 |
| T6 | 工具链固定 Go 1.23；`go get @latest` 会把 go.mod 拉到新版本，执行时需 `GOTOOLCHAIN=local` | 已在脚本中设置 |
| T7 | 快照用 JSON 而非压缩 MsgPack；大站点（> 10 万码）快照会膨胀 | I7 或压测后按技术方案 7 节改为哈希集合 + Redis 详情 |
| T8 | ~~网关在 I6 之前对站点请求返回判定 JSON 而非回源内容~~ | I6 已接入反向代理；`X-Cloak-Debug: decide` 保留 JSON 用于排障 |
| T10 | 证书存本地目录，多网关节点需共享（对象存储或 Redis 存储后端） | I14 多节点部署前处理 |
| T11 | 静态包、模板响应未做磁盘/内存缓存，大流量下每次读文件与渲染 | 压测后按需加 LRU |
| T12 | ~~引导条文案与链接延续只处理原始 HTML，脚本动态生成的链接需 JS SDK 补齐~~ | I10 JS SDK 已提供引导条；SDK 暂不改写动态链接的 `_ck`（页面内跳转由 Cookie 延续） |
| T18 | WordPress 插件 v2 只做了 `php -l` 与签名器直连网关验证，未在真实 WordPress/WooCommerce 站点跑过（本地无 WP 环境）；站内页面模式 `render_wp_page` 对部分主题/页面构建器可能需要调整 | 测试服务器到位后装 WP + Woo 按 I10 验收步骤 4–7 验证 |
| T19 | JS SDK 的审核/真实内容都在页面源码里，只是切换显示；SDK 不做严格隐藏（文档已注明），`_cloak/sdk.js` 公钥模式限流只按站点（限额已可在系统配置热调） | 需要严格隐藏用服务端或托管接入；滥用时再加按 IP 限流 |
| T20 | 插件审核页「站内页面」模式用 `WP_Query` 替换主查询渲染，整页缓存插件会把首个访客的页面缓存给所有人 | `docs/v2/runbook.md` 第 8 节已写排除方法；插件 readme 已提示关闭整页缓存 |
| T30 | `docs/prd.md`、`docs/technical-design.md`、`docs/deploy-linux.md`、`docs/deploy-debian-bt.md` 仍是 V1 文档 | 新部署以 `docs/v2/deploy-guide.md` 为准；旧文档待观察期结束后删除 |
| T33 | 测试机证书是 Let's Encrypt staging，浏览器不信任；控制台目前是 HTTP | 切正式域名后按 runbook 第 6 节关掉 staging，并为控制台补 HTTPS |
| T31 | 回放工具只能重建首访请求：回访 / 交接依赖令牌，报告归为 `needs_token`；参数值按归因结果重建，校验失败类请求归为 `param_reconstructed` | 可接受；需要精确回放时在网关加「记录原始查询串」的采样开关 |
| T32 | 回放干跑仍会在引擎里签发访客令牌与写访客记录（只是不写决策日志、不计额度） | 可接受；令牌不会被使用，访客表按站点保留期清理 |
| T9 | ~~广告 Referer 域名表与内置浏览器 UA 特征写死在代码里~~ | I12 已由总后台「系统配置」维护，经 `cfg:global` 下发；关闭自定义即恢复内置表 |
| T27 | 全局配置依赖 Redis：Redis 不可用时网关沿用上次加载的配置（进程重启后为默认值，即不紧急审核、各入口开启） | 可接受；部署时 Redis 按高可用配置 |
| T28 | 总后台菜单管理只能改名称、图标、排序、显示，不能新增或删除菜单；新页面随版本由 `migrate seed` 写入 | 按需再开放 |
| T29 | 模拟登录只能登录商户的管理员账号，未提供「以某个子账号身份登录」 | 按需再加 |
| T21 | 本地没有 SMTP 和 Telegram Bot，邮件与 Telegram 只记 `skipped:not_configured`，页面上的测试按钮保持禁用 | 用户提供账号后写入 `notify`，再按 I11 验收步骤 6 验证真实送达 |
| T22 | 分销码、路径报表和触点细分读聚合表，新流量最多晚一分钟；访问/放行/审核/拦截已用当前小时日志替换同一小时的聚合，避免重复计算 | 可接受；转化笔数直接读 `conversions` |
| T23 | 本地 `configs/worker.yaml` 打开了 `notify.webhook.allow_http`，允许把 Webhook 打到 http 和内网地址。默认值是 false | 生产配置不要打开 |
| T24 | 导出的 CSV 存在 admin 与 worker 共用的本地目录 | 多节点部署时改共享卷，与 T5/T10 同类 |
| T25 | 告警每分钟逐个启用站点评估 | 站点变多后再批量化 |
| T26 | 国家分布依赖 GeoIP；未配置时国家显示「未知」 | 部署时配置 GeoIP 库（见「环境与资源」） |

## 环境与资源

| 资源 | 状态 | 需要在哪个迭代前 |
| --- | --- | --- |
| 本地 Go 1.23.12 / MySQL 8.4 / Redis 7.4 / Node 22 | 就绪 | - |
| Linux 测试服务器 | 未提供（本地已用模拟源站 + 自签证书完成全部可本地验证项，公网验收待服务器到位） | I6 公网验收 / I14 |
| 测试域名 ×2 与 DNS 控制权 | 未提供 | I6 公网验收 / I14 |
| 对象存储（R2/S3） | 未提供 | I6 |
| GeoLite2 账号 | 未提供（代码已接入，把 `GeoLite2-Country.mmdb` 放到 `backend-go/.local/` 即生效） | 日志国家字段 |
| SMTP / Telegram Bot | 未提供 | I11 |
| WordPress + WooCommerce 测试站 | 未提供 | I10 |
| V1 数据副本 | 不再需要（D4） | - |
