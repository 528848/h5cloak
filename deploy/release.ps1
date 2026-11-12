# 不要用本脚本发布测试环境。整包 scp 会静默损坏，而且每次重连 SSH 都很慢。
# 测试环境用：powershell -File deploy/deploy-test.ps1
# 本脚本只保留给已经确认链路稳定、并且需要带安装参数的首次装机。
# 从 Windows 开发机发布到服务器：交叉编译 Go、构建两个前端、打包上传、远程执行 install.sh。
# 用法（仓库根目录）：
#   powershell -File deploy/release.ps1 -Target cloak-test                       # 升级（沿用服务器上已记录的主机名）
#   powershell -File deploy/release.ps1 -Target cloak-test -InstallArgs "--admin-host a.x --merchant-host m.x --entry-host e.x --acme-email ops@x --demo-origins"
#   powershell -File deploy/release.ps1 -Target cloak-test -SkipFrontend         # 只发后端
# -Target 是 ~/.ssh/config 里的别名或 user@ip；需要已配置密钥登录。
param(
    [Parameter(Mandatory = $true)][string]$Target,
    [string]$InstallArgs = '',
    [switch]$SkipFrontend,
    [switch]$SkipBackend,
    [switch]$NoInstall
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
. (Join-Path $root 'scripts\dev\common.ps1')
Use-Toolchain
$env:GOTOOLCHAIN = 'local'; $env:GOFLAGS = '-mod=mod'; $env:CGO_ENABLED = '0'; $env:GOOS = 'linux'; $env:GOARCH = 'amd64'

$out = Join-Path $root 'release'
if (-not (Test-Path $out)) { New-Item -ItemType Directory $out | Out-Null }
foreach ($d in 'bin', 'www', 'migrations') { New-Item -ItemType Directory -Force (Join-Path $out $d) | Out-Null }

$version = (git -C $root describe --tags --always 2>$null); if (-not $version) { $version = 'dev' }
$commit = (git -C $root rev-parse --short HEAD 2>$null); if (-not $commit) { $commit = 'unknown' }
$built = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X cloak/internal/pkg/version.Version=$version -X cloak/internal/pkg/version.Commit=$commit -X cloak/internal/pkg/version.BuildTime=$built"

if (-not $SkipBackend) {
    Write-Host "== 交叉编译 Go ($version $commit) =="
    Push-Location (Join-Path $root 'backend-go')
    foreach ($c in 'admin', 'gateway', 'worker', 'migrate', 'replay', 'mockorigin') {
        & go build -trimpath -ldflags $ldflags -o (Join-Path $out "bin\$c") "./cmd/$c"
        if ($LASTEXITCODE -ne 0) { throw "go build $c failed" }
    }
    Pop-Location
    Remove-Item (Join-Path $out 'migrations\*') -Force -ErrorAction SilentlyContinue
    Copy-Item (Join-Path $root 'backend-go\migrations\*.sql') (Join-Path $out 'migrations')
}
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

if (-not $SkipFrontend) {
    Write-Host "== 构建前端 =="
    Push-Location (Join-Path $root 'frontend')
    # npm 把告警写到 stderr，PowerShell 5 会把它当错误，这里走 cmd 静默执行，失败再重跑一次给出输出
    & cmd /c "npm run build >nul 2>&1"
    if ($LASTEXITCODE -ne 0) {
        & cmd /c "npm run build 2>&1" | Select-Object -Last 30 | ForEach-Object { Write-Host $_ }
        throw "npm run build failed"
    }
    Pop-Location
    foreach ($p in 'admin', 'merchant') {
        $dst = Join-Path $out "www\$p"
        if (Test-Path $dst) { Remove-Item $dst -Recurse -Force }
        Copy-Item (Join-Path $root "frontend\dist\$p") $dst -Recurse
    }
}

Write-Host "== 打包 =="
$deployDst = Join-Path $out 'deploy'
if (Test-Path $deployDst) { Remove-Item $deployDst -Recurse -Force }
Copy-Item $PSScriptRoot $deployDst -Recurse
"$version ($commit) $built" | Set-Content (Join-Path $out 'VERSION') -NoNewline
$tar = Join-Path $root 'release.tar.gz'
if (Test-Path $tar) { Remove-Item $tar }
& tar -czf $tar -C $root release
if ($LASTEXITCODE -ne 0) { throw "tar failed" }
Write-Host ("包大小 {0:N1} MB" -f ((Get-Item $tar).Length / 1MB))

Write-Host "== 上传到 $Target =="
& ssh $Target "mkdir -p /opt/cloak-releases"
$uploaded = $false
for ($try = 1; $try -le 3 -and -not $uploaded; $try++) {
    & scp -q $tar "${Target}:/opt/cloak-releases/release-$commit.tar.gz"
    if ($LASTEXITCODE -eq 0) { $uploaded = $true } else { Write-Host "scp 第 $try 次失败，重试"; Start-Sleep 3 }
}
if (-not $uploaded) { throw "scp failed" }
$remote = "set -e; cd /opt/cloak-releases && rm -rf release && tar -xzf release-$commit.tar.gz && rm -rf current && mv release current && ls -t release-*.tar.gz | tail -n +4 | xargs -r rm -f && sed -i 's/\r$//' current/deploy/*.sh current/deploy/scripts/*.sh && chmod +x current/deploy/*.sh current/deploy/scripts/*.sh"
& ssh $Target $remote
if ($LASTEXITCODE -ne 0) { throw "remote unpack failed" }

if ($NoInstall) { Write-Host "已上传到 /opt/cloak-releases/current，未执行安装"; exit 0 }
Write-Host "== 远程安装 =="
& ssh $Target "cd /opt/cloak-releases/current && ./deploy/install.sh $InstallArgs"
if ($LASTEXITCODE -ne 0) { throw "install.sh failed" }
