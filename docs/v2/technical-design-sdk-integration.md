# 服务端 SDK（含深度代理）技术方案

配套产品文档：`docs/v2/prd-sdk-integration.md`。判定接口、签名、配额、日志全部复用现状；本方案新增的是「**深度代理在客户服务器内执行**」的 SDK 规范，以及多语言一致的改写规则。

## 1. 架构

```
访客 ──HTTP──▶ 客户服务器（集成我们的 SDK）
                   │  1. 入口中间件采集请求
                   │  2. 签名调用判定接口（小 JSON）
                   ▼
            我们的网关 /api/open/v2/decision ──▶ 返回 content=real/review/…
                   │
       ┌───────────┴───────────┐
   放行 │                        │ 审核
   客户自己的真实内容           SDK 在客户服务器内深度代理：
                               回源审核站 → 改写域名 → 输出
（访客的页面与静态资源流量全程在客户服务器，不经过我们）
```

与现状的差别：**深度代理从我们的网关挪到客户服务器**。我们只承载判定 JSON。审核站的回源与改写流量由客户服务器承担。

网关托管深度代理（`internal/proxy`）作为备选保留，逻辑不动。

## 2. 判定接口契约（复用，不改）

- `POST {apiBase}/api/open/v2/decision`
- 鉴权头（v2 签名）：
  - `X-Cloak-Key`：`sk_` 的前 12 位前缀
  - `X-Cloak-Timestamp`：Unix 秒（与服务器相差 ≤300s）
  - `X-Cloak-Nonce`：8–64 字节随机串（Redis 去重 5 分钟）
  - `X-Cloak-Signature`：见 §3
- 请求体（JSON）：`url`（必填，访客完整地址）、`method`、`visitorToken`（访客 Cookie 值）、`ip`、`userAgent`、`referer`、`acceptLanguage`。
- 返回 `data`：
  - `content`：`real` / `bypass` 放行；`review` 审核；`block` 403；`not_found` 404。
  - `cookie`：`{name,value,maxAge,domain,path,secure,httpOnly,sameSite}`，放行时种。
  - `visitorToken`：同 `cookie.value`。
  - `handoffUrl` / `handoffBanner`：内置浏览器引导（可选用）。
  - `renderHint`：审核且审核内容为平台模板时，`{type:"html",html}` 可直接输出，省去回源审核站。
- 探活：`GET {apiBase}/api/open/v2/ping`。

兼容：也支持 V1 `POST /api/open/decision` + `X-API-Key: sk_`，返回里带 `page=real/review`。新 SDK 一律用 v2 签名。

## 3. 签名算法（各语言必须一致）

```
msg = ts + "\n" + nonce + "\n" + UPPER(method) + "\n" + path + "\n" + hex(sha256(body))
sig = hex(HMAC_SHA256(secret, msg))     // 小写 hex
```

`path` 为接口路径（`/api/open/v2/decision`），不含域名与查询串；`body` 为原始请求体字节。参考实现：Go `internal/transport/http/openapi/auth.go` 的 `Sign`、PHP `cloak-attribution/includes/class-cloak-attr-signer.php`。

## 4. SDK 统一接口

各语言按生态封装，但对外概念一致：

```
Client(config)
  config: apiBase, secretKey, reviewOrigin, visitorHost?,
          timeout=3s, failMode=review, cacheTTL=0,
          skipPaths=[...], clientIPHeaders=[...]

decide(request) -> Decision
  Decision: content, cookie?, visitorToken?, handoffUrl?, renderHint?

handle(request, response, next)   // 中间件：判定 + 分流 + 深度代理（审核）/ 放行
proxyReview(request, response)    // 单独调用：在本机深度代理审核站
reportConversion(order)           // 可选
```

中间件 `handle` 的默认流程：

1. 命中 `skipPaths` 或非页面请求 → 直接 `next()`（不判定）。
2. `decide()`：失败按 `failMode`（默认 review）。
3. `content ∈ {real,bypass}` → 种 Cookie → `next()`（客户自己的内容）。
4. `content == review`：
   - `renderHint.type == html` → 直接输出该 HTML（平台模板，最省）。
   - 否则 → `proxyReview()` 深度代理 `reviewOrigin`。
5. `content == block/not_found` → 输出 403 / 404。

## 5. 深度代理（SDK 内，核心）

`proxyReview` 在客户服务器内完成：

### 5.1 回源

- 目标：`reviewOrigin` + 访客的 `path?query`，方法透传（GET/HEAD/POST…）。
- 请求头：带访客 UA、`Accept-Language`；**强制 `Accept-Encoding: identity`** 便于改写；删逐跳头；可带 `X-Forwarded-For`。
- 若 `reviewOrigin` 用 IP + 自定义 Host（源站证书场景），SDK 支持配置回源 Host 与 TLS SNI。

### 5.2 响应头处理

- 删 `Server` `X-Powered-By` `Via` 等源站指纹头与逐跳头。
- `Location` 指向审核站 → 改写成客户域名（协议用访客协议）。
- `Set-Cookie` 的 `Domain=审核站域名` → 去掉或换成客户域名。
- 加 `Cache-Control: no-store`；审核加 `X-Robots-Tag: noindex, nofollow`。

### 5.3 正文改写（规则必须与网关 `rewriteBodyHost` 一致）

仅对文本类型改写：`text/html`、`application/xhtml+xml`、`text/css`、`*javascript`、`*ecmascript`、`application/json`、`text/json`、`*+json`、`text/xml`、`application/xml`、`*+xml`、`image/svg+xml`。其它（图片、字体、二进制）原样透传。正文大小上限（默认 8MB）内才改写，超出原样输出。

把**审核站域名 `oh` → 客户域名 `pub`**，按下列形态逐一替换（大小写不敏感）：

| 形态 | 示例 | 替换为 |
| --- | --- | --- |
| `https://oh` / `http://oh` | `https://review.com/x` | `{scheme}://pub/x`（http 升级为访客协议） |
| `//oh` | `//review.com/a.css` | `//pub/a.css` |
| 转义斜杠 `https:\/\/oh`、`\/\/oh` | JS 里 `https:\/\/review.com` | `{scheme}:\/\/pub` |
| 百分号编码 `https%3A%2F%2Foh`、`%2F%2Foh` | `?url=https%3A%2F%2Freview.com%2F` | `...pub...` |
| 裸主机名 `oh` | 页脚 `@2024 review.com.`、`title="review.com"`、JSON `"host":"review.com"` | `pub` |

**边界规则（防误伤，务必实现）：**

- 带前缀/编码形态：匹配后，主机名**结尾**必须是边界字符之一 `/ ? # : " ' \` 空白 < > ( ) ; , & \ ] % =`，否则不替换（挡住 `review.comx`）。
- 裸主机名：**前**一个字符不能是字母/数字/连字符/点，**后**一个字符不能是字母/数字/连字符（允许点号作句末/根标签）。挡住 `cdn.review.com`（子域名保留）和 `review.companies`（更长主机）。
- `oh == pub`（同域）时跳过。
- 匹配全程大小写不敏感（`HTTP://Review.CC` 也要改）。

参考实现与测试：`internal/proxy/rewrite.go` 的 `rewriteBodyHost` / `replaceHostFold` / `replaceBareHost` / `hostBoundary`，用例见 `proxy_test.go` 的 `TestRewriteBodyHost`。各语言 SDK 需移植并自带等价单测（同一组输入输出）。

### 5.4 输出

- 透传状态码；写改写好的正文。
- 回源时向审核站请求 `gzip`，先解开再改写。改写后的明文写入本机缓存。
- 发给浏览器时，文本正文（HTML、CSS、JS、JSON、XML、SVG）大于 1KB 且访客接受 gzip，再用速度优先的 gzip 压一次。图片和字体不再压缩。
- HTML 的 `Cache-Control` 仍是 `no-store`，避免 CDN 或浏览器把审核页发给下一位访客。样式、脚本、图片、字体对所有访客相同，使用 `public, max-age=86400`。页面加速靠客户服务器自己的缓存，静态资源还可以被浏览器缓存。

### 5.5 回源缓存（各语言 SDK 同一套）

深度代理的页面、样式、脚本、图片第一次回源并完成域名改写后写入本机缓存。同一个键再次进入时直接读缓存返回，不再访问审核站。PHP SDK 已实现（`Cloak\Sdk\ResponseCache`）；Go / Node / Java / Python 按本节对齐，暂未实现。

**存什么。** 存改写之后的正文，外加状态码和 Content-Type。命中后不再回源，也不再改写。

**键。**

```text
sha256(METHOD \n originHost \n publicHost \n scheme \n path \n normalizedQuery)
```

- 主机名、方法、协议忽略大小写。`path` 保留原样，空路径记为 `/`。
- `normalizedQuery`：解析查询串，去掉广告点击参数（`utm_*`、`fbclid`、`gclid`、`gbraid`、`wbraid`、`msclkid`、`ttclid`、`twclid`、`mc_cid`、`mc_eid`、`yclid`、`igshid`、`srsltid`、`_ga`、`ref`），其余参数按名字排序后重新编码。`ver`、分页、筛选留下。
- 访客域名不同则键不同，因为正文里的域名已经改写成该域名。

**什么时候写。** 仅 `GET` 且状态码 `200`，正文非空且不超过 2MB。`POST`、其它状态码、超大正文不写。

**什么时候不读也不写。** 请求 Cookie 里有 `wordpress_logged_in_`、`wp_woocommerce_session_`、`comment_author_`、`wp-postpass_`。这些页面因人而异。

**TTL。** 内部一律用秒。`0` 关闭。WordPress 插件设置页按天填写，保存时乘以 86400；默认 1 天，最长 30 天。设置页有「清除深度代理缓存」按钮，一次删掉全部缓存文件。

**存放。** 客户服务器本地目录，按键的前两位分目录。文件不能通过网址直接访问（目录拒绝外部请求）。不写入数据库，避免整页 HTML 撑爆选项表。

**和下一位访客的关系。** 缓存只省审核站回源。判定接口每次仍调用。浏览器和页面缓存插件继续收到 `no-store`，真实访客不会拿到审核缓存。

## 6. 跳过与边界

- 默认跳过：后台路径、登录、REST、Cron、静态资源后缀（`.css .js .png .jpg .svg .ico .woff...`）、健康检查。
- 入口只对「页面级 GET/HEAD」判定；资源请求不判定、不代理（本就由客户服务器或其 CDN 处理）。
- 判定失败一律 `failMode`（默认 review），不泄露真实内容。

## 7. 各语言落地要点

| 语言 | 挂载点 | HTTP 客户端 | 字节级改写 |
| --- | --- | --- | --- |
| PHP | WordPress `template_redirect` / 入口 front controller | `curl` / `wp_remote_*` | 按字节 `str_ireplace` + 边界校验，避免 `mb_` 破坏二进制 |
| Node | Express/Koa 中间件、原生 `http` | `undici` / `fetch` | `Buffer` 操作 |
| Go | `http.Handler` 包装 / 中间件 | `net/http` | 直接复用网关同款函数 |
| Java | Servlet `Filter` / Spring `HandlerInterceptor` | `HttpClient` | `byte[]` 扫描替换 |
| Python | WSGI / ASGI 中间件 | `httpx` / `requests` | `bytes` 操作 |

共性：配置从环境变量 / 配置文件读；密钥不落前端；提供「一行接入」的中间件与「手动调用」两种用法；自带 `rewriteBodyHost` 等价实现 + 单测。

## 8. 安全

- `sk_` 仅在服务器；SDK 不把密钥写进任何对客户端输出。
- v2 签名：时间戳 + nonce 防重放；站点可设密钥 IP 白名单（客户服务器出口 IP 需在白名单，否则 `ip_not_allowed`）。
- 深度代理只代理配置的 `reviewOrigin`，不做任意 URL 代理（防 SSRF）：目标主机固定为审核站，访客不能借 path 控制目标主机。
- 真实 IP 从配置的可信头取（如 `CF-Connecting-IP`）。

## 9. 发布与文档

- 各语言 SDK 独立仓库 / 包；后台「开发者中心」提供下载与各语言最小示例。
- `integration-guide-dual-content.md` 的「场景三 自研服务端」升级为「服务端 SDK + 深度代理」，并标注各语言包名。
- 版本与判定接口解耦：接口不变，SDK 可独立升级。

## 10. 实施拆分（建议）

1. ✅ 抽出语言无关的「改写规则 + 判定契约 + 签名」说明与**黄金测试用例集**（`sdk/testdata/rewrite-golden.json` 改写用例、`sdk/testdata/sign-golden.json` 签名已知答案向量，供各语言对拍）。
2. ✅ Go SDK（`sdk/go`，参考实现 + 对拍基准）：`RewriteBodyHost` 移植自 `rewrite.go`；`Client.Decide` 做 v2 签名判定；`Client.ProxyReview` 在本机深度代理；`Client.Handle` 一行中间件。单测跑黄金用例集 + 签名对拍 + 代理改写，全绿。
3. ✅ PHP SDK（`sdk/php`，框架无关，`Cloak\Sdk\{Client,Rewriter,Signer,HttpTransport,CurlTransport}`）+ 升级 WordPress 插件：插件 `includes/sdk/` 内置并继承 SDK 的 `Rewriter`/`Signer`，`Cloak_Attr_Signer` 委托给 SDK，新增审核模式「深度代理」（`Cloak_Attr_Deep_Proxy` 用 `wp_remote_*` + `Rewriter` 回源改写），版本升至 2.3.0。单测 `php tests/run.php` 跑黄金用例集 + 签名向量，全绿。
4. Node SDK。
5. Java、Python SDK。
6. 后台开发者中心补下载与示例；指南升级。

> 落地位置：`sdk/go`（独立 module `github.com/h5cloak/cloak-sdk-go`，仅标准库，可单独发包），`sdk/testdata` 为跨语言共享资产。

## 11. 验收

对齐 PRD §8：地址栏不变、链接不跳走、各形态域名改写且边界正确、失败兜底、访客流量不经过我们、各语言最小示例可跑通。各 SDK 的改写单测必须通过统一黄金用例集。
