# 斗篷系统 Debian + 宝塔面板部署文档

本文档面向 **Debian 11 / 12** 服务器，使用 **宝塔 Linux 面板（aaPanel / 宝塔）** 部署斗篷系统。

下文域名以示例代替，请替换为你的真实域名：

| 示例域名 | 用途 |
|----------|------|
| `cloak.example.com` | 总后台前端（`dist/admin`） |
| `merchant.cloak.example.com` | 商户端前端（`dist/merchant`） |
| `api.cloak.example.com` | 管理 API + 开放接口 |
| `entry.cloak.example.com` | 斗篷流量入口（商户 CNAME 目标） |

---

## 1. 部署架构

```text
                    ┌─────────────────────────┐
  平台管理员  ──────►│ cloak.example.com       │  总后台静态站
                    └─────────────────────────┘
                    ┌─────────────────────────┐
  商户用户    ──────►│ merchant.cloak.example.com │  商户端静态站
                    └─────────────────────────┘

                    ┌─────────────────────────┐
  管理后台 / 插件  ─►│ api.cloak.example.com   │  Laravel /api/*
                    └─────────────────────────┘

  商户业务域名 CNAME │
        └───────────►│ entry.cloak.example.com │  Laravel 流量入口 /*
                    └─────────────────────────┘
```

**开放 API 地址（WordPress 插件等）：**

```http
POST https://api.cloak.example.com/api/open/decision
X-API-Key: ck_xxxxxxxx
```

---

## 2. 服务器要求

| 项目 | 建议 |
|------|------|
| 系统 | Debian 11 / 12（64 位） |
| 配置 | 2 核 4G 起，生产建议 4 核 8G |
| 面板 | 宝塔 Linux 面板 7.x / 8.x |
| PHP | **8.2**（必须） |
| MySQL | **5.7 / 8.0 / MariaDB 10** |
| Nginx | 1.20+ |

---

## 3. 安装宝塔面板（Debian）

以 **root** 登录服务器 SSH，执行（以宝塔官网最新脚本为准）：

```bash
# Debian 12 示例
wget -O install.sh https://download.bt.cn/install/install-ubuntu_6.0.sh && sudo bash install.sh ed8484bec
```

> 宝塔安装脚本通常同时支持 Debian / Ubuntu，若提示不支持当前系统，请到 [宝塔官网](https://www.bt.cn/new/download.html) 查看 Debian 专用安装命令。

安装完成后记录：

- 面板地址（如 `https://IP:8888/xxxxxxxx`）
- 用户名与密码

**安全设置（面板 → 安全）：**

- 修改面板默认端口
- 放行端口：`80`、`443`、`8888`（或你的面板端口）
- 开启 SSH 密钥登录（推荐）

---

## 4. 宝塔软件安装

进入 **软件商店**，安装：

| 软件 | 版本要求 | 说明 |
|------|----------|------|
| **Nginx** | 任意稳定版 | Web 服务器 |
| **MySQL** | 5.7+ 或 8.0 | 数据库 |
| **PHP** | **8.2** | 后端运行环境 |
| **phpMyAdmin** | 可选 | 方便管理数据库 |

### 4.1 PHP 8.2 扩展

**软件商店 → PHP 8.2 → 设置 → 安装扩展**，确保已安装：

- `fileinfo`（必装，Laravel 依赖）
- `opcache`（推荐）
- `redis`（可选）

**设置 → 禁用函数**，从禁用列表中**删除**以下函数（Laravel / Composer 需要）：

```
putenv
proc_open
proc_get_status
symlink
pcntl_signal
pcntl_alarm
```

> 若 `composer install` 或 `artisan` 报 `putenv` / `proc_open` 相关错误，几乎都是禁用函数未放开。

### 4.2 调整 PHP 配置（可选）

**PHP 8.2 → 设置 → 配置修改**：

```ini
upload_max_filesize = 20M
post_max_size = 20M
max_execution_time = 300
memory_limit = 256M
```

---

## 5. 上传项目代码

推荐将代码统一放在 `/www/cloak/`，与宝塔站点目录分离，便于更新。

### 5.1 目录规划

```text
/www/cloak/
├── backend/          # Laravel 后端源码
├── frontend/         # Vue 前端源码（用于构建）
└── www/
    └── admin/        # 前端构建产物（dist 内容）
```

### 5.2 上传方式（任选其一）

**方式 A：Git（推荐）**

```bash
mkdir -p /www/cloak
cd /www/cloak
git clone <你的仓库地址> .
```

**方式 B：宝塔文件管理器**

1. 面板 → **文件** → 进入 `/www` → 新建目录 `cloak`
2. 将本地 `backend`、`frontend` 文件夹压缩为 zip 上传并解压

**方式 C：SFTP**

用 FileZilla 等工具上传到 `/www/cloak/`

### 5.3 安装 Composer

SSH 执行：

```bash
cd /www/cloak/backend

# 若未安装 Composer
curl -sS https://getcomposer.org/installer | php
mv composer.phar /usr/local/bin/composer
chmod +x /usr/local/bin/composer

composer install --no-dev --optimize-autoloader
```

---

## 6. 创建数据库

**宝塔 → 数据库 → 添加数据库**：

| 字段 | 示例值 |
|------|--------|
| 数据库名 | `cloak` |
| 用户名 | `cloak` |
| 密码 | 强密码（自行生成） |
| 访问权限 | 本地服务器 |

记下数据库名、用户名、密码，后面写入 `.env`。

---

## 7. 配置后端环境变量

SSH 执行：

```bash
cd /www/cloak/backend
cp .env.example .env
nano .env
```

按实际域名修改（**重点字段**）：

```ini
APP_NAME="Cloak"
APP_ENV=production
APP_DEBUG=false
APP_URL=https://api.cloak.example.com
MERCHANT_PORTAL_URL=https://merchant.cloak.example.com

DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=cloak
DB_USERNAME=cloak
DB_PASSWORD=你的数据库密码

LOG_LEVEL=warning

CLOAK_ENTRY_HOST=entry.cloak.example.com

SANCTUM_STATEFUL_DOMAINS=cloak.example.com,merchant.cloak.example.com,api.cloak.example.com
```

初始化：

```bash
php artisan key:generate
php artisan migrate --force
php artisan db:seed --class=RbacSeeder --force
php artisan config:cache
php artisan route:cache
```

默认账号（**上线后立即修改**）：

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 平台管理员 | admin | admin123 |
| 商户管理员 | merchant | merchant123 |

### 7.1 目录权限

```bash
cd /www/cloak/backend
chown -R www:www storage bootstrap/cache
chmod -R 775 storage bootstrap/cache
```

> 宝塔默认 Web 用户多为 `www`，若报错可执行 `ps aux | grep php-fpm` 确认实际用户。

---

## 8. 构建前端

### 8.1 安装 Node.js

**方式 A：SSH 安装 Node 20**

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs
node -v
```

**方式 B：本机构建后上传**

在 Windows / Mac 本地：

```bash
cd frontend
# 修改 .env.production 里的 VITE_API_URL
npm ci
npm run build:admin
npm run build:merchant
```

将 `dist/admin` 上传到 `/www/cloak/www/admin/`，将 `dist/merchant` 上传到 `/www/cloak/www/merchant/`。

### 8.2 服务器构建

```bash
cd /www/cloak/frontend
nano .env.production
```

```ini
VITE_APP_NAME=斗篷系统
VITE_API_URL=https://api.cloak.example.com/api
VITE_API_PROXY_URL=
```

```bash
npm ci
npm run build:admin
npm run build:merchant

mkdir -p /www/cloak/www/admin /www/cloak/www/merchant
rsync -av --delete dist/admin/ /www/cloak/www/admin/
rsync -av --delete dist/merchant/ /www/cloak/www/merchant/
chown -R www:www /www/cloak/www/admin /www/cloak/www/merchant
```

---

## 9. 宝塔创建站点

### 9.1 站点一：总后台 `cloak.example.com`

**网站 → 添加站点**：

| 配置项 | 值 |
|--------|-----|
| 域名 | `cloak.example.com` |
| 根目录 | `/www/cloak/www/admin` |
| FTP | 不创建 |
| 数据库 | 不创建 |
| PHP 版本 | **纯静态**（选「静态」或 PHP 选「纯静态」） |

**设置 → 伪静态**，填入：

```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

**设置 → SSL → Let's Encrypt** → 申请证书 → 开启 **强制 HTTPS**。

---

### 9.2 站点二：商户端 `merchant.cloak.example.com`

**网站 → 添加站点**：

| 配置项 | 值 |
|--------|-----|
| 域名 | `merchant.cloak.example.com` |
| 根目录 | `/www/cloak/www/merchant` |
| PHP 版本 | **纯静态** |

伪静态与总后台相同：

```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

申请 SSL 并开启强制 HTTPS。

---

### 9.3 站点三：管理 API `api.cloak.example.com`

**网站 → 添加站点**：

| 配置项 | 值 |
|--------|-----|
| 域名 | `api.cloak.example.com` |
| 根目录 | `/www/cloak/backend` |
| PHP 版本 | **PHP 8.2** |

创建后进入站点 **设置**：

1. **网站目录 → 运行目录** → 选择 `/public` → 保存  
2. **防跨站攻击(open_basedir)** → **关闭**（Laravel 需访问上级 `storage` 等目录）  
3. **伪静态** → 选择 **laravel5** 规则（或粘贴下方内容）：

```nginx
location / {
    try_files $uri $uri/ /index.php?$query_string;
}
```

4. **SSL** → 申请 Let's Encrypt → 强制 HTTPS

**配置文件补充（可选，用于反向代理真实 IP）**

站点 **设置 → 配置文件**，在 `server { }` 内、`location` 之前添加：

```nginx
    # 若套了 Cloudflare，可信任 CF 来源 IP
    set_real_ip_from 103.21.244.0/22;
    set_real_ip_from 103.22.200.0/22;
    set_real_ip_from 103.31.4.0/22;
    set_real_ip_from 104.16.0.0/13;
    set_real_ip_from 104.24.0.0/14;
    set_real_ip_from 108.162.192.0/18;
    set_real_ip_from 131.0.72.0/22;
    set_real_ip_from 141.101.64.0/18;
    set_real_ip_from 162.158.0.0/15;
    set_real_ip_from 172.64.0.0/13;
    set_real_ip_from 173.245.48.0/20;
    set_real_ip_from 188.114.96.0/20;
    set_real_ip_from 190.93.240.0/20;
    set_real_ip_from 197.234.240.0/22;
    set_real_ip_from 198.41.128.0/17;
    set_real_ip_from 2400:cb00::/32;
    set_real_ip_from 2606:4700::/32;
    set_real_ip_from 2803:f800::/32;
    set_real_ip_from 2405:b500::/32;
    set_real_ip_from 2405:8100::/32;
    set_real_ip_from 2c0f:f248::/32;
    set_real_ip_from 2a06:98c0::/29;
    real_ip_header CF-Connecting-IP;
```

> 未使用 Cloudflare 可跳过此段。

---

### 9.4 站点四：斗篷流量入口 `entry.cloak.example.com`

与 API 站点配置**基本相同**，区别仅域名不同：

| 配置项 | 值 |
|--------|-----|
| 域名 | `entry.cloak.example.com` |
| 根目录 | `/www/cloak/backend` |
| 运行目录 | `/public` |
| PHP | 8.2 |
| 防跨站 | 关闭 |
| 伪静态 | laravel5 |
| SSL | 申请并强制 HTTPS |

> API 与 Entry 共用同一份 Laravel 代码，Nginx 按域名分流到不同 `server` 块，无需复制代码。

---

## 10. 域名 DNS 解析

在域名服务商处添加：

| 主机记录 | 类型 | 记录值 |
|----------|------|--------|
| `cloak` | A | 服务器公网 IP |
| `api.cloak` | A | 服务器公网 IP |
| `entry.cloak` | A | 服务器公网 IP |

或使用宝塔 **DNS 解析** 插件统一管理。

解析生效后，在宝塔分别为三个站点申请 SSL。

---

## 11. 部署验证

### 11.1 API 健康检查

```bash
curl -I https://api.cloak.example.com/up
```

期望：`HTTP/2 200`

### 11.2 登录管理后台

1. 访问 `https://cloak.example.com`
2. 使用 `admin / admin123` 登录
3. **立即修改密码**

### 11.3 测试开放接口

在商户后台创建 API Key 后：

```bash
curl -X POST "https://api.cloak.example.com/api/open/decision" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ck_你的Token" \
  -d '{"ip":"1.2.3.4","userAgent":"Mozilla/5.0","referer":"https://example.com"}'
```

### 11.4 测试流量入口

1. 商户后台添加域名、斗篷策略
2. 将测试域名 **CNAME** 到 `entry.cloak.example.com`
3. 访问测试域名，确认跳转正确

---

## 12. WordPress 插件对接

```php
define('CLOAK_API_URL', 'https://api.cloak.example.com/api/open/decision');
define('CLOAK_API_TOKEN', 'ck_你的Token');
```

插件包路径：`wordpress-plugin/cloak-api.zip`

---

## 13. 日常更新

### 后端更新

```bash
cd /www/cloak/backend
git pull
composer install --no-dev --optimize-autoloader
php artisan migrate --force
php artisan config:cache
php artisan route:cache

# 宝塔 → PHP 8.2 → 服务 → 重载
```

### 前端更新

```bash
cd /www/cloak/frontend
git pull
npm ci
npm run build:admin
npm run build:merchant
rsync -av --delete dist/admin/ /www/cloak/www/admin/
rsync -av --delete dist/merchant/ /www/cloak/www/merchant/
```

---

## 14. 宝塔常见问题

### Q1：`composer install` 或 `artisan` 报错 `putenv() has been disabled`

→ **PHP 8.2 → 设置 → 禁用函数**，删除 `putenv`、`proc_open` 等（见第 4.1 节）。

### Q2：接口 500，`storage/logs/laravel.log` 报权限错误

```bash
chown -R www:www /www/cloak/backend/storage /www/cloak/backend/bootstrap/cache
chmod -R 775 /www/cloak/backend/storage /www/cloak/backend/bootstrap/cache
```

### Q3：前端能开，登录报「后端接口连接异常」

- 检查 `.env.production` 中 `VITE_API_URL` 是否为 `https://api.cloak.example.com/api`
- 浏览器 F12 → Network，确认请求地址正确
- 确认 API 站点 SSL 有效、运行目录为 `/public`

### Q4：登录后立刻退出

- `.env` 中 `SANCTUM_STATEFUL_DOMAINS` 需包含 `cloak.example.com`
- 前后端必须都用 HTTPS

### Q5：`open_basedir` 限制报错

→ API / Entry 站点 **关闭防跨站攻击**，或把 `/www/cloak/backend/` 加入白名单。

### Q6：地区判定不准

- 流量入口建议套 **Cloudflare**，系统读取 `CF-IPCountry` 头
- 本系统不内置 IP 地理库

### Q7：`artisan migrate` 在 SSH 正常，面板终端失败

→ 确认面板终端使用的 PHP 版本为 8.2：

```bash
/www/server/php/82/bin/php artisan migrate --force
```

---

## 15. 安全 checklist（上线前）

- [ ] `APP_DEBUG=false`
- [ ] 修改 admin / merchant 默认密码
- [ ] 四个站点均已开启强制 HTTPS
- [ ] 数据库禁止远程访问（宝塔默认仅本地）
- [ ] 面板修改默认端口，限制面板访问 IP（可选）
- [ ] 宝塔 → 计划任务 → 添加数据库定期备份

---

## 16. 目录与配置速查

| 路径 | 说明 |
|------|------|
| `/www/cloak/backend/` | Laravel 源码 |
| `/www/cloak/backend/public/` | API / Entry 网站运行目录 |
| `/www/cloak/www/admin/` | 总后台静态文件 |
| `/www/cloak/www/merchant/` | 商户端静态文件 |
| `/www/cloak/backend/.env` | 后端环境配置 |
| `/www/cloak/frontend/.env.production` | 前端构建配置 |
| `/www/cloak/backend/storage/logs/laravel.log` | 后端错误日志 |

| 宝塔站点 | 根目录 | 运行目录 |
|----------|--------|----------|
| cloak.example.com | `/www/cloak/www/admin` | /（根） |
| merchant.cloak.example.com | `/www/cloak/www/merchant` | /（根） |
| api.cloak.example.com | `/www/cloak/backend` | `/public` |
| entry.cloak.example.com | `/www/cloak/backend` | `/public` |

---

## 17. 单域名简化方案（仅测试）

若暂时只有一个域名 `cloak.example.com`，可在宝塔为该站点添加**反向代理**：

| 代理目录 | 目标 URL |
|----------|----------|
| `/api` | `http://127.0.0.1`（同服务器 API 站点）或单独配置 PHP 站点 |

前端 `.env.production`：

```ini
VITE_API_URL=https://cloak.example.com/api
```

**流量入口 `entry.` 子域名仍建议单独配置**，否则商户无法 CNAME。

更完整的 Nginx 手工配置见：[deploy-linux.md](./deploy-linux.md)
