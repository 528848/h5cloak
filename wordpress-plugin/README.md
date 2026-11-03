# Cloak API - WordPress 插件

将 `cloak-api` 目录上传到 WordPress 的 `wp-content/plugins/`，在后台启用即可。

## 安装

1. 复制本目录 `cloak-api` 到 `wp-content/plugins/`
2. WordPress 后台 → 插件 → 启用「Cloak API」
3. 设置 → API 斗篷 → 填写配置并保存

或打包为 zip 后通过「插件 → 安装插件 → 上传」安装：

```powershell
Compress-Archive -Path cloak-api -DestinationPath cloak-api.zip -Force
```

## 后台配置项

| 配置项 | 说明 |
|--------|------|
| 启用斗篷 | 总开关 |
| API 地址 | 如 `https://域名/api/open/decision` |
| API Token | 斗篷后台创建的 `ck_` 前缀 Token |
| 审核页方式 | 站内页面 / 外部链接 / HTML 跳转 / HTML 输出 |
| 真实页方式 | 同上四种模式 |
| 排除路径 | 如 `/blog/*`、`/shop/checkout`，匹配时跳过斗篷 |
| 接口失败时 | 默认展示审核页或真实页 |

## wp-config.php 可选覆盖

```php
define('CLOAK_API_URL', 'https://your-domain.com/api/open/decision');
define('CLOAK_API_TOKEN', 'ck_xxxxxxxx');
```

## 文件结构

```
cloak-api/
├── cloak-api.php          # 插件入口
├── includes/              # 核心逻辑
├── admin/                 # 后台设置页
├── readme.txt
└── uninstall.php
```
