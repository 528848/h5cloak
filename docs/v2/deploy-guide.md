# 斗篷系统 V2 部署指南

测试环境与生产用同一套脚本。当前测试机已经按本文装好，地址见文末「当前测试环境」。

## 1. 组成

| 进程 | 作用 | 监听 |
| --- | --- | --- |
| Nginx | 80 端口按 Host 分流 | `:80` |
| cloak-admin | 总后台与商户端 API | `127.0.0.1:8080` |
| cloak-gateway | 托管反向代理、开放接口、HTTPS 证书 | `127.0.0.1:8090`、`:443` |
| cloak-worker | 日志入库、聚合、告警、Webhook、导出、清理 | 指标 `127.0.0.1:9091` |
| MySQL 8.4 | 业务库 `cloak_v2` | `127.0.0.1:3306` |
| Redis | 会话、快照、限流 | `127.0.0.1:6379` |

控制台是静态文件，放在 `/opt/cloak/www/admin` 与 `/opt/cloak/www/merchant`。`/api` 由 Nginx 转到 admin。其它 Host（商户托管域名、ACME 校验）转到网关。网关自己监听 443，只给已绑定域名签发证书。

程序与密钥：

- 二进制 `/opt/cloak/bin/`
- 配置 `/opt/cloak/configs/*.yaml`（不含密码）
- 密钥与地址 `/opt/cloak/env/*.env`（`common.env` 首次安装生成，权限 640）
- 数据 `/opt/cloak/data/`（上传、导出、证书、日志 WAL）
- 备份 `/opt/cloak/backups/`，每天 03:30，保留 7 天

## 2. 服务器要求

Debian 12/13 或 Ubuntu 22.04/24.04，amd64，2 核 4G 起。开放 22、80、443。脚本会装 Nginx、Redis、MySQL 8.4 LTS（官方源，签名密钥用 `RPM-GPG-KEY-mysql-2025`）。时区设为 `Asia/Shanghai`。

没有自己的域名时，可以用 `sslip.io`：`名字.<IP 用横线>.sslip.io` 会解析到这台机器，适合先跑通。正式环境换成自己的域名。

## 3. 从开发机发布

开发机需要能 `ssh` 到服务器（`~/.ssh/config` 里写好别名和密钥）。在仓库根目录：

```powershell
# 首次安装
powershell -File deploy/release.ps1 -Target cloak-test -InstallArgs "--admin-host admin.example.com --merchant-host merchant.example.com --entry-host entry.example.com --acme-email ops@example.com --demo-origins"

# 之后升级（主机名记在服务器 /opt/cloak/env/install.conf）
powershell -File deploy/release.ps1 -Target cloak-test
```

脚本做的事：交叉编译 Linux amd64 二进制、构建两个前端、打包上传到 `/opt/cloak-releases/`、执行 `deploy/install.sh`。`install.sh` 会迁移数据库、重载 Nginx、重启三个服务并做健康检查。已有的 `common.env`（数据库密码、JWT）不会被覆盖。

只要后端变了可以加 `-SkipFrontend`。`-NoInstall` 只上传不安装。

服务器上已有 Go 和 Node 时，也可以在服务器执行 `deploy/build.sh` 再 `sudo ./deploy/install.sh`。

## 4. 证书

默认 `CLOAK_GATEWAY_TLS_STAGING=true`，用 Let's Encrypt 测试环境签发。浏览器会提示不受信任，这是预期。流程跑通后改成正式证书：

```bash
sed -i 's/CLOAK_GATEWAY_TLS_STAGING="true"/CLOAK_GATEWAY_TLS_STAGING="false"/' /opt/cloak/env/gateway.env
systemctl restart cloak-gateway
```

证书按域名按需签发，存在 `/opt/cloak/data/certs`。只给快照里已绑定的域名签。

## 5. 托管域名

商户在「站点管理」添加域名，校验方式用 CNAME，目标是 `entry_host`。域名也可以用 A 记录直接指到入口机器的 IP，校验同样通过。

测试机上开了演示源站（`--demo-origins`）：真实 `http://127.0.0.1:9101`，审核 `http://127.0.0.1:9102`。生产不要启用这两个服务。

## 6. 验收（测试机已通过）

同一 URL：

- 不带参数：审核页
- 带已登记分销码：真实页，并写下访客 Cookie
- 带着 Cookie 再访问其它路径：仍然是真实页
- 不带 Cookie 直接打开内页：审核页

调试：请求头 `X-Cloak-Debug: decide` 返回判定 JSON，不回源。

## 7. 当前测试环境

| 项 | 值 |
| --- | --- |
| 机器 | `170.178.174.31`，Debian 13，2 核 4G，主机名 `cloak-go` |
| 登录 | `ssh cloak-test`（本机 `~/.ssh/config`，密钥 `~/.ssh/cloak_test`） |
| 域名 | `h5cloak.com`（Cloudflare DNS，目前只有根域名 A 记录） |
| 商户端 | http://h5cloak.com （`merchant` / `merchant123`） |
| 总后台 | http://admin.h5cloak.com ，需先加 A 记录到 `170.178.174.31`。记录生效前仍可用 http://admin.170-178-174-31.sslip.io |
| 入口 | 商户托管域名 CNAME 到 `h5cloak.com`，或 A 记录指向 `170.178.174.31` |
| 演示站点 | http://demo.170-178-174-31.sslip.io ，分销码 `DEMO1` |
| 证书 | 托管域名走 Let's Encrypt 正式环境。控制台目前是 HTTP |
| 版本 | 见 `curl -s http://127.0.0.1:8080/api/version`（在服务器上） |

种子账号只用于测试。接到真实流量前改掉密码。root 密码曾经在对话里出现过，建议在服务器上执行 `passwd` 换掉。
