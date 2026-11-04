# 测试服务器部署

目标：公网测试域名上验证「同一地址，带归因参数返回真实内容，不带返回审核内容」，HTTPS 证书自动签发。

## 需要准备

- 一台 Linux 服务器（Ubuntu 22.04 / Debian 12），开放 80、443。
- 两个可控 DNS 的域名：
  - `entry.<你的域名>`：A 记录指到服务器，作为托管入口，`gateway.entry_host` 填它。
  - `shop.<你的域名>`：模拟商户站点，CNAME 到 `entry.<你的域名>`。
- MySQL 8 与 Redis 7（同机即可）。

## 步骤

```bash
git clone <repo> /srv/cloak && cd /srv/cloak
sudo ./deploy/install.sh
```

按脚本结尾提示填配置、建库、迁移、启动。

### 源站

测试阶段可以直接用仓库里的模拟源站：

```bash
cd backend-go && go build -o /opt/cloak/bin/mockorigin ./cmd/mockorigin
nohup /opt/cloak/bin/mockorigin -kind real   -addr 127.0.0.1:9101 &
nohup /opt/cloak/bin/mockorigin -kind review -addr 127.0.0.1:9102 &
```

在商户端把站点真实源站填 `http://127.0.0.1:9101`，审核源站 `http://127.0.0.1:9102`，审核内容来源选「审核源站」。

### 证书

`gateway.tls.mode: acme`，首次访问某托管域名时按需签发；只对快照中已绑定的域名签发，其它 SNI 直接拒绝握手。
先用 `staging: true` 跑通流程（浏览器会提示证书不受信任，属正常），再改为 `false` 并重启网关。

### 验收

1. 无痕窗口打开 `https://shop.<域名>/`：审核内容。
2. 打开 `https://shop.<域名>/?ref=<已登记分销码>`：真实内容，地址栏不变。
3. 同一窗口再开 `https://shop.<域名>/second`：仍是真实内容。
4. 新无痕窗口打开同一地址：审核内容。
5. 停掉真实源站进程，带码访问按站点「失败兜底」返回。
6. `curl -sI https://shop.<域名>/ | grep -i server` 无输出（不暴露源站特征）。

### 前置 CDN 的注意事项

Cloudflare 等 CDN 前置时，必须把 HTML 设为不缓存（网关已返回 `Cache-Control: private, no-store` 与 `Vary: Cookie`，但请在 CDN 侧确认规则），并把 CDN 出口 IP 段加入 `gateway.trusted_proxies`。
