# 停止由 start-*.ps1 启动的进程（MySQL、Redis、后台 Go 服务）。
. (Join-Path $PSScriptRoot 'common.ps1')

foreach ($name in @('admin', 'gateway', 'worker', 'scheduler', 'redis', 'mysql')) {
    $procId = Read-Pid $name
    if ($procId -gt 0) {
        $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
        if ($proc) {
            if ($name -eq 'mysql') {
                # 让 MySQL 正常落盘
                Use-Toolchain
                & (Join-Path $MysqlBin 'mysqladmin.exe') -uroot --host=127.0.0.1 --port=$MysqlPort shutdown 2>$null
                Start-Sleep -Seconds 2
                if (Get-Process -Id $procId -ErrorAction SilentlyContinue) { Stop-Process -Id $procId -Force }
            } else {
                Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
                # go run 会派生子进程，连同端口占用者一起清理
            }
            Write-Host "stopped $name (pid $procId)"
        }
        Remove-Item (Join-Path $PidDir "$name.pid") -ErrorAction SilentlyContinue
    }
}

# 兜底：清理仍占用本项目端口的进程
foreach ($port in @(8080, 8090, 9080, 9090, 9091, 9092)) {
    Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue | ForEach-Object {
        Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue
        Write-Host "freed port $port"
    }
}
