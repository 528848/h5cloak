# 斗篷系统 Linux 服务器部署文档

本文档适用于在 Linux 服务器（推荐 **Ubuntu 22.04 / 24.04**）上部署斗篷系统生产环境。

> **Debian + 宝塔面板部署**请直接阅读专用文档：**[deploy-debian-bt.md](./deploy-debian-bt.md)**

---

## 1. 架构说明

系统由四部分组成。总后台和商户端是两套独立构建、独立部署的前端，不再共用同一个站点：

| 组件 | 说明 | 推荐域名示例 |
|------|------|-------------|
| **总后台前端** | `npm run build:admin`，产物 `frontend/dist/admin` | `cloak.example.com` |
| **商户端前端** | `npm run build:merchant`，产物 `frontend/dist/merchant` | `merchant.cloak.example.com` |
| **管理 API 后端** | Laravel 12，提供登录、商户管理、API Key、开放接口等 | `api.cloak.example.com` |
| **斗篷流量入口** | Laravel 同一套代码的 Web 路由，处理商户域名 CNAME 进来的访客流量 | `entry.cloak.example.com` |

开放接口（WordPress 插件、第三方调用）地址：

```http
POST https://api.cloak.example.com/api/open/decision
Header: X-API-Key: ck_xxxxxxxx
```

> **说明：** 管理 API 与斗篷流量入口可共用同一 Laravel 代码目录，通过 Nginx 按域名分流；也可在同一域名下用路径区分（流量入口建议独立子域名，便于商户 CNAME）。

### 推荐目录结构

```text
/opt/cloak/
├── backend/          # Laravel 源码（git clone）
├── frontend/         # Vue 源码（构建用）
└── www/
    ├── admin/        # 总后台 dist/admin
    └── merchant/     # 商户端 dist/merchant
```

---

## 2. 服务器配置要求

### 最低配置（测试 / 小规模）

| 项目 | 要求 |
|------|------|
| CPU | 2 核 |
| 内存 | 4 GB |
| 磁盘 | 40 GB SSD |
| 系统 | Ubuntu 22.04+ / Debian 12+ / CentOS 8+ |

### 推荐配置（生产）

| 项目 | 要求 |
|------|------|
| CPU | 4 核+ |
| 内存 | 8 GB+ |
| 磁盘 | 100 GB SSD |
| 带宽 | 按访问量，斗篷入口建议独立节点或 CDN 前置 |

### 软件版本

| 软件 | 版本 |
|------|------|
| PHP | **8.2+**（需扩展见下） |
| Composer | 2.x |
| MySQL | **8.0+** 或 MariaDB 10.6+ |
| Nginx | 1.20+ |
| Node.js | **18+** 或 20 LTS（仅构建前端时需要） |

### PHP 必需扩展

```bash
php -m | grep -E 'bcmath|ctype|curl|dom|fileinfo|json|mbstring|openssl|pdo|pdo_mysql|tokenizer|xml'
```

安装示例（Ubuntu）：

```bash
sudo apt update
sudo apt install -y nginx mysql-server \
  php8.2-fpm php8.2-cli php8.2-mysql php8.2-mbstring php8.2-xml \
  php8.2-curl php8.2-zip php8.2-bcmath php8.2-tokenizer php8.2-fileinfo \
  git unzip curl
```

---

## 3. 域名与 DNS 规划

部署前请先规划并解析以下记录：

| 记录 | 类型 | 值 | 用途 |
|------|------|-----|------|
| `cloak.example.com` | A | 服务器公网 IP | 总后台 |
| `merchant.cloak.example.com` | A | 服务器公网 IP | 商户端 |
| `api.cloak.example.com` | A | 服务器公网 IP | 管理 API |
| `entry.cloak.example.com` | A | 服务器公网 IP | 商户域名 CNAME 目标 |

商户在后台添加域名后，需将业务域名 **CNAME** 到 `CLOAK_ENTRY_HOST`（如 `entry.cloak.example.com`）。

若使用 Cloudflare 等 CDN，建议在流量入口开启 **CF-IPCountry** 头传递，系统据此做地区判定。

---

## 4. 创建数据库

```bash
sudo mysql -u root -p
```

```sql
CREATE DATABASE cloak CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'cloak'@'localhost' IDENTIFIED BY '你的强密码';
GRANT ALL PRIVILEGES ON cloak.* TO 'cloak'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

---

## 5. 部署后端（Laravel）

### 5.1 拉取代码

```bash
sudo mkdir -p /opt/cloak
sudo chown -R $USER:$USER /opt/cloak
cd /opt/cloak

# 按实际仓库地址替换
git clone <你的仓库地址> .
# 或仅上传 backend、frontend 目录
```

### 5.2 安装依赖

```bash
cd /opt/cloak/backend

# 安装 Composer（若未安装）
curl -sS https://getcomposer.org/installer | php
sudo mv composer.phar /usr/local/bin/composer

composer install --no-dev --optimize-autoloader
```

### 5.3 配置环境变量

```bash
cp .env.example .env
nano .env
```

生产环境关键配置：

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

# 商户域名 CNAME 目标（与 entry 子域名一致）
CLOAK_ENTRY_HOST=entry.cloak.example.com

SANCTUM_STATEFUL_DOMAINS=cloak.example.com,merchant.cloak.example.com,api.cloak.example.com
```

生成应用密钥并初始化：

```bash
php artisan key:generate
php artisan migrate --force
php artisan db:seed --class=RbacSeeder --force
```

默认账号（**上线后请立即修改密码**）：

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 平台管理员 | `admin` | `admin123` |
| 商户管理员 | `merchant` | `merchant123` |

### 5.4 目录权限

```bash
cd /opt/cloak/backend

sudo chown -R www-data:www-data storage bootstrap/cache
sudo chmod -R 775 storage bootstrap/cache
```

### 5.5 优化

```bash
php artisan config:cache
php artisan route:cache
php artisan view:cache
```

> 修改 `.env` 后需执行 `php artisan config:clear` 再重新 `config:cache`。

---

## 6. 构建并部署前端

### 6.1 安装 Node.js（若服务器未安装）

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node -v && npm -v
```

### 6.2 配置生产环境变量

```bash
cd /opt/cloak/frontend
nano .env.production
```

```ini
VITE_APP_NAME=H5Cloak
VITE_API_URL=https://api.cloak.example.com/api
VITE_API_PROXY_URL=
```

> `VITE_API_URL` 必须指向线上 API 根路径，且包含 `/api` 后缀（与 Laravel 路由前缀一致）。两端共用这一份接口地址，页面标题由各自入口决定。

### 6.3 构建

```bash
npm ci
npm run build:admin
npm run build:merchant
```

### 6.4 发布静态文件

```bash
sudo mkdir -p /opt/cloak/www/admin /opt/cloak/www/merchant
sudo rsync -av --delete dist/admin/ /opt/cloak/www/admin/
sudo rsync -av --delete dist/merchant/ /opt/cloak/www/merchant/
sudo chown -R www-data:www-data /opt/cloak/www/admin /opt/cloak/www/merchant
```

后续更新某一端时，只重新构建并同步对应目录。

---

## 7. Nginx 配置

配置文件建议放在 `/etc/nginx/sites-available/cloak.conf`，并软链到 `sites-enabled`。

### 7.1 总后台前端 — `cloak.example.com`

```nginx
server {
    listen 80;
    server_name cloak.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name cloak.example.com;

    # SSL 证书（Certbot 或自有证书）
    ssl_certificate     /etc/letsencrypt/live/cloak.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cloak.example.com/privkey.pem;

    root /opt/cloak/www/admin;
    index index.html;

    # Vue Router history 模式
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    access_log /var/log/nginx/cloak-admin.access.log;
    error_log  /var/log/nginx/cloak-admin.error.log;
}
```

### 7.2 商户端前端 — `merchant.cloak.example.com`

```nginx
server {
    listen 80;
    server_name merchant.cloak.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name merchant.cloak.example.com;

    ssl_certificate     /etc/letsencrypt/live/merchant.cloak.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/merchant.cloak.example.com/privkey.pem;

    root /opt/cloak/www/merchant;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    access_log /var/log/nginx/cloak-merchant.access.log;
    error_log  /var/log/nginx/cloak-merchant.error.log;
}
```

### 7.3 管理 API — `api.cloak.example.com`

```nginx
server {
    listen 80;
    server_name api.cloak.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.cloak.example.com;

    ssl_certificate     /etc/letsencrypt/live/api.cloak.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.cloak.example.com/privkey.pem;

    root /opt/cloak/backend/public;
    index index.php;

    client_max_body_size 20m;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.2-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;

        # 若 API 在 Cloudflare / 反向代理后，传递真实 IP
        fastcgi_param HTTP_X_FORWARDED_FOR $proxy_add_x_forwarded_for;
        fastcgi_param HTTP_X_FORWARDED_PROTO $scheme;
    }

    location ~ /\.(?!well-known).* {
        deny all;
    }

    access_log /var/log/nginx/cloak-api.access.log;
    error_log  /var/log/nginx/cloak-api.error.log;
}
```

### 7.4 斗篷流量入口 — `entry.cloak.example.com`

商户业务域名 CNAME 到此地址，由 `routes/cloak.php` 处理跳转。

```nginx
server {
    listen 80;
    server_name entry.cloak.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name entry.cloak.example.com;

    ssl_certificate     /etc/letsencrypt/live/entry.cloak.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/entry.cloak.example.com/privkey.pem;

    root /opt/cloak/backend/public;
    index index.php;

    # 传递 CDN 国家码（Cloudflare 会自动带 CF-IPCountry）
    set_real_ip_from 0.0.0.0/0;
    real_ip_header X-Forwarded-For;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.2-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    access_log /var/log/nginx/cloak-entry.access.log;
    error_log  /var/log/nginx/cloak-entry.error.log;
}
```

### 7.5 启用并重载 Nginx

```bash
sudo ln -sf /etc/nginx/sites-available/cloak.conf /etc/nginx/sites-enabled/cloak.conf
sudo nginx -t
sudo systemctl reload nginx
```

---

## 8. SSL 证书（Let's Encrypt）

```bash
sudo apt install -y certbot python3-certbot-nginx

sudo certbot --nginx \
  -d cloak.example.com \
  -d merchant.cloak.example.com \
  -d api.cloak.example.com \
  -d entry.cloak.example.com

# 自动续期测试
sudo certbot renew --dry-run
```

---

## 9. 防火墙

```bash
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
sudo ufw status
```

仅开放 **22、80、443**，不要对公网暴露 3306。

---

## 10. 部署后验证清单

### 10.1 健康检查

```bash
curl -I https://api.cloak.example.com/up
# 期望 HTTP 200
```

### 10.2 登录管理后台

1. 浏览器访问 `https://cloak.example.com`
2. 使用 `admin / admin123` 登录
3. 登录后立即修改密码

### 10.3 测试开放 API

```bash
curl -X POST "https://api.cloak.example.com/api/open/decision" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ck_你的Token" \
  -d '{"ip":"1.2.3.4","userAgent":"Mozilla/5.0","referer":"https://example.com"}'
```

期望返回：

```json
{
  "code": 200,
  "msg": "判定成功",
  "data": {
    "page": "review",
    "action": "white",
    ...
  }
}
```

### 10.4 测试斗篷流量入口

1. 在商户后台添加域名与斗篷策略
2. 将测试域名 CNAME 到 `entry.cloak.example.com`
3. 访问测试域名，确认跳转到审核页或真实页

---

## 11. 与 WordPress 插件对接

WordPress 插件配置：

```php
define('CLOAK_API_URL', 'https://api.cloak.example.com/api/open/decision');
define('CLOAK_API_TOKEN', 'ck_你的Token');
```

插件位置：`wordpress-plugin/cloak-api.zip`

---

## 12. 更新与维护

### 后端更新

```bash
cd /opt/cloak/backend
git pull
composer install --no-dev --optimize-autoloader
php artisan migrate --force
php artisan config:cache
php artisan route:cache
sudo systemctl reload php8.2-fpm
```

### 前端更新

```bash
cd /opt/cloak/frontend
git pull
npm ci
npm run build:admin
npm run build:merchant
sudo rsync -av --delete dist/admin/ /opt/cloak/www/admin/
sudo rsync -av --delete dist/merchant/ /opt/cloak/www/merchant/
```

### 日志位置

| 日志 | 路径 |
|------|------|
| Laravel | `/opt/cloak/backend/storage/logs/laravel.log` |
| Nginx 管理后台 | `/var/log/nginx/cloak-admin.*.log` |
| Nginx API | `/var/log/nginx/cloak-api.*.log` |
| Nginx 流量入口 | `/var/log/nginx/cloak-entry.*.log` |

---

## 13. 常见问题

### Q1：前端能打开但接口 404 / CORS 错误

- 检查 `VITE_API_URL` 是否为 `https://api.cloak.example.com/api`
- 确认 Nginx `root` 指向 `backend/public`
- 确认 `php artisan route:cache` 后 `/api/login` 可访问

### Q2：`artisan migrate` 失败

- 检查 MySQL 账号权限与 `DB_*` 配置
- 确认 PHP 已安装 `pdo_mysql`

### Q3：地区判定不准

- 流量入口需经过 Cloudflare 等 CDN，或自行在反向代理传入 `CF-IPCountry` / `X-Country-Code` 头
- 本系统不内置 IP 库，依赖边缘节点国家码

### Q4：`storage` 或 `bootstrap/cache` 权限错误

```bash
sudo chown -R www-data:www-data storage bootstrap/cache
sudo chmod -R 775 storage bootstrap/cache
```

### Q5：商户域名 CNAME 后无法访问

- 确认 `CLOAK_ENTRY_HOST` 与 Nginx `server_name` 一致
- 确认 entry 域名 SSL 证书有效
- 在平台后台检查域名是否已校验、斗篷是否启用

### Q6：登录后立刻退出

- 检查 `SANCTUM_STATEFUL_DOMAINS` 是否包含管理后台域名
- 确认前后端均使用 HTTPS

---

## 14. 宝塔面板部署（简要）

完整 Debian + 宝塔分步文档见：**[deploy-debian-bt.md](./deploy-debian-bt.md)**

若使用宝塔面板：

1. 创建站点 `cloak.example.com`，根目录指向 `/opt/cloak/www/admin`，开启「伪静态」→ `vue` 或自定义 `try_files`
2. 创建站点 `merchant.cloak.example.com`，根目录指向 `/opt/cloak/www/merchant`，伪静态与总后台相同
3. 创建站点 `api.cloak.example.com`，根目录指向 `/opt/cloak/backend/public`，PHP 版本选 **8.2**
4. 创建站点 `entry.cloak.example.com`，根目录同样指向 `/opt/cloak/backend/public`
5. 四个站点分别申请 SSL
5. 在「计划任务」中可添加 `certbot renew` 或依赖宝塔自动续签
6. 数据库通过宝塔 MySQL 管理创建，其余步骤与本文第 5～6 节相同

---

## 15. 安全建议（上线前必做）

- [ ] 修改默认管理员、商户密码
- [ ] `APP_DEBUG=false`
- [ ] 数据库使用强密码，禁止外网访问
- [ ] 配置服务器 SSH 密钥登录，禁用 root 密码登录
- [ ] 定期备份数据库与 `storage/` 目录
- [ ] API Token 仅通过服务端配置，不要写入前端代码

---

## 附录：单机单域名简化方案（不推荐生产）

若暂时只有一个域名，可将 API 挂在前端域名下：

```nginx
# cloak.example.com 站点内
location /api/ {
    alias /opt/cloak/backend/public/;
    try_files $uri $uri/ /index.php?$query_string;
    # ... php-fpm 配置
}
```

前端 `.env.production`：

```ini
VITE_API_URL=https://cloak.example.com/api
```

流量入口仍建议独立 `entry.` 子域名，否则商户 CNAME 无法与管理后台共用。
