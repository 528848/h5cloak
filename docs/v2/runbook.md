# 斗篷系统 V2 运行手册

部署步骤见 `deploy-guide.md`。本文是装好之后的日常操作。命令都在服务器上执行。

## 1. 看状态

```bash
systemctl status cloak-admin cloak-gateway cloak-worker
curl -s http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8090/_cloak/health
curl -s http://127.0.0.1:8080/api/version
journalctl -u cloak-gateway -n 100 --no-pager
journalctl -u cloak-admin -u cloak-worker -p warning --since '1 hour ago'
```

三个服务都是 `Restart=always`。改 `/opt/cloak/env/*.env` 或 yaml 后要重启对应服务。

## 2. 升级

开发机执行 `powershell -File deploy/release.ps1 -Target <别名>`。脚本会替换二进制和前端、跑迁移、重启服务。`/opt/cloak/env/common.env` 里的密码和密钥保留。发布包留最近 3 份在 `/opt/cloak-releases/`。

回滚：把上一份 `release-<commit>.tar.gz` 解到 `current` 再跑 `./deploy/install.sh`。数据库迁移的 down 脚本在 `migrations/`，不要在有数据的库上随意回滚。

## 3. 备份与恢复

每天 03:30 `cloak-backup.timer` 把库和配置打到 `/opt/cloak/backups/`，保留 7 天。手动备份：

```bash
/opt/cloak/bin/cloak-backup.sh
```

恢复（会覆盖当前库，先停写入）：

```bash
systemctl stop cloak-admin cloak-worker
zcat /opt/cloak/backups/cloak_v2-<时间>.sql.gz | mysql -uroot cloak_v2
systemctl start cloak-admin cloak-worker
```

网关可以不停：它读的是 Redis 快照，恢复库之后在总后台对站点点一次发布，或重启 admin（启动时会全量发布快照）。

## 4. 密钥

| 文件 | 内容 |
| --- | --- |
| `common.env` | 数据库密码、`CLOAK_AUTH_JWT_SECRET`、`CLOAK_AUTH_SECRET_KEY` |
| `admin.env` | 总后台对外地址、商户端地址、入口域名 |
| `gateway.env` | ACME 邮箱、是否用测试证书 |
| `worker.env` | 商户端地址；SMTP、Telegram 也写在这里 |

`JWT_SECRET` 与 `SECRET_KEY` 必须 admin 和 gateway 一致。改 `JWT_SECRET` 会使所有人重新登录。改 `SECRET_KEY` 后，已有站点的服务端密钥无法解密，需要在站点页重置。

## 5. 换域名或入口

改 `admin.env`、`gateway.env`、`worker.env` 里的地址，再改 `/opt/cloak/env/install.conf`，然后：

```bash
cd /opt/cloak-releases/current && ./deploy/install.sh
```

`install.sh` 会按 `install.conf` 重写 Nginx 与 env（不重写 `common.env`）并重启。

## 6. 证书从测试切到正式

```bash
sed -i 's/CLOAK_GATEWAY_TLS_STAGING="true"/CLOAK_GATEWAY_TLS_STAGING="false"/' /opt/cloak/env/gateway.env
systemctl restart cloak-gateway
```

下次访问该域名时重新签发。确认：

```bash
echo | openssl s_client -connect 127.0.0.1:443 -servername <域名> 2>/dev/null | openssl x509 -noout -issuer
```

签发方不应再出现 `(STAGING)`。

## 7. 日志与磁盘

决策日志按商户保留天数清理，平台上限在总后台「系统配置」。导出文件在 `/opt/cloak/data/exports`，过期由 worker 删。证书与上传在 `/opt/cloak/data`。

磁盘告警看 `df -h /` 和 `du -sh /opt/cloak/data /opt/cloak/backups /var/log`。

## 8. WordPress 整页缓存

插件的「站内页面」审核模式会替换主查询。WP Super Cache、W3 Total Cache、LiteSpeed Cache 这类整页缓存会把第一个访客的页面缓存给后面所有人。使用该模式时：

- 关闭整页缓存，或
- 把带分销参数的 URL、以及 Cookie 名以站点设置为准的 `_ck` 开头的请求排除出缓存。

托管接入和「审核源站」模式不受这个影响。

## 9. 常见问题

| 现象 | 处理 |
| --- | --- |
| 域名校验失败 | 看返回的 `verifyMessage`。CNAME 应指向入口域名；A 记录指向入口 IP 也可以 |
| HTTPS 握手失败 | `journalctl -u cloak-gateway` 搜 `acme`。80 端口必须能从公网访问到这台机器 |
| 浏览器提示证书不受信任 | 仍在用 staging，见第 6 节 |
| 改了站点配置网关没跟上 | 正常应秒级生效。没有 Redis 时重启 gateway |
| 开放接口 429 | 总后台「系统配置」里的每密钥每秒、每站点每分钟 |
| 商户无法启用站点 | 总后台强制停用过，需总后台点恢复 |
