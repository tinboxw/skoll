param(
    [string]$Package = "./internal/plugin",
    [string]$Run = "TestPharmaOA",
    [int]$Count = 1
)

$ErrorActionPreference = "Stop"

Write-Host "=== Pharma OA plugin lifecycle smoke ==="
$goArgs = @("test", $Package, "-run", $Run, "-count", ([string]$Count), "-v")
& go @goArgs
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host "Pharma OA plugin lifecycle smoke passed."
