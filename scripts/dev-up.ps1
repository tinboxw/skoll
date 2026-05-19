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

Start-Process powershell -ArgumentList "-NoExit", "-Command", $backendCmd | Out-Null
Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCmd | Out-Null

Write-Host "Backend and frontend start commands have been launched in new terminals."
Write-Host "Backend: http://127.0.0.1:8080/skoll/health"
Write-Host "Frontend: http://127.0.0.1:5173/skoll/"
