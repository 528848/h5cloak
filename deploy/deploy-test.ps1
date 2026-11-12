# 发布到测试环境（SSH 别名 cloak-test）。一条连接传完，不再逐段重连。
#
#   powershell -File deploy/deploy-test.ps1            # 增量：只编译 admin，前端全量，复用服务器上的 gateway/worker 等
#   powershell -File deploy/deploy-test.ps1 -Full      # 六个 Go 程序都重编，并走完整 install.sh（含 apt）
#   powershell -File deploy/deploy-gw.ps1              # 只换网关二进制并重启 cloak-gateway
#
# 为什么不用 deploy/release.ps1：
#   整包 scp 到这台机器会静默损坏（大小一样、哈希不同）。
#   每开一次 SSH 握手要一两分钟，所以不能一段一个连接。
# 做法：本地打 tar，经同一条 SSH 按 4MB 原始字节发给 deploy/scripts/chunk-recv.pl，
# 对端校验 sha256 后再安装。增量安装加 --skip-packages，不跑 apt。
param([switch]$Full)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
. (Join-Path $root 'scripts\dev\common.ps1')
Use-Toolchain
$env:GOTOOLCHAIN = 'local'
$env:GOFLAGS = '-mod=mod'
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'

$out = Join-Path $root 'release'
if (Test-Path $out) { Remove-Item $out -Recurse -Force }
foreach ($d in 'bin', 'www', 'migrations') { New-Item -ItemType Directory -Force (Join-Path $out $d) | Out-Null }

$version = (git -C $root describe --tags --always 2>$null); if (-not $version) { $version = 'dev' }
$commit = (git -C $root rev-parse --short HEAD 2>$null); if (-not $commit) { $commit = 'unknown' }
$built = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X cloak/internal/pkg/version.Version=$version -X cloak/internal/pkg/version.Commit=$commit -X cloak/internal/pkg/version.BuildTime=$built"
$cmds = if ($Full) { 'admin', 'gateway', 'worker', 'migrate', 'replay', 'mockorigin' } else { 'admin' }

Write-Host "== 交叉编译 $($cmds -join ',') ($version $commit) =="
Push-Location (Join-Path $root 'backend-go')
foreach ($c in $cmds) {
    & go build -trimpath -ldflags $ldflags -o (Join-Path $out "bin\$c") "./cmd/$c"
    if ($LASTEXITCODE -ne 0) { throw "go build $c failed" }
}
Pop-Location
Copy-Item (Join-Path $root 'backend-go\migrations\*.sql') (Join-Path $out 'migrations')
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host '== 构建前端 =='
Push-Location (Join-Path $root 'frontend')
& cmd /c "npm run build >nul 2>&1"
if ($LASTEXITCODE -ne 0) {
    & cmd /c "npm run build 2>&1" | Select-Object -Last 40 | ForEach-Object { Write-Host $_ }
    throw 'npm run build failed'
}
Pop-Location
foreach ($p in 'admin', 'merchant') {
    Copy-Item (Join-Path $root "frontend\dist\$p") (Join-Path $out "www\$p") -Recurse
}

Copy-Item (Join-Path $root 'deploy') (Join-Path $out 'deploy') -Recurse
"$version ($commit) $built" | Set-Content (Join-Path $out 'VERSION') -NoNewline
$tarPath = Join-Path $root 'release.tar.gz'
if (Test-Path $tarPath) { Remove-Item $tarPath }
& tar -czf $tarPath -C $root release
if ($LASTEXITCODE -ne 0) { throw 'tar failed' }
Write-Host ("包大小 {0:N1} MB  $built" -f ((Get-Item $tarPath).Length / 1MB))

$chunk = 4MB
$bytes = [IO.File]::ReadAllBytes($tarPath)
$sha = [Security.Cryptography.SHA256]::Create()
$fullHash = ([BitConverter]::ToString($sha.ComputeHash($bytes)) -replace '-', '').ToLower()
$n = [int][math]::Ceiling($bytes.Length / $chunk)
$parts = @()
for ($i = 0; $i -lt $n; $i++) {
    $off = [int64]$i * $chunk
    $len = [int][math]::Min([int64]$chunk, $bytes.Length - $off)
    $part = New-Object byte[] $len
    [Array]::Copy($bytes, $off, $part, 0, $len)
    $parts += @{ Name = ('part_{0:D3}' -f $i); Data = $part; Hash = ([BitConverter]::ToString($sha.ComputeHash($part)) -replace '-', '').ToLower() }
}
Write-Host "parts=$n sha256=$fullHash"

$remoteDir = "/opt/cloak-releases/parts-$commit"
$names = ($parts | ForEach-Object { "$remoteDir/$($_.Name)" }) -join ' '
$skip = if ($Full) { '' } else { '--skip-packages' }
$install = @"
set -e
cat $names > /opt/cloak-releases/release-$commit.tar.gz
gzip -t /opt/cloak-releases/release-$commit.tar.gz
got=`$(sha256sum /opt/cloak-releases/release-$commit.tar.gz | cut -d' ' -f1)
echo "ARCHIVE `$got"
test "`$got" = "$fullHash"
cd /opt/cloak-releases
rm -rf release
tar -xzf release-$commit.tar.gz
if [ ! -d current/bin ]; then mv release current; else
  cp -a release/bin/. current/bin/
  rm -rf current/www/admin current/www/merchant
  mkdir -p current/www
  cp -a release/www/admin current/www/admin
  cp -a release/www/merchant current/www/merchant
  cp -a release/migrations/. current/migrations/
  cp -a release/deploy/. current/deploy/
  cp -a release/VERSION current/VERSION
  rm -rf release
fi
rm -rf "$remoteDir"
ls -t release-*.tar.gz | tail -n +4 | xargs -r rm -f
cd current
sed -i 's/\r`$//' deploy/*.sh deploy/scripts/*.sh
chmod +x deploy/*.sh deploy/scripts/*.sh
./deploy/install.sh $skip
echo INSTALL_OK
"@
$installBytes = [Text.Encoding]::UTF8.GetBytes(($install -replace "`r`n", "`n" -replace "`r", "`n"))

$perl = [IO.File]::ReadAllBytes((Join-Path $PSScriptRoot 'scripts\chunk-recv.pl'))
$perl = [Text.Encoding]::UTF8.GetBytes(([Text.Encoding]::UTF8.GetString($perl) -replace "`r`n", "`n" -replace "`r", "`n"))
$b64 = [Convert]::ToBase64String($perl)
$remoteCmd = "echo $b64 | base64 -d > /tmp/cloak-recv.pl && CLOAK_PARTS='$remoteDir' exec perl /tmp/cloak-recv.pl"

$ErrorActionPreference = 'Continue'
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = 'ssh'
$psi.Arguments = "-o ConnectTimeout=120 -o ServerAliveInterval=5 -o ServerAliveCountMax=120 -o TCPKeepAlive=yes cloak-test `"$remoteCmd`""
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true
$proc = [Diagnostics.Process]::Start($psi)
$stream = $proc.StandardInput.BaseStream

function Send-Text([string]$s) {
    $buf = [Text.Encoding]::ASCII.GetBytes("$s`n")
    $stream.Write($buf, 0, $buf.Length)
    $stream.Flush()
}
function Read-Out([int]$ms) {
    $task = $proc.StandardOutput.ReadLineAsync()
    if (-not $task.Wait($ms)) { throw 'read timeout' }
    return $task.Result
}

try {
    $ready = Read-Out 180000
    if ($ready -ne 'READY') { throw "not ready: $ready" }
    Write-Host 'READY'
    foreach ($part in $parts) {
        Write-Host "send $($part.Name)"
        Send-Text "PUT $($part.Name) $($part.Data.Length)"
        $stream.Write($part.Data, 0, $part.Data.Length)
        $stream.Flush()
        $line = Read-Out 180000
        if ($line -ne "HASH $($part.Name) $($part.Hash)") { throw "bad part: $line" }
        Write-Host "$($part.Name) OK"
    }
    Send-Text "RUN $($installBytes.Length)"
    $stream.Write($installBytes, 0, $installBytes.Length)
    $stream.Flush()
    Write-Host '== install =='
    $last = $null
    while ($true) {
        $line = Read-Out 300000
        if ($null -eq $line) { break }
        Write-Host $line
        $last = $line
        if ($line -eq 'INSTALL_OK') { break }
    }
    if ($last -ne 'INSTALL_OK') { throw 'install failed' }
    Write-Host 'DEPLOY_OK'
} finally {
    if ($proc -and -not $proc.HasExited) { try { $proc.Kill() } catch {} }
}
