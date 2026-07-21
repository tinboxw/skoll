param(
    [Parameter(Mandatory = $true)]
    [string[]]$ReportPath
)

$ErrorActionPreference = "Stop"
$failures = @()
foreach ($path in $ReportPath) {
    if (-not (Test-Path -LiteralPath $path)) {
        $failures += "Missing metrics evidence: $path"
        continue
    }
    $json = [System.IO.File]::ReadAllText($path, [System.Text.Encoding]::UTF8)
    $report = $json | ConvertFrom-Json
    if ($report.schemaVersion -ne "skoll.h5-job-soak.v1") {
        $failures += "Unsupported evidence schema: $path"
    }
    if ($report.locale -ne "zh-CN") {
        $failures += "Default locale is not zh-CN: $path"
    }
    if (-not $report.raceEnabled) {
        $failures += "Race verification is not recorded: $path"
    }
    if (-not $report.passed) {
        $failures += "Report did not pass: $path"
    }
    foreach ($scenario in $report.scenarios) {
        if (-not $scenario.passed -or $scenario.alerts.Count -gt 0) {
            $failures += "Scenario did not pass: $($scenario.name)"
        }
        if ($scenario.failures -gt $scenario.thresholds.maxFailures) {
            $failures += "Failure count exceeds threshold: $($scenario.name)"
        }
        if ($scenario.duplicateSideEffects -gt $scenario.thresholds.maxDuplicateSideEffects) {
            $failures += "Duplicate side effects exceed threshold: $($scenario.name)"
        }
    }
}

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { Write-Error $_ }
    exit 1
}

Write-Host "H5-03 metrics audit passed: $($ReportPath.Count) reports."
