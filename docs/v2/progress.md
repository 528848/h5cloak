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
| I6 | 托管反向代理与证书 | 待验收 | 2026-11-04 | 2026-11-04 | `v2-i6` |
| I7 | 决策日志与归因管道 | 未开始 | | | |
| I8 | 服务端判定接口 v2 与 V1 兼容 | 未开始 | | | |
| I9 | 跨浏览器交接 | 未开始 | | | |
| I10 | WordPress/WooCommerce 插件 v2 与 JS SDK | 未开始 | | | |
| I11 | 看板、报表、告警、Webhook | 未开始 | | | |
| I12 | 总后台监管与系统配置 | 未开始 | | | |
| I13 | 迁移工具与影子对比 | 未开始 | | | |
| I14 | 部署与切换 | 未开始 | | | |
| I15 | 可选增强层 | 未开始 | | | |

## 当前迭代：I6 托管反向代理与证书

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

## 上一迭代记录：I5 配置快照与判定引擎

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

## 已知缺陷与技术债

| 编号 | 描述 | 计划 |
| --- | --- | --- |
| T1 | 便携版 Go/MySQL/Redis 复用自 `D:\pay-virtual\.runtime`，路径写在 `scripts/dev/common.ps1` | 后续换独立安装只改该文件 |
| T2 | 本地无 golangci-lint，门禁中跳过 | 安装后自动启用 |
| T3 | 本地无 ClickHouse，日志仓库先用 MySQL 分区实现 | I7 在测试服务器接 ClickHouse |
| T4 | 管理 API CORS 当前放行任意 Origin，生产由 Nginx 收紧 | I14 部署时处理 |
| T5 | 头像与审核静态包都存在本地目录，多节点部署需切对象存储 | I6 接入对象存储后切换 |
| T6 | 工具链固定 Go 1.23；`go get @latest` 会把 go.mod 拉到新版本，执行时需 `GOTOOLCHAIN=local` | 已在脚本中设置 |
| T7 | 快照用 JSON 而非压缩 MsgPack；大站点（> 10 万码）快照会膨胀 | I7 或压测后按技术方案 7 节改为哈希集合 + Redis 详情 |
| T8 | ~~网关在 I6 之前对站点请求返回判定 JSON 而非回源内容~~ | I6 已接入反向代理；`X-Cloak-Debug: decide` 保留 JSON 用于排障 |
| T10 | 证书存本地目录，多网关节点需共享（对象存储或 Redis 存储后端） | I14 多节点部署前处理 |
| T11 | 静态包、模板响应未做磁盘/内存缓存，大流量下每次读文件与渲染 | 压测后按需加 LRU |
| T12 | 引导条文案与链接延续只处理原始 HTML，脚本动态生成的链接需 JS SDK 补齐 | I10 |
| T9 | 广告 Referer 域名表与内置浏览器 UA 特征写死在代码里 | I12 移到 `cfg:global` 由总后台维护 |

## 环境与资源

| 资源 | 状态 | 需要在哪个迭代前 |
| --- | --- | --- |
| 本地 Go 1.23.12 / MySQL 8.4 / Redis 7.4 / Node 22 | 就绪 | - |
| Linux 测试服务器 | 未提供（本地已用模拟源站 + 自签证书完成全部可本地验证项，公网验收待服务器到位） | I6 公网验收 / I14 |
| 测试域名 ×2 与 DNS 控制权 | 未提供 | I6 公网验收 / I14 |
| 对象存储（R2/S3） | 未提供 | I6 |
| GeoLite2 账号 | 未提供 | I7 |
| SMTP / Telegram Bot | 未提供 | I11 |
| WordPress + WooCommerce 测试站 | 未提供 | I10 |
| V1 数据副本 | 未提供 | I13 |
