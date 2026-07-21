param(
    [ValidateRange(1, 10)]
    [int]$BenchmarkCount = 3,
    [ValidateRange(10, 10000)]
    [int]$BenchmarkIterations = 200,
    [string]$OutputPath = "docs/refactor/current/evidence/h5-04/performance-baseline.json",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$resolvedOutput = if ([System.IO.Path]::IsPathRooted($OutputPath)) { $OutputPath } else { Join-Path $repoRoot $OutputPath }
$h501Path = Join-Path $repoRoot "docs/refactor/current/evidence/h5-01/mysql-query-benchmark.json"
$h502Path = Join-Path $repoRoot "docs/refactor/current/evidence/h5-02/browser-matrix.json"
$h503Path = Join-Path $repoRoot "docs/refactor/current/evidence/h5-03/h5-03-soak-summary.json"

function Read-Utf8Json([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) { throw "Missing source evidence: $Path" }
    return [System.IO.File]::ReadAllText($Path, [System.Text.Encoding]::UTF8) | ConvertFrom-Json
}

function Get-Median([double[]]$Values) {
    $ordered = @($Values | Sort-Object)
    if ($ordered.Count -eq 0) { return 0 }
    $middle = [math]::Floor($ordered.Count / 2)
    if ($ordered.Count % 2 -eq 1) { return [double]$ordered[$middle] }
    return ([double]$ordered[$middle - 1] + [double]$ordered[$middle]) / 2
}

function Get-GzipLength([string]$Path) {
    $inputBytes = [System.IO.File]::ReadAllBytes($Path)
    $output = New-Object System.IO.MemoryStream
    $gzip = New-Object System.IO.Compression.GZipStream($output, [System.IO.Compression.CompressionMode]::Compress, $true)
    $gzip.Write($inputBytes, 0, $inputBytes.Length)
    $gzip.Dispose()
    $length = $output.Length
    $output.Dispose()
    return [int64]$length
}

function Get-BundleMetric([string]$Pattern, [int64]$MaxRawBytes, [int64]$MaxGzipBytes) {
    $files = @(Get-ChildItem -LiteralPath (Join-Path $repoRoot "web/dist/assets") -Filter $Pattern -File)
    if ($files.Count -ne 1) { throw "Expected one bundle matching $Pattern, found $($files.Count)" }
    $raw = [int64]$files[0].Length
    $gzip = Get-GzipLength $files[0].FullName
    return [ordered]@{
        name = $files[0].Name
        rawBytes = $raw
        gzipBytes = $gzip
        maxRawBytes = $MaxRawBytes
        maxGzipBytes = $MaxGzipBytes
        passed = [bool]($raw -le $MaxRawBytes -and $gzip -le $MaxGzipBytes)
    }
}

$h501 = Read-Utf8Json $h501Path
$h502 = Read-Utf8Json $h502Path
$h503 = Read-Utf8Json $h503Path
if (-not $h501.passed -or -not $h502.passed -or -not $h503.passed) {
    throw "H5-01, H5-02, or H5-03 source evidence is not passing"
}

Push-Location $repoRoot
try {
    $benchmarkPattern = 'Benchmark(PharmaOAEmployeePagedQuery|PharmaOACustomerScopedQuery|LocalCacheSetGet)$'
    Write-Host "Running allocation benchmarks..."
    $benchmarkOutput = @(& go test ./tests/benchmark -run '^$' -bench $benchmarkPattern -benchmem "-benchtime=$($BenchmarkIterations)x" -count $BenchmarkCount 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "Allocation benchmarks failed`n$($benchmarkOutput -join [Environment]::NewLine)" }

    $samples = @{}
    $pattern = '^(Benchmark\S+)-\d+\s+\d+\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op$'
    foreach ($line in $benchmarkOutput) {
        if ($line -match $pattern) {
            $name = $Matches[1]
            if (-not $samples.ContainsKey($name)) { $samples[$name] = @() }
            $samples[$name] += [pscustomobject]@{ nsPerOp = [double]$Matches[2]; bytesPerOp = [int64]$Matches[3]; allocationsPerOp = [int64]$Matches[4] }
        }
    }

    $budgets = [ordered]@{
        BenchmarkLocalCacheSetGet = [ordered]@{ maxNsPerOp = 5000; maxBytesPerOp = 512; maxAllocationsPerOp = 8 }
        BenchmarkPharmaOAEmployeePagedQuery = [ordered]@{ maxNsPerOp = 40000000; maxBytesPerOp = 2097152; maxAllocationsPerOp = 25000 }
        BenchmarkPharmaOACustomerScopedQuery = [ordered]@{ maxNsPerOp = 30000000; maxBytesPerOp = 1572864; maxAllocationsPerOp = 18000 }
    }
    $benchmarks = @()
    foreach ($name in $budgets.Keys) {
        $values = @($samples[$name])
        if ($values.Count -ne $BenchmarkCount) { throw "Expected $BenchmarkCount samples for $name, found $($values.Count)" }
        $budget = $budgets[$name]
        $nsValues = [double[]]@($values | ForEach-Object { $_.nsPerOp })
        $maxNs = [double](($nsValues | Measure-Object -Maximum).Maximum)
        $maxBytes = [int64](($values.bytesPerOp | Measure-Object -Maximum).Maximum)
        $maxAllocs = [int64](($values.allocationsPerOp | Measure-Object -Maximum).Maximum)
        $benchmarks += [ordered]@{
            name = $name
            samples = $values.Count
            nsPerOpMedian = Get-Median $nsValues
            nsPerOpMax = $maxNs
            bytesPerOpMax = $maxBytes
            allocationsPerOpMax = $maxAllocs
            thresholds = $budget
            passed = [bool]($maxNs -le $budget.maxNsPerOp -and $maxBytes -le $budget.maxBytesPerOp -and $maxAllocs -le $budget.maxAllocationsPerOp)
        }
    }

    if (-not $SkipBuild) {
        Write-Host "Building frontend bundles..."
        & npm --prefix web run build
        if ($LASTEXITCODE -ne 0) { throw "Frontend build failed" }
    } elseif (-not (Test-Path -LiteralPath (Join-Path $repoRoot "web/dist/assets"))) {
        throw "web/dist/assets is missing; run without -SkipBuild"
    }

    $bundles = @(
        (Get-BundleMetric "DataTable-*.js" 65536 20480),
        (Get-BundleMetric "el-pagination-*.js" 16384 6144)
    )
    $queryBudgetsPass = @($h501.results | Where-Object { -not $_.passed -or $_.p95Millis -gt $_.p95BudgetMillis }).Count -eq 0
    $benchmarkPass = @($benchmarks | Where-Object { -not $_.passed }).Count -eq 0
    $bundlePass = @($bundles | Where-Object { -not $_.passed }).Count -eq 0
    $goVersion = (& go version).Trim()
    $nodeVersion = (& node --version).Trim()
    $report = [ordered]@{
        schemaVersion = "skoll.h5-performance-baseline.v1"
        locale = "zh-CN"
        generatedAt = (Get-Date).ToUniversalTime().ToString("o")
        environment = [ordered]@{
            os = [System.Environment]::OSVersion.VersionString
            processorCount = [System.Environment]::ProcessorCount
            go = $goVersion
            node = $nodeVersion
            mysql = $h501.version
        }
        sourceEvidence = [ordered]@{
            database = "../h5-01/mysql-query-benchmark.json"
            largeList = "../h5-02/browser-matrix.json"
            jobSoak = "../h5-03/h5-03-soak-summary.json"
        }
        databaseQueries = $h501.results
        allocationBenchmarks = $benchmarks
        bundles = $bundles
        largeListPassed = [bool]$h502.passed
        jobSoakPassed = [bool]$h503.passed
        passed = [bool]($queryBudgetsPass -and $benchmarkPass -and $bundlePass -and $h502.passed -and $h503.passed)
    }
    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedOutput) -Force | Out-Null
    $report | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $resolvedOutput -Encoding utf8
    if (-not $report.passed) { throw "One or more H5 performance budgets failed" }
    Write-Host "H5 performance baseline passed: $resolvedOutput"
} finally {
    Pop-Location
}
