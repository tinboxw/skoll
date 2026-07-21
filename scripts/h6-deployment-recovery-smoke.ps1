param(
    [string]$HostName = "127.0.0.1",
    [ValidateRange(1, 65535)]
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$PasswordEnvironmentVariable = "SKOLL_H6_MYSQL_PASSWORD",
    [string]$OutputPath = "docs/refactor/current/evidence/h6-01/deployment-recovery-smoke.json",
    [switch]$AllowDatabaseLifecycle,
    [switch]$KeepDatabases
)

$ErrorActionPreference = "Stop"
if (-not $AllowDatabaseLifecycle) { throw "Smoke creates and drops isolated skoll_h6_* databases; pass -AllowDatabaseLifecycle explicitly" }
$password = [Environment]::GetEnvironmentVariable($PasswordEnvironmentVariable)
if ([string]::IsNullOrWhiteSpace($password)) { throw "Set $PasswordEnvironmentVariable before running the smoke" }
$repoRoot = Split-Path -Parent $PSScriptRoot
$resolvedOutput = if ([System.IO.Path]::IsPathRooted($OutputPath)) { $OutputPath } else { Join-Path $repoRoot $OutputPath }
$suffix = "{0}_{1}" -f (Get-Date -Format "yyyyMMddHHmmss"), $PID
$databases = @(
    "skoll_h6_clean_$suffix",
    "skoll_h6_restore_$suffix",
    "skoll_h6_upgrade_$suffix",
    "skoll_h6_rollback_$suffix"
)
$cleanDatabase, $restoreDatabase, $upgradeDatabase, $rollbackDatabase = $databases
$tempRoot = [System.IO.Path]::GetFullPath((Join-Path ([System.IO.Path]::GetTempPath()) "skoll-h6-$suffix"))
New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null
$binary = Join-Path $tempRoot "skoll.exe"
$mysql = (Get-Command mysql -ErrorAction Stop).Source
$backupScript = Join-Path $PSScriptRoot "skoll-mysql-backup.ps1"
$restoreScript = Join-Path $PSScriptRoot "skoll-mysql-restore.ps1"
$upgradeScript = Join-Path $PSScriptRoot "skoll-mysql-upgrade.ps1"
$previousPassword = $env:MYSQL_PWD
$previousSharedPassword = $env:SKOLL_MYSQL_PASSWORD
$env:MYSQL_PWD = $password
$env:SKOLL_MYSQL_PASSWORD = $password
$activeProcess = $null
$startedAt = Get-Date

function Invoke-MySQL([string]$Database, [string]$SQL) {
    $arguments = @("--host=$HostName", "--port=$Port", "--user=$User", "--default-character-set=utf8mb4", "--batch", "--skip-column-names")
    if (-not [string]::IsNullOrWhiteSpace($Database)) { $arguments += "--database=$Database" }
    $output = @(& $mysql @arguments --execute=$SQL)
    if ($LASTEXITCODE -ne 0) { throw "MySQL command failed for database $Database" }
    return ($output -join "`n").Trim()
}

function New-H6Database([string]$Database) {
    if ($Database -notmatch '^skoll_h6_[A-Za-z0-9_]+$') { throw "Unsafe rehearsal database name: $Database" }
    Invoke-MySQL "" "CREATE DATABASE ``$Database`` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" | Out-Null
}

function Remove-H6Database([string]$Database) {
    if ($Database -notmatch '^skoll_h6_[A-Za-z0-9_]+$') { throw "Unsafe rehearsal database name: $Database" }
    Invoke-MySQL "" "DROP DATABASE IF EXISTS ``$Database``;" | Out-Null
}

function Get-FreeTCPPort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $selected = ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port
    $listener.Stop()
    return $selected
}

function Wait-SkollEndpoint([string]$URL, [System.Diagnostics.Process]$Process) {
    $deadline = (Get-Date).AddSeconds(45)
    while ((Get-Date) -lt $deadline) {
        if ($Process.HasExited) { throw "Skoll exited before $URL became available" }
        try {
            $response = Invoke-RestMethod -Uri $URL -Method Get -TimeoutSec 2
            if ($response.code -in @("ok", "ready")) { return }
        } catch {
            Start-Sleep -Milliseconds 250
        }
    }
    throw "Timed out waiting for $URL"
}

function Start-H6Skoll([string]$Database, [string]$Label) {
    $httpPort = Get-FreeTCPPort
    $stdout = Join-Path $tempRoot "$Label.stdout.log"
    $stderr = Join-Path $tempRoot "$Label.stderr.log"
    $logDir = Join-Path $tempRoot "$Label-logs"
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    $saved = @{
        SKOLL_SERVER_ADDRESS = $env:SKOLL_SERVER_ADDRESS
        SKOLL_API_BASE_PREFIX = $env:SKOLL_API_BASE_PREFIX
        SKOLL_STORE_MODE = $env:SKOLL_STORE_MODE
        SKOLL_STORE_DSN = $env:SKOLL_STORE_DSN
        SKOLL_SECURITY_JWT_SECRET = $env:SKOLL_SECURITY_JWT_SECRET
        SKOLL_LOG_DIR = $env:SKOLL_LOG_DIR
        SKOLL_LOG_FILE = $env:SKOLL_LOG_FILE
    }
    try {
        $env:SKOLL_SERVER_ADDRESS = "127.0.0.1:$httpPort"
        $env:SKOLL_API_BASE_PREFIX = "/skoll"
        $env:SKOLL_STORE_MODE = "mysql"
        $env:SKOLL_STORE_DSN = "$User`:$password@tcp($HostName`:$Port)/$Database`?charset=utf8mb4&parseTime=True&loc=UTC"
        $env:SKOLL_SECURITY_JWT_SECRET = "h6-rehearsal-secret-32-characters"
        $env:SKOLL_LOG_DIR = $logDir
        $env:SKOLL_LOG_FILE = "skoll.log"
        $process = Start-Process -FilePath $binary -WorkingDirectory $tempRoot -WindowStyle Hidden -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    } finally {
        foreach ($key in $saved.Keys) { [Environment]::SetEnvironmentVariable($key, $saved[$key], "Process") }
    }
    try {
        Wait-SkollEndpoint "http://127.0.0.1:$httpPort/skoll/health" $process
        Wait-SkollEndpoint "http://127.0.0.1:$httpPort/skoll/ready" $process
    } catch {
        $log = if (Test-Path $stderr) { [System.IO.File]::ReadAllText($stderr) } else { "" }
        if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force }
        throw "$($_.Exception.Message)`n$log"
    }
    return [pscustomobject]@{ Process = $process; Port = $httpPort; Health = "passed"; Readiness = "passed" }
}

function Stop-H6Skoll($Run) {
    if ($null -eq $Run -or $null -eq $Run.Process -or $Run.Process.HasExited) { return }
    Stop-Process -Id $Run.Process.Id
    if (-not $Run.Process.WaitForExit(10000)) { Stop-Process -Id $Run.Process.Id -Force }
}

try {
    Write-Host "Validating Compose and Kubernetes examples..."
    $savedCompose = @{
        SKOLL_MYSQL_PASSWORD = $env:SKOLL_MYSQL_PASSWORD
        SKOLL_MYSQL_ROOT_PASSWORD = $env:SKOLL_MYSQL_ROOT_PASSWORD
        SKOLL_SECURITY_JWT_SECRET = $env:SKOLL_SECURITY_JWT_SECRET
    }
    $env:SKOLL_MYSQL_PASSWORD = "h6-compose-password"
    $env:SKOLL_MYSQL_ROOT_PASSWORD = "h6-compose-root-password"
    $env:SKOLL_SECURITY_JWT_SECRET = "h6-compose-jwt-secret-32-characters"
    $composeConfig = @(& docker compose -f (Join-Path $repoRoot "deploy/compose/docker-compose.yaml") config)
    $composeText = $composeConfig -join "`n"
    if ($LASTEXITCODE -ne 0 -or $composeText -notmatch '/app/plugins' -or $composeText -notmatch '/skoll/ready') { throw "Docker Compose config validation failed" }
    foreach ($key in $savedCompose.Keys) { [Environment]::SetEnvironmentVariable($key, $savedCompose[$key], "Process") }
    $kustomized = @(& kubectl kustomize (Join-Path $repoRoot "deploy/k8s"))
    $kustomizeText = $kustomized -join "`n"
    if ($LASTEXITCODE -ne 0 -or $kustomizeText -notmatch 'kind: Deployment' -or $kustomizeText -notmatch 'claimName: skoll-plugins' -or $kustomizeText -notmatch '/skoll/health' -or $kustomizeText -notmatch '/skoll/ready') { throw "Kubernetes kustomize validation failed" }

    Write-Host "Building Skoll rehearsal binary..."
    Push-Location $repoRoot
    try { & go build -trimpath -o $binary ./cmd/skoll } finally { Pop-Location }
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $binary)) { throw "Skoll binary build failed" }

    foreach ($database in $databases) { New-H6Database $database }

    Write-Host "Rehearsing clean deployment and probes..."
    $cleanRun = Start-H6Skoll $cleanDatabase "clean"
    $activeProcess = $cleanRun
    $cleanTables = [int](Invoke-MySQL $cleanDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE();")
    if ($cleanTables -lt 30) { throw "Clean deployment created only $cleanTables tables" }
    Stop-H6Skoll $cleanRun
    $activeProcess = $null
    Invoke-MySQL $cleanDatabase "CREATE TABLE h6_rehearsal_markers (id VARCHAR(32) PRIMARY KEY, value VARCHAR(64) NOT NULL); INSERT INTO h6_rehearsal_markers VALUES ('marker', 'clean-v1');" | Out-Null

    Write-Host "Rehearsing backup and restore..."
    $cleanBackup = Join-Path $tempRoot "clean-backup.sql"
    & $backupScript -Database $cleanDatabase -OutputPath $cleanBackup -HostName $HostName -Port $Port -User $User
    Invoke-MySQL $cleanDatabase "UPDATE h6_rehearsal_markers SET value='mutated';" | Out-Null
    & $restoreScript -Database $restoreDatabase -InputPath $cleanBackup -HostName $HostName -Port $Port -User $User -AllowRecreate
    $restoredMarker = Invoke-MySQL $restoreDatabase "SELECT value FROM h6_rehearsal_markers WHERE id='marker';"
    if ($restoredMarker -ne "clean-v1") { throw "Restore marker mismatch: $restoredMarker" }

    Write-Host "Preparing source schema and rehearsing forward upgrade..."
    & $restoreScript -Database $upgradeDatabase -InputPath $cleanBackup -HostName $HostName -Port $Port -User $User -AllowRecreate
    Invoke-MySQL $upgradeDatabase "DROP TABLE IF EXISTS pharma_oa_announcements, pharma_oa_cold_chain_records; UPDATE h6_rehearsal_markers SET value='source-v1' WHERE id='marker';" | Out-Null
    $announcementBefore = [int](Invoke-MySQL $upgradeDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_announcements';")
    $coldChainBefore = [int](Invoke-MySQL $upgradeDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_cold_chain_records';")
    if ($announcementBefore -ne 0 -or $coldChainBefore -ne 0) { throw "Target-only tables exist before upgrade" }
    $preUpgradeBackup = Join-Path $tempRoot "pre-upgrade.sql"
    $upgradeReport = Join-Path $tempRoot "upgrade-report.json"
    & $upgradeScript -Database $upgradeDatabase -FromExclusive 20260718000021 -ToInclusive 20260718000022 -BackupPath $preUpgradeBackup -HostName $HostName -Port $Port -User $User -ReportPath $upgradeReport -AllowUpgrade
    $announcementAfter = [int](Invoke-MySQL $upgradeDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_announcements';")
    $coldChainAfter = [int](Invoke-MySQL $upgradeDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_cold_chain_records';")
    if ($announcementAfter -ne 1 -or $coldChainAfter -ne 1) { throw "Forward migration did not create both target tables" }
    $upgradeRun = Start-H6Skoll $upgradeDatabase "upgrade"
    $activeProcess = $upgradeRun
    Stop-H6Skoll $upgradeRun
    $activeProcess = $null

    Write-Host "Rehearsing backup-based rollback..."
    & $restoreScript -Database $rollbackDatabase -InputPath $preUpgradeBackup -HostName $HostName -Port $Port -User $User -AllowRecreate
    $announcementRollback = [int](Invoke-MySQL $rollbackDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_announcements';")
    $coldChainRollback = [int](Invoke-MySQL $rollbackDatabase "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='pharma_oa_cold_chain_records';")
    $rollbackMarker = Invoke-MySQL $rollbackDatabase "SELECT value FROM h6_rehearsal_markers WHERE id='marker';"
    if ($announcementRollback -ne 0 -or $coldChainRollback -ne 0 -or $rollbackMarker -ne "source-v1") { throw "Rollback state mismatch" }

    $dockerStatus = "daemon-unavailable"
    $dockerInfoOutput = Join-Path $tempRoot "docker-info.stdout.log"
    $dockerInfoError = Join-Path $tempRoot "docker-info.stderr.log"
    $dockerInfo = Start-Process -FilePath (Get-Command docker -ErrorAction Stop).Source -ArgumentList @("info", "--format", "{{.ServerVersion}}") -NoNewWindow -Wait -PassThru -RedirectStandardOutput $dockerInfoOutput -RedirectStandardError $dockerInfoError
    if ($dockerInfo.ExitCode -eq 0) { $dockerStatus = "available-compose-config-validated" }
    $cleanMetadata = [System.IO.File]::ReadAllText("$cleanBackup.metadata.json", [System.Text.Encoding]::UTF8) | ConvertFrom-Json
    $preUpgradeMetadata = [System.IO.File]::ReadAllText("$preUpgradeBackup.metadata.json", [System.Text.Encoding]::UTF8) | ConvertFrom-Json
    $report = [ordered]@{
        schemaVersion = "skoll.h6-deployment-recovery.v1"
        locale = "zh-CN"
        generatedAt = (Get-Date).ToUniversalTime().ToString("o")
        environment = [ordered]@{ mysqlHost = $HostName; mysqlPort = $Port; docker = $dockerStatus; kubernetes = "kustomize-client-validated" }
        cleanDeployment = [ordered]@{ tables = $cleanTables; health = $cleanRun.Health; readiness = $cleanRun.Readiness; passed = $true }
        backupRestore = [ordered]@{ bytes = $cleanMetadata.bytes; sha256 = $cleanMetadata.sha256; marker = $restoredMarker; passed = $true }
        forwardUpgrade = [ordered]@{ from = 20260718000021; to = 20260718000022; targetTablesBefore = $announcementBefore + $coldChainBefore; targetTablesAfter = $announcementAfter + $coldChainAfter; health = $upgradeRun.Health; readiness = $upgradeRun.Readiness; passed = $true }
        rollback = [ordered]@{ strategy = "restore-point"; backupBytes = $preUpgradeMetadata.bytes; targetTablesAfterRollback = $announcementRollback + $coldChainRollback; marker = $rollbackMarker; passed = $true }
        compose = [ordered]@{ configValidated = $true; persistentVolumes = @("mysql_data", "skoll_data", "skoll_logs", "skoll_plugins") }
        kubernetes = [ordered]@{ kustomizeValidated = $true; probes = @("/skoll/health", "/skoll/ready"); secretName = "skoll-runtime" }
        durationSeconds = [math]::Round(((Get-Date) - $startedAt).TotalSeconds, 3)
        passed = $true
    }
    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedOutput) -Force | Out-Null
    $report | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $resolvedOutput -Encoding utf8
    Write-Host "H6-01 deployment and recovery smoke passed: $resolvedOutput"
} finally {
    if ($null -ne $activeProcess) { Stop-H6Skoll $activeProcess }
    if (-not $KeepDatabases) {
        foreach ($database in $databases) {
            try { Remove-H6Database $database } catch { Write-Warning "Failed to clean $database`: $($_.Exception.Message)" }
        }
    }
    $env:MYSQL_PWD = $previousPassword
    $env:SKOLL_MYSQL_PASSWORD = $previousSharedPassword
    $safeTempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    if ($tempRoot.StartsWith($safeTempRoot, [System.StringComparison]::OrdinalIgnoreCase) -and (Split-Path -Leaf $tempRoot) -like 'skoll-h6-*') {
        Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
