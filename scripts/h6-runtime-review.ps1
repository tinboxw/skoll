param(
    [string]$OutputPath = "docs/refactor/current/evidence/h6-03/runtime-review.json"
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$resolvedOutput = if ([System.IO.Path]::IsPathRooted($OutputPath)) { $OutputPath } else { Join-Path $repoRoot $OutputPath }
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) "skoll-h6-runtime-$PID"
$binary = Join-Path $tempRoot "skoll.exe"
$stdout = Join-Path $tempRoot "stdout.log"
$stderr = Join-Path $tempRoot "stderr.log"
$process = $null
$startedAt = Get-Date
$saved = @{
    SKOLL_SERVER_ADDRESS = $env:SKOLL_SERVER_ADDRESS
    SKOLL_API_BASE_PREFIX = $env:SKOLL_API_BASE_PREFIX
    SKOLL_STORE_MODE = $env:SKOLL_STORE_MODE
    SKOLL_SECURITY_JWT_SECRET = $env:SKOLL_SECURITY_JWT_SECRET
    SKOLL_LOG_DIR = $env:SKOLL_LOG_DIR
    SKOLL_LOG_FILE = $env:SKOLL_LOG_FILE
}

function Get-FreeTCPPort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port
    $listener.Stop()
    return $port
}

function Wait-Endpoint([string]$URL) {
    $deadline = (Get-Date).AddSeconds(45)
    while ((Get-Date) -lt $deadline) {
        if ($process.HasExited) { throw "Skoll exited before $URL became available" }
        try { return Invoke-RestMethod -Uri $URL -TimeoutSec 2 } catch { Start-Sleep -Milliseconds 250 }
    }
    throw "Timed out waiting for $URL"
}

New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null
try {
    Push-Location $repoRoot
    try { & go build -trimpath -o $binary ./cmd/skoll } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $binary)) { throw "Runtime review build failed" }

    $port = Get-FreeTCPPort
    $env:SKOLL_SERVER_ADDRESS = "127.0.0.1:$port"
    $env:SKOLL_API_BASE_PREFIX = "/skoll"
    $env:SKOLL_STORE_MODE = "memory"
    $env:SKOLL_SECURITY_JWT_SECRET = "h6-runtime-review-secret-32-characters"
    $env:SKOLL_LOG_DIR = Join-Path $tempRoot "logs"
    $env:SKOLL_LOG_FILE = "skoll.log"
    $process = Start-Process -FilePath $binary -WorkingDirectory $repoRoot -WindowStyle Hidden -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    $base = "http://127.0.0.1:$port/skoll"
    $health = Wait-Endpoint "$base/health"
    $ready = Wait-Endpoint "$base/ready"

    $openAPIResponse = Invoke-WebRequest -UseBasicParsing -Uri "$base/docs/openapi.yaml" -TimeoutSec 10
    $openAPI = if ($openAPIResponse.Content -is [byte[]]) { [System.Text.Encoding]::UTF8.GetString($openAPIResponse.Content) } else { [string]$openAPIResponse.Content }
    if ($openAPI -notmatch '(?m)^  /ready:') { throw "Served OpenAPI does not contain /ready" }

    $login = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/login" -ContentType "application/json" -Body (@{ account = "admin"; password = "Admin@123456" } | ConvertTo-Json) -TimeoutSec 10
    $token = [string]$login.data.token
    if ([string]::IsNullOrWhiteSpace($token)) { throw "Runtime login returned no token" }
    $headers = @{ Authorization = "Bearer $token" }
    $me = Invoke-RestMethod -Uri "$base/v1/auth/me" -Headers $headers -TimeoutSec 10
    $plugins = Invoke-RestMethod -Uri "$base/v1/plugins" -TimeoutSec 10
    if (($plugins | ConvertTo-Json -Depth 20) -notmatch 'pharma_oa') { throw "Pharma OA plugin was not discovered" }

    $report = [ordered]@{
        schemaVersion = "skoll.h6-runtime-review.v1"
        locale = "zh-CN"
        generatedAt = (Get-Date).ToUniversalTime().ToString("o")
        mode = "memory-isolated"
        health = [ordered]@{ code = $health.code; passed = ($health.code -eq "ok") }
        readiness = [ordered]@{ code = $ready.code; passed = ($ready.code -eq "ready") }
        openapi = [ordered]@{ readinessPath = $true; passed = $true }
        authentication = [ordered]@{ tokenIssued = $true; profileReturned = ($null -ne $me.data); passed = ($null -ne $me.data) }
        pluginDiscovery = [ordered]@{ pharmaOA = $true; passed = $true }
        durationSeconds = [math]::Round(((Get-Date) - $startedAt).TotalSeconds, 3)
        passed = ($health.code -eq "ok" -and $ready.code -eq "ready" -and $null -ne $me.data)
    }
    if (-not $report.passed) { throw "Runtime review assertions failed" }
    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedOutput) -Force | Out-Null
    $report | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $resolvedOutput -Encoding utf8
    Write-Host "H6-03 isolated runtime review passed: $resolvedOutput"
} catch {
    $failure = $_.Exception.Message
    if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
        $process.WaitForExit(10000) | Out-Null
    }
    $log = ""
    if (Test-Path -LiteralPath $stderr) {
        try { $log = [System.IO.File]::ReadAllText($stderr) } catch { $log = "stderr unavailable: $($_.Exception.Message)" }
    }
    throw "$failure`n$log"
} finally {
    if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id
        if (-not $process.WaitForExit(10000)) { Stop-Process -Id $process.Id -Force }
    }
    foreach ($key in $saved.Keys) { [Environment]::SetEnvironmentVariable($key, $saved[$key], "Process") }
    $safeTempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    $resolvedTemp = [System.IO.Path]::GetFullPath($tempRoot)
    if ($resolvedTemp.StartsWith($safeTempRoot, [System.StringComparison]::OrdinalIgnoreCase) -and (Split-Path -Leaf $resolvedTemp) -like 'skoll-h6-runtime-*') {
        Remove-Item -LiteralPath $resolvedTemp -Recurse -Force -ErrorAction SilentlyContinue
    }
}
