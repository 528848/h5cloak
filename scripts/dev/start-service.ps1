# 以开发模式启动一个 Go 服务：admin | gateway | worker | scheduler
param(
    [Parameter(Mandatory = $true)][ValidateSet('admin', 'gateway', 'worker', 'scheduler')][string]$Name,
    [switch]$Background
)
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain

$backend = Join-Path $Root 'backend-go'
Set-Location $backend

if ($Background) {
    Ensure-Dir (Join-Path $LocalDir 'logs')
    $log = Join-Path $LocalDir "logs\$Name.log"
    $p = Start-Process -FilePath (Join-Path $GoBin 'go.exe') -ArgumentList @('run', "./cmd/$Name") -WorkingDirectory $backend -WindowStyle Hidden -PassThru -RedirectStandardOutput $log -RedirectStandardError "$log.err"
    Save-Pid $Name $p.Id
    Write-Host "$Name started in background (pid $($p.Id)), log: $log"
} else {
    & (Join-Path $GoBin 'go.exe') run "./cmd/$Name"
}
