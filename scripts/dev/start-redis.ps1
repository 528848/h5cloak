# 启动本项目专用的 Redis（端口 6380，数据目录 .local/redis）。
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain

if (Test-PortListening $RedisPort) {
    Write-Host "Redis already listening on $RedisPort"
    exit 0
}

Ensure-Dir $RedisDataDir
$args = @(
    '--port', "$RedisPort",
    '--bind', '127.0.0.1',
    '--dir', "$RedisDataDir",
    '--appendonly', 'no',
    '--save', '""',
    '--maxmemory', '512mb',
    '--maxmemory-policy', 'allkeys-lru'
)
$p = Start-Process -FilePath (Join-Path $RedisBin 'redis-server.exe') -ArgumentList $args -WindowStyle Hidden -PassThru
Save-Pid 'redis' $p.Id

if (-not (Wait-Port $RedisPort 20)) { throw 'Redis did not start' }
Write-Host "Redis started (pid $($p.Id)) on 127.0.0.1:$RedisPort"
