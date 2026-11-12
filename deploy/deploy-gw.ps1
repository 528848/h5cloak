# 只发布网关到测试环境（SSH 别名 cloak-test）。
# 不编译前端，不跑 install.sh / apt / 迁移。深度代理这类 gateway 改动用这个。
#
#   powershell -File deploy/deploy-gw.ps1
#   powershell -File deploy/deploy-gw.ps1 -Name worker
#
param(
    [ValidateSet('gateway', 'worker')]
    [string]$Name = 'gateway'
)
# 传输方式和 deploy-test.ps1 相同：一条 SSH，按 4MB 发给 chunk-recv.pl，校验 sha256 后再替换。
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
. (Join-Path $root 'scripts\dev\common.ps1')
Use-Toolchain
$env:GOTOOLCHAIN = 'local'
$env:GOFLAGS = '-mod=mod'
$env:CGO_ENABLED = '0'
$env:GOOS = 'linux'
$env:GOARCH = 'amd64'

$version = (git -C $root describe --tags --always 2>$null); if (-not $version) { $version = 'dev' }
$commit = (git -C $root rev-parse --short HEAD 2>$null); if (-not $commit) { $commit = 'unknown' }
$built = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X cloak/internal/pkg/version.Version=$version -X cloak/internal/pkg/version.Commit=$commit -X cloak/internal/pkg/version.BuildTime=$built"

$stage = Join-Path $root 'release-gw'
if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory -Force $stage | Out-Null

Write-Host "== 交叉编译 $Name ($version $commit $built) =="
Push-Location (Join-Path $root 'backend-go')
& go build -trimpath -ldflags $ldflags -o (Join-Path $stage $Name) "./cmd/$Name"
if ($LASTEXITCODE -ne 0) { throw 'go build gateway failed' }
Pop-Location
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

$tarPath = Join-Path $root 'gateway.tar.gz'
if (Test-Path $tarPath) { Remove-Item $tarPath }
& tar -czf $tarPath -C $stage $Name
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

$remoteDir = "/opt/cloak-releases/$Name-$commit"
$names = ($parts | ForEach-Object { "$remoteDir/$($_.Name)" }) -join ' '
$install = @"
set -e
mkdir -p /opt/cloak-releases
cat $names > /opt/cloak-releases/$Name-$commit.tar.gz
gzip -t /opt/cloak-releases/$Name-$commit.tar.gz
got=`$(sha256sum /opt/cloak-releases/$Name-$commit.tar.gz | cut -d' ' -f1)
echo "ARCHIVE `$got"
test "`$got" = "$fullHash"
rm -rf /tmp/cloak-bin
mkdir -p /tmp/cloak-bin
tar -xzf /opt/cloak-releases/$Name-$commit.tar.gz -C /tmp/cloak-bin
test -f /tmp/cloak-bin/$Name
cp -a /opt/cloak/bin/$Name /opt/cloak/bin/$Name.prev
install -m 755 /tmp/cloak-bin/$Name /opt/cloak/bin/$Name
chown cloak:cloak /opt/cloak/bin/$Name
rm -rf "$remoteDir" /tmp/cloak-bin
systemctl restart cloak-$Name
ok=0
if [ "$Name" = "gateway" ]; then
  for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
    if curl -fsS --max-time 2 http://127.0.0.1:8090/_cloak/health >/tmp/cloak-health; then
      ok=1
      break
    fi
    sleep 1
  done
else
  for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
    if systemctl is-active --quiet cloak-$Name; then
      ok=1
      break
    fi
    sleep 1
  done
fi
if [ "`$ok" != 1 ]; then
  echo HEALTH_FAIL
  systemctl --no-pager --full status cloak-$Name | head -n 20 || true
  journalctl -u cloak-$Name -n 40 --no-pager || true
  if [ -f /opt/cloak/bin/$Name.prev ]; then
    install -m 755 /opt/cloak/bin/$Name.prev /opt/cloak/bin/$Name
    chown cloak:cloak /opt/cloak/bin/$Name
    systemctl restart cloak-$Name || true
    echo RESTORED_PREV
  fi
  exit 1
fi
echo "ACTIVE:`$(systemctl is-active cloak-$Name)"
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
        $line = Read-Out 180000
        if ($null -eq $line) { break }
        Write-Host $line
        $last = $line
        if ($line -eq 'INSTALL_OK') { break }
    }
    if ($last -ne 'INSTALL_OK') { throw 'install failed' }
    Write-Host "DEPLOY_OK $built"
} finally {
    if ($proc -and -not $proc.HasExited) { try { $proc.Kill() } catch {} }
}
