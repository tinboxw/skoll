param(
    [int]$Count = 1
)

$ErrorActionPreference = "Stop"

Write-Host "=== Pharma OA inventory end-to-end smoke ==="
$goArgs = @("test", "./tests/integration", "-run", "^TestPharmaOAInventoryEndToEndSmoke$", "-count", ([string]$Count), "-v")
& go @goArgs
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host "Pharma OA inventory end-to-end smoke passed."
