# cloak-sdk-php

斗篷（Cloak）服务端 SDK 的 **PHP 实现**，框架无关。与 Go SDK、网关共用同一套深度代理改写规则与 v2 签名，并通过 `sdk/testdata` 的黄金用例集对拍。

在你自己的服务器上，于网站入口调用判定接口决定「放行真实内容」还是「审核」；审核时**在本机深度代理审核站**（回源 → 把审核站域名改写成你的域名 → 输出）。访客流量不经过斗篷网关。

## 安装

```bash
composer require h5cloak/cloak-sdk-php
```

或直接引入 `src/` 下的类（PSR-4，命名空间 `Cloak\Sdk\`）。

## 一行接入

```php
require 'vendor/autoload.php';

$client = new \Cloak\Sdk\Client([
    'apiBase'      => 'https://gw.example.com', // 斗篷网关地址
    'secretKey'    => 'sk_xxx',                 // 服务端密钥，只放服务器
    'reviewOrigin' => 'https://review.com',     // 审核站地址
]);

$client->handle(); // 放行 → return 继续渲染真实内容；审核/拦截 → 直接输出并 exit
// 下面是你自己的真实页面
```

## 手动用法

```php
$d = $client->decide();                 // 调判定接口，返回 data 数组或 null
if ($d && in_array($d['content'] ?? '', ['real', 'bypass'], true)) {
    // 放行
} else {
    $client->proxyReview();             // 本机深度代理审核站
}
```

## 配置

| 键 | 说明 | 默认 |
| --- | --- | --- |
| `apiBase` | 网关地址 | 必填 |
| `secretKey` | 服务端密钥 `sk_...` | 必填 |
| `reviewOrigin` | 审核站地址 | 必填 |
| `visitorHost` | 对外域名 | 请求 Host |
| `reviewHost` | 回源 Host（源站用 IP 时） | `reviewOrigin` 主机 |
| `cookieName` | 访客 Cookie 名 | `_ck_vid` |
| `timeout` | 判定/回源超时（秒） | 3 |
| `failOpen` | 判定失败是否放行真实内容 | false（按审核兜底） |
| `skipPaths` / `skipSuffixes` | 跳过判定的路径/后缀 | 后台、登录、静态资源等 |
| `clientIpHeaders` | 真实 IP 头 | `HTTP_CF_CONNECTING_IP` 等 |
| `cacheDir` | 深度代理缓存目录，空则关闭 | 空 |
| `cacheTtl` | 缓存秒数，0 关闭 | 0 |

同一页面第一次回源并改写后写入 `cacheDir`，有效期内直接返回。键是方法、审核站主机、对外域名、协议、路径和去掉广告参数后的查询串。登录和购物车 Cookie 不进缓存。WordPress 插件默认开启 600 秒，并在设置页提供清除按钮。

## 自定义 HTTP 传输

默认用 ext-curl。可实现 `Cloak\Sdk\HttpTransport` 注入自己的客户端（如 WordPress 的 `wp_remote_*`）：

```php
$client = new \Cloak\Sdk\Client($cfg, new MyWpTransport());
```

WordPress 插件 `wordpress-plugin/cloak-attribution` 即复用本 SDK 的 `Rewriter` / `Signer`，并提供「深度代理」审核模式。

## 测试

```bash
php -n tests/run.php
```

跑共享黄金用例集（改写规则 + 签名向量 + Content-Type 判定）。
