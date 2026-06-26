param(
    [int]$Port = 34115,
    [switch]$SkipBrowser,
    [switch]$ForceBuild
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$frontendDir = Join-Path $repoRoot "frontend"
$distIndex = Join-Path $frontendDir "dist\index.html"
$nodeModules = Join-Path $frontendDir "node_modules"
$nodeModulesLock = Join-Path $nodeModules ".package-lock.json"
$packageLock = Join-Path $frontendDir "package-lock.json"

function Test-Command($Name) {
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Get-NewestWriteTime($Path) {
    if (-not (Test-Path $Path)) {
        return [datetime]::MinValue
    }
    $items = Get-ChildItem -Path $Path -Recurse -File
    if (-not $items) {
        return [datetime]::MinValue
    }
    return ($items | Sort-Object LastWriteTime -Descending | Select-Object -First 1).LastWriteTime
}

if (-not (Test-Command "go")) {
    throw "Go was not found on PATH."
}
if (-not (Test-Command "npm")) {
    throw "npm was not found on PATH."
}

Push-Location $repoRoot
try {
    if ((-not (Test-Path $nodeModules)) -or (Test-Path $packageLock -and (-not (Test-Path $nodeModulesLock) -or (Get-Item $packageLock).LastWriteTime -gt (Get-Item $nodeModulesLock).LastWriteTime))) {
        Write-Host "Installing frontend dependencies..."
        Push-Location $frontendDir
        try {
            npm ci
        } finally {
            Pop-Location
        }
    }

    $needsBuild = $ForceBuild -or -not (Test-Path $distIndex)
    if (-not $needsBuild) {
        $distTime = (Get-Item $distIndex).LastWriteTime
        $srcTime = Get-NewestWriteTime (Join-Path $frontendDir "src")
        $publicTime = Get-NewestWriteTime (Join-Path $frontendDir "public")
        $packageTime = (Get-Item $packageLock).LastWriteTime
        $needsBuild = $srcTime -gt $distTime -or $publicTime -gt $distTime -or $packageTime -gt $distTime
    }

    if ($needsBuild) {
        Write-Host "Building frontend..."
        Push-Location $frontendDir
        try {
            npm run build
        } finally {
            Pop-Location
        }
    }

    $url = "http://127.0.0.1:$Port"
    Write-Host "Starting AgentMeter at $url"
    if (-not $SkipBrowser) {
        Start-Job -ScriptBlock {
            param($Target)
            Start-Sleep -Seconds 2
            Start-Process $Target
        } -ArgumentList $url | Out-Null
    }
    go run . -http ":$Port"
} finally {
    Pop-Location
}
