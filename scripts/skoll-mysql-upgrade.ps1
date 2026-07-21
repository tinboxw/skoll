param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9_]+$')]
    [string]$Database,
    [Parameter(Mandatory = $true)]
    [long]$FromExclusive,
    [Parameter(Mandatory = $true)]
    [long]$ToInclusive,
    [Parameter(Mandatory = $true)]
    [string]$BackupPath,
    [string]$MigrationDirectory = "migrations/mysql",
    [string]$HostName = "127.0.0.1",
    [ValidateRange(1, 65535)]
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$PasswordEnvironmentVariable = "SKOLL_MYSQL_PASSWORD",
    [string]$ReportPath = "",
    [switch]$AllowUpgrade
)

$ErrorActionPreference = "Stop"
if (-not $AllowUpgrade) { throw "Upgrade changes schema; pass -AllowUpgrade explicitly" }
if ($ToInclusive -le $FromExclusive) { throw "ToInclusive must be greater than FromExclusive" }
$repoRoot = Split-Path -Parent $PSScriptRoot
$migrationRoot = if ([System.IO.Path]::IsPathRooted($MigrationDirectory)) { $MigrationDirectory } else { Join-Path $repoRoot $MigrationDirectory }
$files = @(Get-ChildItem -LiteralPath $migrationRoot -Filter '*.sql' -File | ForEach-Object {
    if ($_.BaseName -notmatch '^(\d{8})_(\d{6})_') { return }
    [pscustomobject]@{ Version = [long]("$($Matches[1])$($Matches[2])"); File = $_ }
} | Where-Object { $_.Version -gt $FromExclusive -and $_.Version -le $ToInclusive } | Sort-Object Version)
if ($files.Count -eq 0) { throw "No migrations selected for ($FromExclusive, $ToInclusive]" }

$backupScript = Join-Path $PSScriptRoot "skoll-mysql-backup.ps1"
& $backupScript -Database $Database -OutputPath $BackupPath -HostName $HostName -Port $Port -User $User -PasswordEnvironmentVariable $PasswordEnvironmentVariable
if ($LASTEXITCODE -ne 0) { throw "Pre-upgrade backup failed" }

$password = [Environment]::GetEnvironmentVariable($PasswordEnvironmentVariable)
if ([string]::IsNullOrWhiteSpace($password)) { throw "Set $PasswordEnvironmentVariable before upgrade" }
$tool = (Get-Command mysql -ErrorAction Stop).Source
$previousMySQLPassword = $env:MYSQL_PWD
$env:MYSQL_PWD = $password
$startedAt = Get-Date
$applied = @()
try {
    foreach ($entry in $files) {
        $stderr = "$($entry.File.FullName).h6.stderr"
        $migrationInput = Join-Path ([System.IO.Path]::GetTempPath()) "skoll-migration-$PID-$($entry.Version).sql"
        try {
            $migrationSQL = "SET SESSION default_storage_engine=InnoDB;`r`n" + [System.IO.File]::ReadAllText($entry.File.FullName)
            [System.IO.File]::WriteAllText($migrationInput, $migrationSQL, [System.Text.UTF8Encoding]::new($false))
            $arguments = @("--host=$HostName", "--port=$Port", "--user=$User", "--default-character-set=utf8mb4", "--database=$Database")
            $process = Start-Process -FilePath $tool -ArgumentList $arguments -NoNewWindow -Wait -PassThru -RedirectStandardInput $migrationInput -RedirectStandardError $stderr
            if ($process.ExitCode -ne 0) {
                $message = if (Test-Path $stderr) { [System.IO.File]::ReadAllText($stderr) } else { "unknown migration failure" }
                throw "Migration $($entry.File.Name) failed: $message. Restore $BackupPath before retry."
            }
            $applied += $entry.File.Name
        } finally {
            Remove-Item -LiteralPath $migrationInput -Force -ErrorAction SilentlyContinue
            Remove-Item -LiteralPath $stderr -Force -ErrorAction SilentlyContinue
        }
    }
} finally {
    $env:MYSQL_PWD = $previousMySQLPassword
}
$report = [ordered]@{
    schemaVersion = "skoll.mysql-upgrade.v1"
    database = $Database
    fromExclusive = $FromExclusive
    toInclusive = $ToInclusive
    backup = [System.IO.Path]::GetFullPath($BackupPath)
    applied = $applied
    durationMilliseconds = [math]::Round(((Get-Date) - $startedAt).TotalMilliseconds, 3)
    passed = $true
}
if (-not [string]::IsNullOrWhiteSpace($ReportPath)) {
    $resolvedReport = [System.IO.Path]::GetFullPath($ReportPath)
    New-Item -ItemType Directory -Path (Split-Path -Parent $resolvedReport) -Force | Out-Null
    $report | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $resolvedReport -Encoding utf8
}
Write-Host "MySQL upgrade completed: $($applied -join ', ')"
