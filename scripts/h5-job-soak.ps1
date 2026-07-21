param(
    [ValidateRange(1, 10000)]
    [int]$Iterations = 100,
    [string]$EvidenceDirectory = "docs/refactor/current/evidence/h5-03"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$evidenceRoot = if ([System.IO.Path]::IsPathRooted($EvidenceDirectory)) {
    $EvidenceDirectory
} else {
    Join-Path $repoRoot $EvidenceDirectory
}
$pharmaReport = Join-Path $evidenceRoot "pharma-job-soak.json"
$eventReport = Join-Path $evidenceRoot "business-event-soak.json"
$summaryReport = Join-Path $evidenceRoot "h5-03-soak-summary.json"

New-Item -ItemType Directory -Path $evidenceRoot -Force | Out-Null
$env:SKOLL_H5_SOAK_ITERATIONS = $Iterations.ToString()
$env:SKOLL_H5_RACE_ENABLED = "true"
$env:SKOLL_H5_PHARMA_SOAK_OUTPUT = $pharmaReport
$env:SKOLL_H5_EVENT_SOAK_OUTPUT = $eventReport

Push-Location $repoRoot
try {
    Write-Host "Running pharma job race/soak ($Iterations iterations)..."
    go test -race ./internal/service/pharmaoa -run '^TestH5PharmaJobSoak$' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw "Pharma job race/soak failed" }

    Write-Host "Running plugin event race/soak ($Iterations iterations)..."
    go test -race ./internal/event -run '^(TestH5BusinessEventSoak|TestBusinessEventBusMovesExhaustedRetriesToDeadLetter)$' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw "Plugin event race/soak failed" }

    & (Join-Path $PSScriptRoot "h5-job-metrics-audit.ps1") -ReportPath @($pharmaReport, $eventReport)
    if ($LASTEXITCODE -ne 0) { throw "Job metrics audit failed" }

    $pharma = [System.IO.File]::ReadAllText($pharmaReport, [System.Text.Encoding]::UTF8) | ConvertFrom-Json
    $events = [System.IO.File]::ReadAllText($eventReport, [System.Text.Encoding]::UTF8) | ConvertFrom-Json
    $summary = [ordered]@{
        schemaVersion = "skoll.h5-job-soak-summary.v1"
        locale = "zh-CN"
        generatedAt = (Get-Date).ToUniversalTime().ToString("o")
        iterations = $Iterations
        raceEnabled = $true
        passed = [bool]($pharma.passed -and $events.passed)
        reports = @(
            [ordered]@{ name = "pharma jobs"; path = "pharma-job-soak.json"; scenarios = $pharma.scenarios }
            [ordered]@{ name = "plugin business events"; path = "business-event-soak.json"; scenarios = $events.scenarios }
        )
    }
    $summary | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $summaryReport -Encoding utf8
    Write-Host "H5-03 race/soak acceptance passed. Evidence: $evidenceRoot"
} finally {
    Pop-Location
    Remove-Item Env:SKOLL_H5_SOAK_ITERATIONS -ErrorAction SilentlyContinue
    Remove-Item Env:SKOLL_H5_RACE_ENABLED -ErrorAction SilentlyContinue
    Remove-Item Env:SKOLL_H5_PHARMA_SOAK_OUTPUT -ErrorAction SilentlyContinue
    Remove-Item Env:SKOLL_H5_EVENT_SOAK_OUTPUT -ErrorAction SilentlyContinue
}
