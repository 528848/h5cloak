# 本地开发公共配置。所有路径集中在这里，换环境只改这一个文件。
$ErrorActionPreference = 'Stop'

$script:Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$script:LocalDir = Join-Path $Root '.local'

# 便携工具链。当前复用 D:\pay-virtual\.runtime 下的二进制，数据目录与端口与其它项目隔离。
$script:RuntimeDir = 'D:\pay-virtual\.runtime'
$script:GoBin = Join-Path $RuntimeDir 'go\bin'
$script:NodeBin = Join-Path $RuntimeDir 'node'
$script:MysqlBin = Join-Path $RuntimeDir 'mysql\bin'
$script:RedisBin = Join-Path $RuntimeDir 'redis\Redis-7.4.2-Windows-x64-msys2'

# 本项目独占的端口与数据目录
$script:MysqlPort = 3307
$script:MysqlDataDir = Join-Path $LocalDir 'mysql\data'
$script:MysqlLogDir = Join-Path $LocalDir 'mysql\log'
$script:MysqlDb = 'cloak_v2'
$script:MysqlUser = 'cloak'
$script:MysqlPassword = 'cloak'

$script:RedisPort = 6380
$script:RedisDataDir = Join-Path $LocalDir 'redis'

$script:PidDir = Join-Path $LocalDir 'pids'

function Use-Toolchain {
    $env:PATH = "$GoBin;$NodeBin;$MysqlBin;$RedisBin;" + $env:PATH
    $env:GOFLAGS = '-mod=mod'
    # 固定使用本地 Go 版本，避免依赖升级时自动下载新工具链
    $env:GOTOOLCHAIN = 'local'
}

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path $Path)) { New-Item -ItemType Directory -Path $Path -Force | Out-Null }
}

function Save-Pid([string]$Name, [int]$ProcessId) {
    Ensure-Dir $PidDir
    Set-Content -Path (Join-Path $PidDir "$Name.pid") -Value $ProcessId
}

function Read-Pid([string]$Name) {
    $f = Join-Path $PidDir "$Name.pid"
    if (Test-Path $f) { return [int](Get-Content $f -ErrorAction SilentlyContinue) }
    return 0
}

function Test-PortListening([int]$Port) {
    return [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Wait-Port([int]$Port, [int]$TimeoutSec = 30) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (Test-PortListening $Port) { return $true }
        Start-Sleep -Milliseconds 500
    }
    return $false
}
