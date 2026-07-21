param(
    [int]$Rows = 10000,
    [int]$Iterations = 100,
    [string]$Output = "docs/refactor/current/evidence/h5-01/mysql-query-benchmark.json",
    [ValidateSet("zh-CN", "en-US")]
    [string]$Locale = "zh-CN"
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($env:SKOLL_BENCHMARK_MYSQL_DSN)) {
    throw "SKOLL_BENCHMARK_MYSQL_DSN is required."
}

$startMessage = if ($Locale -eq "en-US") {
    "Running Pharma OA MySQL query and index benchmark..."
} else {
    [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("5q2j5Zyo6L+Q6KGM5Yy76I2vIE9BIE15U1FMIOafpeivouS4jue0ouW8leWfuuWHhi4uLg=="))
}
$passedMessage = if ($Locale -eq "en-US") {
    "Pharma OA MySQL query and index benchmark passed."
} else {
    [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("5Yy76I2vIE9BIE15U1FMIOafpeivouS4jue0ouW8leWfuuWHhumAmui/h+OAgg=="))
}

Write-Host $startMessage
go run ./cmd/skoll-db-benchmark --allow-write-fixtures --rows $Rows --iterations $Iterations --output $Output
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
Write-Host $passedMessage
