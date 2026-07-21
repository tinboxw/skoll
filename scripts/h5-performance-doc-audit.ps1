param(
    [string]$DocumentPath = "docs/refactor/current/performance_capacity_baseline_2026-07-21.md",
    [string]$EvidencePath = "docs/refactor/current/evidence/h5-04/performance-baseline.json"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$document = Join-Path $repoRoot $DocumentPath
$evidence = Join-Path $repoRoot $EvidencePath
if (-not (Test-Path -LiteralPath $document)) { throw "Missing baseline document: $document" }
if (-not (Test-Path -LiteralPath $evidence)) { throw "Missing baseline evidence: $evidence" }

$text = [System.IO.File]::ReadAllText($document, [System.Text.Encoding]::UTF8)
$requiredTokens = @(
    "# Skoll",
    "zh-CN",
    "P50",
    "P95",
    "P99",
    "B/op",
    "allocs/op",
    "DataTable",
    "dead_letter",
    "BenchmarkCount",
    "SKOLL_BENCHMARK_MYSQL_DSN",
    "## English Summary"
)
foreach ($token in $requiredTokens) {
    if (-not $text.Contains($token)) { throw "Missing required document token: $token" }
}
if (@([regex]::Matches($text, '(?m)^## ')).Count -lt 12) { throw "Baseline document has fewer than 12 required sections" }
if ($text -match 'TODO|TBD') { throw "Baseline document contains an unresolved placeholder" }

$json = [System.IO.File]::ReadAllText($evidence, [System.Text.Encoding]::UTF8) | ConvertFrom-Json
if ($json.schemaVersion -ne "skoll.h5-performance-baseline.v1" -or $json.locale -ne "zh-CN" -or -not $json.passed) {
    throw "Baseline evidence schema, locale, or pass state is invalid"
}
if (@($json.databaseQueries).Count -ne 6 -or @($json.allocationBenchmarks).Count -ne 3 -or @($json.bundles).Count -ne 2) {
    throw "Baseline evidence has an unexpected metric count"
}
Write-Host "H5-04 document audit passed."
