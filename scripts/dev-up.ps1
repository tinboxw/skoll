param(
    [string]$StoreMode = "mysql",
    [string]$StoreDSN = "root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local",
    [switch]$ForceRestart,
    [switch]$SkipPostChecks,
    [switch]$FailOnCheckError,
    [string]$AdminAccount = "admin",
    [string]$AdminPassword = "Admin@123456"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$webRoot = Join-Path $repoRoot "web"
$logRoot = Join-Path $repoRoot "log"

if (-not (Test-Path -LiteralPath $logRoot)) {
    New-Item -ItemType Directory -Path $logRoot -Force | Out-Null
}

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

function Start-BackgroundPowerShell {
    param(
        [string]$Command,
        [string]$StdOutLog,
        [string]$StdErrLog
    )

    return Start-Process -FilePath "powershell" `
        -ArgumentList @("-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", $Command) `
        -WindowStyle Hidden `
        -RedirectStandardOutput $StdOutLog `
        -RedirectStandardError $StdErrLog `
        -PassThru
}

function Invoke-PostStartChecks {
    param(
        [string]$BackendBase,
        [string]$FrontendHealth,
        [string]$PluginsRoot,
        [string]$Account,
        [string]$Password,
        [bool]$FailOnError
    )

    Write-Host "Post checks: start"

    try {
        $frontHealth = Invoke-RestMethod -Uri $FrontendHealth -Method Get -TimeoutSec 3
        if ($null -eq $frontHealth -or $frontHealth.code -ne "ok") {
            throw "unexpected frontend health response"
        }
        Write-Host "Post checks: frontend health ok"
    } catch {
        $msg = "Post checks: frontend health failed - $($_.Exception.Message)"
        if ($FailOnError) {
            throw $msg
        }
        Write-Host $msg
        return
    }

    try {
        $loginPayload = @{ account = $Account; password = $Password } | ConvertTo-Json
        $loginResp = Invoke-RestMethod -Uri ($BackendBase + "/skoll/v1/auth/login") -Method Post -ContentType "application/json" -Body $loginPayload -TimeoutSec 5
        $token = ""
        if ($null -ne $loginResp -and $null -ne $loginResp.data -and $null -ne $loginResp.data.token) {
            $token = [string]$loginResp.data.token
        }
        if ([string]::IsNullOrWhiteSpace($token)) {
            throw "missing token in login response"
        }

        $headers = @{ Authorization = "Bearer $token" }
        $configResp = Invoke-RestMethod -Uri ($BackendBase + "/skoll/v1/plugins/dev/config") -Method Get -Headers $headers -TimeoutSec 5
        if ($null -eq $configResp -or $configResp.code -ne "ok") {
            throw "dev/config did not return code=ok"
        }

        $catalogUri = $BackendBase + "/skoll/v1/plugins/dev/permission-catalog?pluginsRoot=" + [System.Uri]::EscapeDataString($PluginsRoot)
        $catalogResp = Invoke-RestMethod -Uri $catalogUri -Method Get -Headers $headers -TimeoutSec 5
        if ($null -eq $catalogResp -or $catalogResp.code -ne "ok") {
            throw "permission-catalog did not return code=ok"
        }

        $frameworkCount = 0
        if ($null -ne $catalogResp.data -and $catalogResp.data.framework -is [System.Array]) {
            $frameworkCount = $catalogResp.data.framework.Count
        }
        Write-Host ("Post checks: dev/config ok, permission-catalog ok (framework={0})" -f $frameworkCount)
    } catch {
        $msg = "Post checks: dev endpoint checks failed - $($_.Exception.Message)"
        if ($FailOnError) {
            throw $msg
        }
        Write-Host $msg
    }
}

if ($ForceRestart) {
    Stop-PortListeners -Ports @(8080, 5173)
}

$backendCmd = @(
    "Set-Location '$repoRoot'",
    "`$env:SKOLL_LOG_DIR='log'",
    "`$env:SKOLL_LOG_FILE='skoll.log'",
    "`$env:SKOLL_DEV_PORTAL_ENABLED='true'",
    "`$env:SKOLL_DEV_PLUGINS_ROOT='plugins'",
    "`$env:SKOLL_STORE_MODE='$StoreMode'",
    "`$env:SKOLL_STORE_DSN='$StoreDSN'",
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
    $backendOutLog = Join-Path $logRoot "dev-backend.out.log"
    $backendErrLog = Join-Path $logRoot "dev-backend.err.log"
    $backendProc = Start-BackgroundPowerShell -Command $backendCmd -StdOutLog $backendOutLog -StdErrLog $backendErrLog
    Write-Host ("Backend started in background. PID={0}" -f $backendProc.Id)
    Write-Host ("Backend logs: {0} | {1}" -f $backendOutLog, $backendErrLog)
} else {
    Write-Host "Backend launch skipped."
}

if ($launchFrontend) {
    $frontendOutLog = Join-Path $logRoot "dev-frontend.out.log"
    $frontendErrLog = Join-Path $logRoot "dev-frontend.err.log"
    $frontendProc = Start-BackgroundPowerShell -Command $frontendCmd -StdOutLog $frontendOutLog -StdErrLog $frontendErrLog
    Write-Host ("Frontend started in background. PID={0}" -f $frontendProc.Id)
    Write-Host ("Frontend logs: {0} | {1}" -f $frontendOutLog, $frontendErrLog)
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

if (-not $SkipPostChecks) {
    Invoke-PostStartChecks `
        -BackendBase "http://127.0.0.1:8080" `
        -FrontendHealth "http://127.0.0.1:5173/skoll/health" `
        -PluginsRoot "plugins" `
        -Account $AdminAccount `
        -Password $AdminPassword `
        -FailOnError ([bool]$FailOnCheckError)
}
