# 部署

步骤、测试环境地址和验收方法在 `docs/v2/deploy-guide.md`。
装好之后的备份、升级、证书和排障在 `docs/v2/runbook.md`。

从 Windows 开发机发布：

```powershell
powershell -File deploy/release.ps1 -Target <ssh 别名> -InstallArgs "--admin-host ... --merchant-host ... --entry-host ... --acme-email ..."
```

服务器上已有 Go 与 Node 时用 `deploy/build.sh` 生成 `release/`，再 `sudo ./deploy/install.sh`。
