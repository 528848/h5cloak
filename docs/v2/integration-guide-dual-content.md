# 同域双内容接入指南

同一条链接，带有效归因的访客看到真实内容，其余访客在相同地址看到审核内容，地址栏始终不变。

本指南以**服务端 SDK + 深度代理**为主线：判定由我们的网关完成（只传一次轻量 JSON），**审核内容的回源与改写在你自己的服务器里执行**，访客的页面和静态资源流量不经过我们的网关——带宽压力留在你这边，接入也更简单。没有服务器或不想改代码的，仍可用托管接入或 JS SDK。

## 判定结果只有两类

| 返回 | 含义 | 该输出 |
| --- | --- | --- |
| `content` 为 `real` 或 `bypass` | 放行 | 真实内容 |
| 其他（含 `review`、`block`、`not_found`） | 不放行 | 审核内容 / 403 / 404 |

接口失败、超时时一律先按审核处理，确认业务允许后再改放行。

**不要对被判定的页面做全页缓存。** WP Rocket、LiteSpeed、Nginx 的整页缓存会把第一个访客的页面发给所有人。被判定的响应必须加 `Cache-Control: no-store`，或把前台按访客 Cookie 分开缓存。

## 先选场景

| 你的情况 | 用哪一节 | 地址栏 | 改代码 | 访客流量 |
| --- | --- | --- | --- | --- |
| WordPress 站点 | 场景一 插件（深度代理） | 不变 | 不用，装插件 | 在你的服务器 |
| 自研 PHP 服务端 | 场景二 PHP SDK | 不变 | 装 SDK，入口加几行 | 在你的服务器 |
| 其他后端语言（Node / Java / Go / Python…） | 场景三 直连判定接口 | 不变 | 按契约实现 | 在你的服务器 |
| 已有网站、完全不想改代码，愿意让流量过网关 | 场景四 托管接入 | 不变 | 不用 | 经过网关 |
| 纯静态 HTML，两套内容可同时下载到浏览器 | 场景五 JS SDK | 不变 | 改页面 | 浏览器直连 |
| 审核页必须是另一个固定网址 | 场景六 跳转 | 会变 | 改成跳转 | — |

> **SDK 模式的核心（场景一、二、三）**：你在网站入口调用一次判定接口拿到 `content`；放行就渲染你自己的真实内容；审核就在**你的服务器内**把审核站整站代理过来——回源审核站、把页面里的审核站域名改写成你的域名、再在当前地址输出。我们的网关只参与那一次判定，不承载访客的页面与资源流量。

---

## 判定接口契约（场景一、二、三共用）

SDK 已封装好这一节；自研或排查时看这里。

### 请求

```
POST {网关地址}/api/open/v2/decision
Content-Type: application/json
```

签名请求头（v2）：

| 头 | 值 |
| --- | --- |
| `X-Cloak-Key` | 服务端密钥 `sk_` 的前 12 位 |
| `X-Cloak-Timestamp` | Unix 秒，与服务器相差 ≤ 300 秒 |
| `X-Cloak-Nonce` | 8–64 字节随机串（网关 5 分钟内去重） |
| `X-Cloak-Signature` | 见下方签名算法 |

请求体（JSON）：

```json
{
  "url": "https://你的域名/landing?code=ALICE01",
  "method": "GET",
  "visitorToken": "访客 Cookie _ck_vid 的值，没有传空",
  "ip": "访客真实 IP",
  "userAgent": "访客 UA",
  "referer": "来源地址",
  "acceptLanguage": "访客 Accept-Language"
}
```

`url` 必填，是访客正在打开的完整地址（归因码就在里面）。

### 签名算法（各语言一致）

```
msg = ts + "\n" + nonce + "\n" + 大写(method) + "\n" + path + "\n" + hex(sha256(body))
sig = hex(HMAC_SHA256(secret, msg))        // 小写 hex
```

`path` 是接口路径 `/api/open/v2/decision`，不含域名与查询串；`body` 是原始请求体字节。

### 返回

```json
{
  "code": 200,
  "msg": "ok",
  "data": {
    "content": "real",
    "attributed": true,
    "visitorToken": "eyJ…",
    "cookie": { "name": "_ck_vid", "value": "eyJ…", "maxAge": 2592000, "path": "/", "secure": true, "httpOnly": true, "sameSite": "Lax" },
    "handoffUrl": "https://你的域名/landing?code=ALICE01&_ck=…",
    "handoffBanner": { "enabled": true, "text": "在浏览器中打开，继续访问" },
    "renderHint": { "type": "html", "html": "…" }
  }
}
```

| 字段 | 用法 |
| --- | --- |
| `content` | `real` / `bypass` 放行；`review` 审核；`block` 403；`not_found` 404 |
| `cookie` | 放行、审核都要按它写访客 Cookie，回访免带码 |
| `visitorToken` | 同 `cookie.value`，下次请求放进 `visitorToken` |
| `renderHint` | 审核且审核内容为平台模板时，`type=html` 的 `html` 可直接输出，省去回源审核站 |
| `handoffUrl` / `handoffBanner` | 应用内浏览器里的引导数据。弹框不由接口输出，由客户站点上的 JS SDK 绘制，见场景五 |

- 探活：`GET {网关地址}/api/open/v2/ping`。
- 兼容 V1：`POST /api/open/decision` + `X-API-Key: sk_`，返回里带 `page=real/review`。新接入一律用上面的 v2 签名。

### 深度代理的域名改写规则（SDK 已内置）

审核内容回源后，要把正文里的**审核站域名 `oh` 改写成你的域名 `pub`**，覆盖以下形态（全程大小写不敏感）：

| 形态 | 示例 | 改写为 |
| --- | --- | --- |
| `https://oh` / `http://oh` | `https://review.com/x` | `{访客协议}://pub/x`（http 升级为访客协议） |
| `//oh` | `//review.com/a.css` | `//pub/a.css` |
| 转义斜杠 `https:\/\/oh`、`\/\/oh` | JS 里 `https:\/\/review.com` | `{访客协议}:\/\/pub` |
| 百分号编码 `https%3A%2F%2Foh`、`%2F%2Foh` | `?url=https%3A%2F%2Freview.com%2F` | `…pub…` |
| 裸主机名 `oh` | 页脚 `@2024 review.com.`、`title="review.com"`、JSON `"host":"review.com"` | `pub` |

边界规则（防误伤）：带前缀/编码形态匹配后主机名结尾必须是边界字符（`/ ? # : " ' 空白 < > 等`），否则不改；裸主机名两侧都要求边界，`cdn.review.com`（子域名）与 `review.comx`（更长主机）保持原样。只改写文本类型（HTML、CSS、JS、JSON、XML、SVG），图片/字体/二进制原样透传。`Location`、`Set-Cookie` 的审核站域名也一并改写。

---

## 场景一：WordPress 插件（深度代理）

装 `wordpress-plugin/cloak-attribution`（v2.3.0 起内置深度代理），不用写代码。不要和旧的 `cloak-api` 同时启用。

准备：

- 站点服务端密钥 `sk_`（站点配置页创建 / 重置，只显示一次）。
- 网关地址。
- 一套审核内容。真实页面就是你现在的网站，不用另做。

步骤：

1. 上传启用插件。设置 → Cloak 归因。
2. 密钥写进 `wp-config.php`，不要存数据库：

```php
define('CLOAK_API_BASE', 'https://网关地址');
define('CLOAK_SECRET', 'sk_xxxxxxxx');
```

3. 点「测试连接」。失败先看服务器能否访问网关、密钥是否 `sk_`。
4. 「网关不可用时」选「显示审核内容」。
5. 选审核内容的输出方式（见下表），保存。
6. 无痕打开首页看到审核内容，地址不变；带归因码链接打开看到原网站，地址仍不变。

审核内容的输出方式：

| 后台选项 | 什么时候用 | 你要准备 | 地址 |
| --- | --- | --- | --- |
| **深度代理整站（推荐）** | 审核页是另一个完整站点，要整站出现在本域名下、点击不跳走 | 审核站地址（可选回源 Host） | 不变。插件在本机回源并把审核站域名改写成本站域名 |
| 使用平台配置的审核模板 | 审核页在商户后台维护 | 后台选好模板 | 不变 |
| 渲染站内页面 | 审核页是本站一个页面 | 发布页面并选中 | 不变 |
| 输出服务器 HTML 文件 | 审核页是独立 html | 服务器上的绝对路径 | 不变 |
| 服务端拉取外部地址 | 只取单页、不需要整站改写 | 该地址的 https URL | 不变（**不改写域名**） |

> 「深度代理整站」与「服务端拉取外部地址」的区别：后者只把目标页抓回来原样输出，页内链接仍指向审核站；前者会把审核站域名整体改写成本站域名，站内点击继续停留在本域名。要做「审核站整站挂在本域名下」，选深度代理。

深度代理设置：填审核站地址，如 `https://review.example.com`。源站用 IP 部署、证书是另一个域名时，第二个框填「回源 Host」（证书/虚拟主机名），插件回源时带上它。

同一页面第一次回源后会缓存在 WordPress 服务器本机。设置页的「深度代理缓存」按天填写，保存时换算成秒，默认 1 天，`0` 为关闭；「清除深度代理缓存」立刻删掉全部已缓存页面。判定仍然每次都做，缓存里只有审核站页面，不会把审核页发给已经归因的访客。

排除路径每行一条（如 `/cart`、`/blog/*`），这些地址不判定、直接真实内容。后台、登录、REST、Cron 自动跳过。已登录用户是否跳过按店铺勾选。

插件深度代理会用到审核站回源，确保 **你的 WordPress 服务器能访问审核站**（审核站若有防火墙/IP 白名单，放行你的服务器出口 IP）。

### 应用内浏览器引导（JS SDK，插件自动引入）

Facebook、微信、抖音等内置浏览器里的「在系统浏览器中打开」弹框，由客户站点上的 JS SDK 绘制，插件不自己画弹框。

插件 **2.3.8** 起，访客在内置浏览器里看到真实页时，会在输出 HTML 的 `</body>` 前自动插入：

```html
<script>
  window.__CLOAK_HANDOFF__ = { "url": "https://你的域名/当前路径", "os": "android", "text": "在系统浏览器中打开", "enabled": true };
</script>
<script src="https://网关地址/_cloak/sdk.js" data-banner="preset" async></script>
```

不用在主题里再写一遍。要出现这个弹框，同时满足：

- `wp-config.php` 里定义了 `CLOAK_API_BASE`（脚本地址是 `{CLOAK_API_BASE}/_cloak/sdk.js`）。
- 插件设置里勾选「内置浏览器引导条」。
- 商户后台「跨浏览器接力」打开「打开提示条」。
- 当前访问的 User-Agent 被识别为内置浏览器。

已放行的访客，`url` 用接口返回的 `handoffUrl`（带接力令牌，换到系统浏览器后仍是同一次访问）。审核页不弹引导。弹出时机、关闭、重试、电脑和手机文案见 `docs/v2/prd-handoff-prompt.md`。点击「打开」由 SDK 处理：Android 交给系统默认浏览器，iPhone 尝试交给 Safari。

---

## 场景二：PHP SDK（自研 PHP，含深度代理）

适合自己的 PHP 程序（非 WordPress）。SDK 在入口判定并分流，审核时在本机深度代理审核站。

安装：

```bash
composer require h5cloak/cloak-sdk-php
```

或直接引入 `sdk/php/src` 下的类（PSR-4，命名空间 `Cloak\Sdk\`）。

### 一行接入

在网站入口（front controller）最前面：

```php
require 'vendor/autoload.php';

$client = new \Cloak\Sdk\Client([
    'apiBase'      => 'https://网关地址',
    'secretKey'    => getenv('CLOAK_SECRET'),   // sk_...，只放服务器
    'reviewOrigin' => 'https://review.example.com', // 审核站地址
]);

$client->handle();
// 放行：handle() 直接 return，下面继续渲染你的真实页面
// 审核 / 拦截：handle() 已输出并 exit
```

### 手动用法

要自己控制分流时：

```php
$d = $client->decide();              // 返回 data 数组，失败返回 null
if ($d && in_array($d['content'] ?? '', ['real', 'bypass'], true)) {
    // 放行：渲染真实内容（Cookie 已由 handle 处理；手动时自行按 $d['cookie'] 写）
} else {
    $client->proxyReview();          // 在本机深度代理审核站并输出
    exit;
}
```

### 配置

| 键 | 说明 | 默认 |
| --- | --- | --- |
| `apiBase` | 网关地址 | 必填 |
| `secretKey` | 服务端密钥 `sk_...` | 必填 |
| `reviewOrigin` | 审核站地址 | 必填 |
| `visitorHost` | 对外域名 | 请求 Host |
| `reviewHost` | 回源 Host（源站用 IP 时填） | `reviewOrigin` 主机 |
| `cookieName` | 访客 Cookie 名 | `_ck_vid` |
| `timeout` | 判定 / 回源超时（秒） | 3 |
| `failOpen` | 判定失败是否放行真实内容 | false（按审核兜底） |
| `skipPaths` / `skipSuffixes` | 跳过判定的路径 / 后缀 | 后台、登录、静态资源等 |
| `clientIpHeaders` | 取真实 IP 的头 | `HTTP_CF_CONNECTING_IP` 等 |

默认用 ext-curl 回源。要换成框架自己的 HTTP 客户端，实现 `Cloak\Sdk\HttpTransport` 注入：`new \Cloak\Sdk\Client($cfg, $myTransport)`。

### 注意

- `sk_` 只放环境变量 / 服务器配置，不要写进任何前端输出。
- SDK 的深度代理只代理你配置的 `reviewOrigin`，访客无法通过 path 控制回源目标（防 SSRF）。
- 确保**你的 PHP 服务器能访问审核站**；审核站有防火墙时放行你的服务器出口 IP。
- 若网关站点开了密钥 IP 白名单，把**你的服务器出口 IP**加进去，否则判定返回 `ip_not_allowed`。

---

## 场景三：其他后端语言（直连判定接口）

Node、Java、Go、Python 等暂无官方 SDK（规划中）。按上面「判定接口契约」自行实现，流程与 SDK 一致：

1. 入口过滤：后台、登录、REST、Cron、静态资源后缀不判定。
2. 用 v2 签名调 `POST /api/open/v2/decision`，带访客完整 URL、`visitorToken`、IP、UA、Referer。
3. 返回里有 `cookie` 就按名字 / 有效期 / 域名写入（放行、审核都写）。
4. `content` 为 `real` / `bypass`：输出真实内容，结束。
5. 否则审核：
   - `renderHint.type == html`：直接输出 `renderHint.html`（平台模板，最省）。
   - 否则在**你的服务器内**回源 `reviewOrigin`，按「深度代理的域名改写规则」把审核站域名改写成你的域名后，以 200 在当前地址输出，加 `Cache-Control: no-store`、`X-Robots-Tag: noindex, nofollow`。
6. 超时 / 错误：输出审核内容。

深度代理只代理你配置的审核站，别让访客的 path 决定回源主机（防 SSRF）。改写规则、边界、要改写的 Content-Type 见契约那一节；可参考 `sdk/go`、`sdk/php` 的实现与 `sdk/testdata` 的黄金用例集对拍。

---

## 场景四：托管接入（流量过网关，不改代码）

适合已有网站、完全不想写判定代码、接受访客流量经过网关的商户。网关站在域名和源站之间，同一条 URL 回源不同内容。

步骤：

1. 商户端 → 站点管理 → 新建站点，接入方式选「托管接入」（混合接入也可）。
2. 填真实源站地址（如 `https://origin.example.com`），审核内容三选一（平台模板 / 审核源站 / 静态包）。
3. 域名管理 → 添加「托管域名」，按提示把 CNAME / A 记录指到网关入口。
4. 点校验，状态变「已绑定」。
5. **在真实源站防火墙放行网关回源 IP**（见下）。
6. 无痕打开域名看到审核内容；带归因码打开看到真实源站，地址都不变。

### 真实源站必须放行网关回源 IP

网关取真实页时用一个**固定出口 IP** 连源站（见接入指南顶部「网关入口」处，或问运维）。源站的防火墙 / 安全软件常把「同一 IP 频繁规律回源」判成攻击自动封禁，表现为网关一握手就被重置（`connection reset by peer`），真实页取不回，带归因码的访客退回审核站。

- **加永久白名单，不是解一次封**（否则几分钟后被安全软件再次自动封）。
- **加在真正拦截的那一层**：主机防火墙 CSF（`csf -a 网关IP`，并从 `csf.deny` 删除）、Imunify360 / lfd、宝塔防火墙、Wordfence、源站前置的 Cloudflare 回源白名单；共享主机没权限就开工单让主机商永久加白。
- **确认已放行**：站点配置里真实源站「检测」可达；判定日志里带归因码请求 `origin_kind=real` 且 `fallback=0`。若 `origin_kind=review`、`fallback=1` 或源站「不可达 / connection reset by peer」，说明还没放行。

建议先放行、检测通过，再把主域名切到网关。自有域名（走场景二 / 三）不经过这条回源。

---

## 场景五：JS SDK

JS SDK 是跑在**客户站点页面里**的一段脚本，文件在网关：`https://网关地址/_cloak/sdk.js`。它做两件事：静态页上的双内容切换，以及应用内浏览器里的引导弹框。弹框只在浏览器里绘制，服务端不输出弹框 HTML。

引入位置都是**每个要生效的 HTML 页**，放在 `</body>` 之前，全站只放一次。放在 `<head>` 里时必须加 `async`，否则会挡住页面。不要放进 CSS / 图片请求，也不要每个组件各引一次。

### 5.1 只要引导弹框（页面已经由服务端判定过）

WordPress 插件、PHP SDK、其它后端深度代理都属于这类。判定已经在服务器完成，页面上**不要**再带 `data-key` 去请求一次判定，只让 SDK 画弹框并处理点击。

WordPress 插件 2.3.8 起会在真实页的 `</body>` 前自动插入，见场景一，不用手写。自研程序在放行后的 HTML 里，于 `</body>` 前插入：

```html
<script>
  window.__CLOAK_HANDOFF__ = {
    url: "https://你的域名/当前路径",
    os: "android",
    text: "在系统浏览器中打开",
    enabled: true
  };
</script>
<script src="https://网关地址/_cloak/sdk.js" data-banner="preset" async></script>
```

| 字段 | 填什么 |
| --- | --- |
| `url` | 已放行且接口给了 `handoffUrl` 时用它；否则用访客当前页的完整地址 |
| `os` | 接口 `data.inAppBrowser.os`，取 `ios` 或 `android` |
| `text` | 接口 `handoffBanner.text`，没有就用默认文案 |
| `enabled` | 商户后台关掉「打开提示条」时不要插入这段脚本 |

`data-banner="preset"` 表示只画弹框，不调用判定接口。访客点击「打开」后，SDK 再按系统跳转：Android 交给默认浏览器，iPhone 尝试交给 Safari。

### 5.2 静态页：浏览器里判定，并顺带画引导弹框

适合没有服务器、审核与真实内容可以同时出现在 HTML 源码里的落地页。脚本同样放在 `</body>` 前（或 `<head>` 末尾加 `async`）：

```html
<script src="https://网关地址/_cloak/sdk.js" data-key="pk_你的公钥" async></script>

<div data-cloak="review" hidden>…审核内容…</div>
<div data-cloak="real" hidden>…真实内容…</div>
```

1. `data-key` 填站点公钥 `pk_`（可出现在页面里，不是 `sk_`）。
2. 打开页面的域名必须已经绑在该站点上。SDK 会向 `POST /api/open/v2/js/capture` 上报当前网址。
3. 返回后只显示对应的 `data-cloak` 区块。

查看源代码能看到两套内容；要隐藏真实 HTML，用场景一 / 二 / 三 / 四。可选属性：`data-endpoint` 覆盖网关地址，`data-cookie` 改访客 Cookie 名，`data-banner="off"` 关闭引导弹框。

后台打开了「打开提示条」、且访客在内置浏览器里时，SDK 用返回的 `handoffUrl` 画同一个弹框。已放行的地址带接力令牌；未归因时打开当前页。监听判定结果：

```html
<script>
  window.addEventListener('cloak:decision', function (e) {
    console.log(e.detail.content, e.detail.handoffUrl);
  });
</script>
```

---

## 场景六：兼容跳转（地址会变）

只在审核页必须换成另一个固定网址时用。302 会让地址栏变成目标地址、慢网下闪一下，刷新 / 复制 / 分享 / 历史都变目标地址，对回访和广告落地页不友好——优先用前面的同域方案。

- WordPress 插件里审核页 / 真实页可各自选「站内页面（跳转）」「外部链接（跳转，仅 https，主机须在白名单）」「HTML 地址（跳转）」。
- 旧插件 `cloak-api` 暂不换程序时：API 指到 V2 `POST /api/open/decision`，`X-API-Key` 换成站点 `sk_`，`page` 字段仍在，`ck_` 不能用。

---

## 上线前检查

1. 密钥是 `sk_`，只在服务器配置 / 环境变量里，页面源码和 Git 里没有。
2. 无痕、无参数：审核内容；带有效分销码 / 点击参数：真实内容；同一浏览器再开无参数链接：仍真实内容。
3. 同域场景地址栏与输入一致、响应 200；只有跳转场景才有 302。
4. 全页缓存已排除或按 Cookie 分开。
5. `/wp-admin`、登录、支付回调在排除路径里，打开它们不会变审核页。
6. 拔掉网关或改错密钥：前台变审核内容，不会露出真实页。
7. 商户端「API 调用日志」能看到这次请求，`content` / `page` 与页面一致。
8. **SDK 模式（场景一 / 二 / 三）**：审核时地址栏不变、站内点击不跳到审核站域名；你的服务器能访问审核站；开了密钥 IP 白名单时已加入你的服务器出口 IP。
9. **托管接入（场景四）**：真实源站「检测」可达，带归因码请求 `origin_kind=real` 且 `fallback=0`；否则先查源站防火墙是否放行了网关回源 IP。
