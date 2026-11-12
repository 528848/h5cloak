# Fast frontend-only deploy to the test server (SSH alias cloak-test).
#
#   powershell -File deploy/deploy-fe.ps1
#
# Only rebuilds the two frontends and replaces /opt/cloak/www. No Go binary,
# no install.sh, no apt, no service restart (nginx serves the static files
# directly). The whole tar is sent in one piece over a single SSH connection
# (no chunking) and verified by sha256; on mismatch it retries the whole file.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
. (Join-Path $root 'scripts\dev\common.ps1')
Use-Toolchain

Write-Host '== build frontend =='
Push-Location (Join-Path $root 'frontend')
& cmd /c "npm run build >nul 2>&1"
if ($LASTEXITCODE -ne 0) {
    & cmd /c "npm run build 2>&1" | Select-Object -Last 40 | ForEach-Object { Write-Host $_ }
    throw 'npm run build failed'
}
Pop-Location

$stage = Join-Path $root 'release-fe'
if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory $stage | Out-Null
foreach ($p in 'admin', 'merchant') {
    Copy-Item (Join-Path $root "frontend\dist\$p") (Join-Path $stage $p) -Recurse
}
$tar = Join-Path $root 'release-fe.tar.gz'
if (Test-Path $tar) { Remove-Item $tar }
& tar -czf $tar -C $stage .
if ($LASTEXITCODE -ne 0) { throw 'tar failed' }

$bytes = [IO.File]::ReadAllBytes($tar)
$sha = [Security.Cryptography.SHA256]::Create()
$hash = ([BitConverter]::ToString($sha.ComputeHash($bytes)) -replace '-', '').ToLower()
Write-Host ("package {0:N1} MB  sha256=$hash" -f ($bytes.Length / 1MB))

$remoteDir = '/opt/cloak-releases/fe-parts'
$install = @"
set -e
D=$remoteDir
mkdir -p /opt/cloak-releases/fe
rm -rf /opt/cloak-releases/fe/*
tar -xzf `$D/fe.tgz -C /opt/cloak-releases/fe
rsync -a --delete /opt/cloak-releases/fe/admin/ /opt/cloak/www/admin/
rsync -a --delete /opt/cloak-releases/fe/merchant/ /opt/cloak/www/merchant/
chown -R cloak:cloak /opt/cloak/www
rm -rf /opt/cloak-releases/fe `$D
echo FE_OK
"@
$installBytes = [Text.Encoding]::UTF8.GetBytes(($install -replace "`r`n", "`n" -replace "`r", "`n"))

$perl = [IO.File]::ReadAllBytes((Join-Path $PSScriptRoot 'scripts\chunk-recv.pl'))
$perl = [Text.Encoding]::UTF8.GetBytes(([Text.Encoding]::UTF8.GetString($perl) -replace "`r`n", "`n" -replace "`r", "`n"))
$b64 = [Convert]::ToBase64String($perl)
$remoteCmd = "echo $b64 | base64 -d > /tmp/cloak-recv.pl && CLOAK_PARTS='$remoteDir' exec perl /tmp/cloak-recv.pl"

$ErrorActionPreference = 'Continue'
function Invoke-Upload {
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = 'ssh'
    $psi.Arguments = "-o ConnectTimeout=120 -o ServerAliveInterval=5 -o ServerAliveCountMax=120 -o TCPKeepAlive=yes cloak-test `"$remoteCmd`""
    $psi.RedirectStandardInput = $true
    $psi.RedirectStandardOutput = $true
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $proc = [Diagnostics.Process]::Start($psi)
    $stream = $proc.StandardInput.BaseStream
    $send = { param($s) $b = [Text.Encoding]::ASCII.GetBytes("$s`n"); $stream.Write($b, 0, $b.Length); $stream.Flush() }
    $readOut = {
        param($ms)
        $t = $proc.StandardOutput.ReadLineAsync()
        if (-not $t.Wait($ms)) { throw 'read timeout' }
        return $t.Result
    }
    try {
        if ((& $readOut 180000) -ne 'READY') { throw 'not ready' }
        & $send "PUT fe.tgz $($bytes.Length)"
        $stream.Write($bytes, 0, $bytes.Length)
        $stream.Flush()
        $line = & $readOut 300000
        if ($line -ne "HASH fe.tgz $hash") { throw "bad transfer: $line" }
        Write-Host 'upload OK'
        & $send "RUN $($installBytes.Length)"
        $stream.Write($installBytes, 0, $installBytes.Length)
        $stream.Flush()
        $last = $null
        while ($true) {
            $l = & $readOut 180000
            if ($null -eq $l) { break }
            Write-Host $l
            $last = $l
            if ($l -eq 'FE_OK') { break }
        }
        return ($last -eq 'FE_OK')
    } finally {
        if ($proc -and -not $proc.HasExited) { try { $proc.Kill() } catch {} }
    }
}

$done = $false
for ($t = 1; $t -le 3 -and -not $done; $t++) {
    Write-Host "attempt $t"
    try { $done = Invoke-Upload } catch { Write-Host "attempt $t failed: $($_.Exception.Message)"; Start-Sleep -Seconds 4 }
}
if (-not $done) { throw 'deploy failed' }
Write-Host 'DEPLOY_OK'
