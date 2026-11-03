# 启动前端开发服务器：admin（5173）或 merchant（5174）
param(
    [Parameter(Mandatory = $true)][ValidateSet('admin', 'merchant')][string]$Portal
)
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain
Set-Location (Join-Path $Root 'frontend')
npm run "dev:$Portal"
