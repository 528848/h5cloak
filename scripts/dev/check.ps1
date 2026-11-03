# 质量门禁：go vet、go test（含覆盖率）、golangci-lint（存在时）、前端 lint 与构建（-Frontend）。
param(
    [switch]$Frontend,
    [switch]$SkipLint
)
. (Join-Path $PSScriptRoot 'common.ps1')
Use-Toolchain

$failed = $false
$backend = Join-Path $Root 'backend-go'
Set-Location $backend

Write-Host '== go vet =='
& go vet ./...
if ($LASTEXITCODE -ne 0) { $failed = $true }

Write-Host '== go test =='
& go test ./... -coverprofile=coverage.out -covermode=atomic
if ($LASTEXITCODE -ne 0) { $failed = $true }
if (Test-Path coverage.out) {
    $total = (& go tool cover -func=coverage.out | Select-String -Pattern 'total:' | ForEach-Object { ($_ -split '\s+')[-1] })
    Write-Host "coverage total: $total"
}

if (-not $SkipLint) {
    $lint = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if ($lint) {
        Write-Host '== golangci-lint =='
        & golangci-lint run ./...
        if ($LASTEXITCODE -ne 0) { $failed = $true }
    } else {
        Write-Host 'golangci-lint not installed, skipped (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)'
    }
}

Write-Host '== gofmt =='
$unformatted = & gofmt -l ./cmd ./internal
if ($unformatted) {
    Write-Host "unformatted files:`n$unformatted"
    $failed = $true
}

if ($Frontend) {
    Set-Location (Join-Path $Root 'frontend')
    # Windows PowerShell 调用 npm.ps1 时会把命令切成 "pm"，改走 npm.cmd
    $npm = Join-Path $NodeBin 'npm.cmd'
    Write-Host '== frontend lint =='
    & $npm run lint:eslint
    if ($LASTEXITCODE -ne 0) { $failed = $true }
    Write-Host '== frontend build (admin + merchant) =='
    & $npm run build
    if ($LASTEXITCODE -ne 0) { $failed = $true }
}

if ($failed) {
    Write-Host 'CHECK FAILED' -ForegroundColor Red
    exit 1
}
Write-Host 'CHECK PASSED' -ForegroundColor Green
