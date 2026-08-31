$ErrorActionPreference = 'Stop'

$repo = "Shashwat-CODING/adx"
$installDir = "$env:LOCALAPPDATA\Programs\adx"

Write-Host "⚡ Installing ADX for Windows..." -ForegroundColor Cyan

# Create install directory
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Resolve latest release
$latestReleaseUrl = "https://api.github.com/repos/$repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $latestReleaseUrl -UseBasicParsing
    $tag = $release.tag_name
} catch {
    $tag = "v1.0.0"
}

$zipUrl = "https://github.com/$repo/releases/download/$tag/adx_${tag}_windows_amd64.zip"
$tempZip = Join-Path $env:TEMP "adx.zip"

Write-Host "Downloading $zipUrl..." -ForegroundColor Yellow
Invoke-WebRequest -Uri $zipUrl -OutFile $tempZip -UseBasicParsing

Write-Host "Extracting..." -ForegroundColor Yellow
Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
Remove-Item $tempZip -Force

# Flatten if inside subdirectory
$nested = Get-ChildItem -Path $installDir -Filter "adx.exe" -Recurse | Select-Object -First 1
if ($nested -and $nested.DirectoryName -ne $installDir) {
    Move-Item -Path $nested.FullName -Destination $installDir -Force
}

# Add to User PATH if not present
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path += ";$installDir"
    Write-Host "Added $installDir to User PATH." -ForegroundColor Green
}

Write-Host "✔ ADX successfully installed to $installDir\adx.exe!" -ForegroundColor Green
Write-Host "Restart your terminal and run 'adx doctor' or 'adx' to begin." -ForegroundColor Cyan
