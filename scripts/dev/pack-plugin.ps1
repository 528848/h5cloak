# Pack the WordPress plugin cloak-attribution into a zip for the merchant
# "download plugin" button. Output: frontend/public/downloads/cloak-attribution.zip
# (vite build copies it into dist, then it ships with a release).
# Re-run this script whenever the plugin code changes.
# ASCII-only on purpose: PowerShell 5.1 reads .ps1 as ANSI and would corrupt
# non-ASCII characters, breaking the script.
$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$src = Join-Path $root 'wordpress-plugin\cloak-attribution'
$dstDir = Join-Path $root 'frontend\public\downloads'
$zip = Join-Path $dstDir 'cloak-attribution.zip'
if (-not (Test-Path $src)) { throw "plugin dir not found: $src" }
if (-not (Test-Path $dstDir)) { New-Item -ItemType Directory -Path $dstDir -Force | Out-Null }
if (Test-Path $zip) { Remove-Item $zip -Force }

Add-Type -AssemblyName System.IO.Compression | Out-Null
Add-Type -AssemblyName System.IO.Compression.FileSystem | Out-Null
$fs = [IO.File]::Open($zip, [IO.FileMode]::CreateNew)
$arch = New-Object IO.Compression.ZipArchive($fs, [IO.Compression.ZipArchiveMode]::Create)
$base = Split-Path $src -Parent
# Use forward-slash paths so the top-level dir is cloak-attribution/, which
# WordPress requires to recognize the upload.
Get-ChildItem -Recurse -File $src | ForEach-Object {
    $rel = $_.FullName.Substring($base.Length + 1).Replace('\', '/')
    $entry = $arch.CreateEntry($rel, [IO.Compression.CompressionLevel]::Optimal)
    $es = $entry.Open()
    $bytes = [IO.File]::ReadAllBytes($_.FullName)
    $es.Write($bytes, 0, $bytes.Length)
    $es.Dispose()
}
$arch.Dispose()
$fs.Dispose()
Write-Host ("plugin packed: {0} ({1:N0} bytes)" -f $zip, (Get-Item $zip).Length)
