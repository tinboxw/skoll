param(
    [ValidateSet("build", "package", "verify", "install", "dev")]
    [string]$Action = "package",
    [string]$DistDir = "",
    [string]$PluginsRoot = ""
)

$ErrorActionPreference = "Stop"
$DistDir = if ($DistDir) { $DistDir } else { Join-Path $PSScriptRoot "dist" }
$PluginsRoot = if ($PluginsRoot) { $PluginsRoot } else { Join-Path $PSScriptRoot ".skoll-dev" }
$RepoRoot = if ($env:SKOLL_REPO_ROOT) { (Resolve-Path $env:SKOLL_REPO_ROOT).Path } else { (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path }
$Artifact = Join-Path $DistDir "pharma_oa-0.10.0.zip"
$Checksum = "$Artifact.sha256"
$Backend = Join-Path $PSScriptRoot "backend/bin/pharma_oa-server.exe"

function Build-Plugin {
    New-Item -ItemType Directory -Force (Split-Path $Backend) | Out-Null
    Push-Location (Join-Path $PSScriptRoot "frontend")
    try {
        npm run build
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
    Push-Location $PSScriptRoot
    try {
        go build -o $Backend ./backend
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
}

function Invoke-SkollPlugin([string[]]$ToolArgs) {
    Push-Location $RepoRoot
    try {
        & go run ./cmd/skoll-plugin @ToolArgs
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
}

switch ($Action) {
    "build" { Build-Plugin }
    "package" { Build-Plugin; Invoke-SkollPlugin @("package", $PSScriptRoot, $DistDir) }
    "verify" { Invoke-SkollPlugin @("verify-package", $Artifact, $Checksum) }
    "install" { Invoke-SkollPlugin @("install-package", $Artifact, $Checksum, $PluginsRoot) }
    "dev" { Build-Plugin; Invoke-SkollPlugin @("dev", $PSScriptRoot, $DistDir, $PluginsRoot) }
}
