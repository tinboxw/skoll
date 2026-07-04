param(
    [string]$Package = "./internal/testing/workflowaudit",
    [string]$Run = "TestWorkflowAuditReplaySmoke",
    [int]$Count = 1
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $repoRoot
try {
    Write-Host "=== Workflow audit replay smoke ===" -ForegroundColor Cyan
    $goArgs = @("test", $Package, "-run", $Run, "-count", ([string]$Count), "-v")
    & go @goArgs
    if ($LASTEXITCODE -ne 0) {
        throw "workflow audit replay smoke failed"
    }
    Write-Host "Workflow audit replay smoke passed." -ForegroundColor Green
}
finally {
    Pop-Location
}
