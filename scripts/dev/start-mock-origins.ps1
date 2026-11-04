# 启动两个模拟源站：真实站 9101、审核站 9102（后台运行，日志在 .local/logs）
param([switch]$Stop)
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain

$backend = Join-Path $Root 'backend-go'
if ($Stop) {
    foreach ($n in 'mock-real', 'mock-review') {
        $procId = Read-Pid $n
        if ($procId -gt 0) { Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue; Write-Host "$n stopped" }
    }
    return
}
Ensure-Dir (Join-Path $LocalDir 'logs')
Set-Location $backend
& (Join-Path $GoBin 'go.exe') build -o (Join-Path $LocalDir 'mockorigin.exe') ./cmd/mockorigin
foreach ($pair in @(@('mock-real', 'real', '127.0.0.1:9101'), @('mock-review', 'review', '127.0.0.1:9102'))) {
    $name, $kind, $addr = $pair
    $log = Join-Path $LocalDir "logs\$name.log"
    $p = Start-Process -FilePath (Join-Path $LocalDir 'mockorigin.exe') -ArgumentList @('-kind', $kind, '-addr', $addr) -WindowStyle Hidden -PassThru -RedirectStandardOutput $log -RedirectStandardError "$log.err"
    Save-Pid $name $p.Id
    Write-Host "$name ($kind) on http://$addr pid $($p.Id)"
}
