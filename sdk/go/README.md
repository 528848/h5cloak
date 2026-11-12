# cloak-sdk-go

斗篷（Cloak）服务端 SDK 的 **Go 参考实现**，也是其它语言 SDK 的**对拍基准**。

在你自己的服务器上，于网站入口调用判定接口决定「放行真实内容」还是「审核」；审核时**在本机深度代理审核站**（抓取 → 把审核站域名改写成你的域名 → 输出）。访客的页面与静态资源流量全程留在你的服务器，不经过斗篷网关——网关只承载一次轻量判定 JSON。

## 安装

```bash
go get github.com/h5cloak/cloak-sdk-go
```

## 一行接入

```go
c := cloak.New(cloak.Config{
    APIBase:      "https://gw.example.com", // 斗篷网关地址
    SecretKey:    "sk_xxx",                 // 服务端密钥，只放服务器
    ReviewOrigin: "https://review.com",     // 审核站地址
})

http.Handle("/", c.Handle(myRealHandler)) // 放行走 myRealHandler，审核自动深度代理
```

## 手动用法

```go
d, err := c.Decide(r.Context(), r)
if err == nil && d.Pass() {
    // 放行：种 Cookie 后展示真实内容
}
// 审核：c.ProxyReview(w, r) 在本机代理审核站
```

## 配置

| 字段 | 说明 | 默认 |
| --- | --- | --- |
| `APIBase` | 网关地址 | 必填 |
| `SecretKey` | 服务端密钥 `sk_...` | 必填 |
| `ReviewOrigin` | 审核站地址 | 必填 |
| `VisitorHost` | 对外域名 | 请求 Host |
| `ReviewHost` | 回源 Host（源站用 IP + 自定义 Host 时） | `ReviewOrigin` 主机 |
| `CookieName` | 访客 Cookie 名 | `_ck_vid` |
| `Timeout` | 判定/回源超时 | 3s |
| `FailOpen` | 判定失败是否放行真实内容 | false（按审核兜底） |
| `SkipPaths` / `SkipSuffixes` | 跳过判定的路径/后缀 | 后台、登录、静态资源等 |
| `ClientIPHeaders` | 真实 IP 头 | `CF-Connecting-IP` / `X-Real-IP` / `X-Forwarded-For` |

## 签名

v2 HMAC-SHA256，与网关 `internal/transport/http/openapi/auth.go` 的 `Sign` 一致：

```
msg = ts + "\n" + nonce + "\n" + UPPER(method) + "\n" + path + "\n" + hex(sha256(body))
sig = hex(HMAC_SHA256(secret, msg))
```

## 深度代理改写规则

`RewriteBodyHost` 把审核站域名改写成访客域名，覆盖 `https://`/`//`/转义斜杠/百分号编码/裸主机名五种形态，并有边界规则防止误伤子域名与更长主机。规则与网关 `rewriteBodyHost` 完全一致，并通过共享黄金用例集 `../testdata/rewrite-golden.json` 对拍——**其它语言 SDK 必须对同一文件产出一致结果**。

## 测试

```bash
go test ./...
```
