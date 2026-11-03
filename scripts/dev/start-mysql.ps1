# 启动本项目专用的 MySQL（端口 3307，数据目录 .local/mysql）。首次运行自动初始化并创建库与账号。
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain

if (Test-PortListening $MysqlPort) {
    Write-Host "MySQL already listening on $MysqlPort"
    exit 0
}

Ensure-Dir $MysqlLogDir
$firstRun = -not (Test-Path (Join-Path $MysqlDataDir 'mysql'))
if ($firstRun) {
    Write-Host "Initializing MySQL data dir at $MysqlDataDir"
    Ensure-Dir (Split-Path $MysqlDataDir)
    & (Join-Path $MysqlBin 'mysqld.exe') --initialize-insecure --datadir="$MysqlDataDir" --log-error="$MysqlLogDir\init.err"
    if ($LASTEXITCODE -ne 0) { throw "mysqld --initialize failed, see $MysqlLogDir\init.err" }
}

$args = @(
    "--datadir=$MysqlDataDir",
    "--port=$MysqlPort",
    "--bind-address=127.0.0.1",
    "--log-error=$MysqlLogDir\mysqld.err",
    "--mysqlx=OFF",
    "--character-set-server=utf8mb4",
    "--collation-server=utf8mb4_unicode_ci",
    "--default-time-zone=+08:00",
    "--sql-mode=STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION",
    "--innodb-buffer-pool-size=256M",
    "--console"
)
$p = Start-Process -FilePath (Join-Path $MysqlBin 'mysqld.exe') -ArgumentList $args -WindowStyle Hidden -PassThru
Save-Pid 'mysql' $p.Id

if (-not (Wait-Port $MysqlPort 60)) { throw "MySQL did not start, see $MysqlLogDir\mysqld.err" }
Write-Host "MySQL started (pid $($p.Id)) on 127.0.0.1:$MysqlPort"

if ($firstRun) {
    Write-Host "Creating database $MysqlDb and user $MysqlUser"
    $sql = @"
CREATE DATABASE IF NOT EXISTS ``$MysqlDb`` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '$MysqlUser'@'127.0.0.1' IDENTIFIED BY '$MysqlPassword';
CREATE USER IF NOT EXISTS '$MysqlUser'@'localhost' IDENTIFIED BY '$MysqlPassword';
GRANT ALL PRIVILEGES ON ``$MysqlDb``.* TO '$MysqlUser'@'127.0.0.1';
GRANT ALL PRIVILEGES ON ``$MysqlDb``.* TO '$MysqlUser'@'localhost';
FLUSH PRIVILEGES;
"@
    $sql | & (Join-Path $MysqlBin 'mysql.exe') -uroot --host=127.0.0.1 --port=$MysqlPort
    if ($LASTEXITCODE -ne 0) { throw 'failed to create database/user' }
}
