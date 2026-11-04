=== Cloak Attribution ===
Contributors: cloaksystem
Tags: attribution, promo code, woocommerce, cloak
Requires at least: 5.8
Tested up to: 6.7
Requires PHP: 7.4
Stable tag: 2.0.0
License: GPLv2 or later

斗篷系统 v2 的 WordPress / WooCommerce 接入插件。同一地址：带有效归因参数的访客看到真实页面，其余访客看到审核内容。

== Description ==

* 服务端接入：每个前台请求调用斗篷网关 `POST /api/open/v2/decision`（HMAC-SHA256 签名），由网关判定真实 / 审核。
* 地址不变：审核内容以 200 在当前 URL 输出，可选平台模板、站内页面、本地 HTML 文件或外部地址内容。
* 访客令牌：按接口返回写入 Cookie，回访无参数也能识别。
* WooCommerce：可对全部商店页面或仅指定商品 / 分类（含对应购物车、结账）判定。
* 跨浏览器交接：在微信 / 抖音等内置浏览器中放行时，页脚注入「在浏览器中打开」引导条（需在站点开启交接）。
* 排除规则、跳过已登录用户、网关不可用时的降级策略、回访判定本地缓存、连接测试。
* 从旧版 cloak-api 插件自动迁移配置（密钥需换成 `sk_` 服务端密钥）。

== Installation ==

1. 上传 `cloak-attribution` 目录到 `wp-content/plugins/`，启用插件。
2. 商户后台 → 站点 → 密钥，复制 `sk_` 开头的服务端密钥。
3. WordPress 后台 → 设置 → Cloak 归因，填写网关地址与密钥，点击「测试连接」。
4. 选择审核内容输出方式，勾选「启用」，保存。

建议在 `wp-config.php` 中定义密钥而不是保存到数据库：

`define('CLOAK_API_BASE', 'https://entry.example.com');`
`define('CLOAK_SECRET', 'sk_xxxxxxxxxxxx...');`

== Frequently Asked Questions ==

= 与旧版 cloak-api 插件能否同时启用？ =
不建议。启用本插件后请停用旧版插件，配置会在本插件激活时自动迁移。

= 为什么测试连接提示「密钥前缀不存在」？ =
v2 使用站点的服务端密钥（`sk_` 开头），不是旧版的 `ck_` Key。若站点是旧版本创建的，需要在后台重置一次密钥。

= 使用了页面缓存插件怎么办？ =
判定按访客区分，整页缓存会让所有人看到同一结果。请对站点关闭整页缓存，或把需要判定的路径排除出缓存。

== Changelog ==

= 2.0.0 =
* 重写为 v2 签名接口，支持访客令牌 Cookie、交接引导条、WooCommerce 范围、判定缓存、连接测试、v1 配置迁移。
