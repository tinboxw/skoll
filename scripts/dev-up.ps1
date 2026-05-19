param(
    [string]$StoreMode = "mysql",
    [string]$StoreDSN = "root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local",
    [switch]$ForceRestart
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$webRoot = Join-Path $repoRoot "web"

function Stop-PortListeners {
    param([int[]]$Ports)

    foreach ($port in $Ports) {
        $listeners = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
        foreach ($listener in $listeners) {
            try {
                Stop-Process -Id $listener.OwningProcess -Force -ErrorAction Stop
                Write-Host ("Stopped PID={0} on port {1}" -f $listener.OwningProcess, $port)
            } catch {
                Write-Host ("Failed to stop PID={0} on port {1}: {2}" -f $listener.OwningProcess, $port, $_.Exception.Message)
            }
        }
    }
}

function Get-PortListeners {
    param([int]$Port)

    return @(Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Wait-BackendReady {
    param(
        [string]$Url,
        [int]$MaxAttempts = 30,
        [int]$DelayMs = 300
    )

    for ($i = 0; $i -lt $MaxAttempts; $i++) {
        try {
            $resp = Invoke-RestMethod -Uri $Url -Method Get -TimeoutSec 2
            if ($null -ne $resp -and $resp.code -eq "ok") {
                return $true
            }
        } catch {
            # Keep retrying until timeout.
        }
        Start-Sleep -Milliseconds $DelayMs
    }

    return $false
}

if ($ForceRestart) {
    Stop-PortListeners -Ports @(8080, 5173)
}

$backendCmd = @(
    "Set-Location '$repoRoot'",
    "$env:SKOLL_LOG_DIR='log'",
    "$env:SKOLL_LOG_FILE='skoll.log'",
    "$env:SKOLL_STORE_MODE='$StoreMode'",
    "$env:SKOLL_STORE_DSN='$StoreDSN'",
    "go run ./cmd/skoll"
) -join "; "

$frontendCmd = @(
    "Set-Location '$webRoot'",
    "npx vite --host 127.0.0.1 --port 5173"
) -join "; "

$backendListeners = Get-PortListeners -Port 8080
$frontendListeners = Get-PortListeners -Port 5173

$launchBackend = $true
$launchFrontend = $true

if (-not $ForceRestart -and @($backendListeners).Length -gt 0) {
    $launchBackend = $false
    Write-Host "Port 8080 is already in use; skipping backend launch. Use -ForceRestart to replace existing process."
}

if (-not $ForceRestart -and @($frontendListeners).Length -gt 0) {
    $launchFrontend = $false
    Write-Host "Port 5173 is already in use; skipping frontend launch. Use -ForceRestart to replace existing process."
}

if ($launchBackend) {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $backendCmd | Out-Null
    Write-Host "Backend launch command sent."
} else {
    Write-Host "Backend launch skipped."
}

if ($launchFrontend) {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCmd | Out-Null
    Write-Host "Frontend launch command sent."
} else {
    Write-Host "Frontend launch skipped."
}

Write-Host "Backend: http://127.0.0.1:8080/skoll/health"
Write-Host "Frontend: http://127.0.0.1:5173/skoll/"

if ($launchBackend) {
    if (Wait-BackendReady -Url "http://127.0.0.1:8080/skoll/health") {
        Write-Host "Backend health check: ok"
    } else {
        Write-Host "Backend health check: not ready yet (check backend terminal/log)."
    }
}
