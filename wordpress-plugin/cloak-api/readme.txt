=== Cloak API ===
Contributors: cloak-system
Tags: cloak, api, traffic, review page
Requires at least: 5.8
Tested up to: 6.7
Requires PHP: 7.4
Stable tag: 1.1.0
License: GPLv2 or later

接入斗篷系统 API 判定接口的 WordPress 插件。

== Description ==

Cloak API 插件帮助 WordPress 独立站在服务端调用斗篷判定接口，根据返回结果自动展示审核页（白页）或真实页（黑页）。

**支持的审核页 / 真实页方式：**

1. WordPress 站内页面
2. 外部链接（https）
3. 服务器 HTML 文件（URL 跳转）
4. 服务器 HTML 文件（PHP 直接输出，URL 不变）

**路径排除：**

* 支持按 URL 路径排除斗篷，如 `/blog/*`、`/privacy-policy`

**功能特点：**

* 后台可视化配置，无需改主题代码
* Api Token 仅保存在服务器
* 支持 wp-config.php 常量覆盖敏感配置
* 支持 Cloudflare 真实 IP 识别
* 内置 API 连接测试
* 接口失败时可配置默认展示审核页或真实页

== Installation ==

1. 将 `cloak-api` 文件夹上传到 `/wp-content/plugins/`
2. 在 WordPress 后台启用插件
3. 进入「设置 → API 斗篷」填写 API 地址、Token 和页面配置
4. 点击「测试 API 连接」确认可用后保存并启用

== Configuration ==

**推荐在 wp-config.php 中配置敏感信息：**

`define('CLOAK_API_URL', 'https://your-domain.com/api/open/decision');`
`define('CLOAK_API_TOKEN', 'ck_xxxxxxxx');`

== Frequently Asked Questions ==

= 为什么斗篷不生效？ =

请检查是否开启了全页缓存插件（WP Rocket、LiteSpeed 等），需要排除前台或按访客区分缓存。

= 外部审核页跳转失败？ =

外部链接必须使用 https，并在「允许的外部域名」中填写域名白名单。

== Changelog ==

= 1.1.0 =
* 真实页支持站内页面、外部链接、HTML 跳转、HTML 直接输出
* 新增排除路径规则（如 /blog/*）

= 1.0.0 =
* 首次发布
