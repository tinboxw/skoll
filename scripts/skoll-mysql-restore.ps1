param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9_]+$')]
    [string]$Database,
    [Parameter(Mandatory = $true)]
    [string]$InputPath,
    [string]$HostName = "127.0.0.1",
    [ValidateRange(1, 65535)]
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$PasswordEnvironmentVariable = "SKOLL_MYSQL_PASSWORD",
    [switch]$AllowRecreate
)

$ErrorActionPreference = "Stop"
if (-not $AllowRecreate) { throw "Restore recreates the target database; pass -AllowRecreate explicitly" }
if ($Database -in @("mysql", "information_schema", "performance_schema", "sys")) {
    throw "System databases cannot be restored by this script"
}
$resolvedInput = [System.IO.Path]::GetFullPath($InputPath)
if (-not (Test-Path -LiteralPath $resolvedInput)) { throw "Backup file not found: $resolvedInput" }
$metadataPath = "$resolvedInput.metadata.json"
if (Test-Path -LiteralPath $metadataPath) {
    $metadata = [System.IO.File]::ReadAllText($metadataPath, [System.Text.Encoding]::UTF8) | ConvertFrom-Json
    $actualHash = (Get-FileHash -LiteralPath $resolvedInput -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($metadata.sha256 -ne $actualHash) { throw "Backup checksum mismatch" }
}
$password = [Environment]::GetEnvironmentVariable($PasswordEnvironmentVariable)
if ([string]::IsNullOrWhiteSpace($password)) { throw "Set $PasswordEnvironmentVariable before restore" }
$tool = (Get-Command mysql -ErrorAction Stop).Source
$stderr = "$resolvedInput.restore.stderr"
$previousMySQLPassword = $env:MYSQL_PWD
$env:MYSQL_PWD = $password
try {
    $serverArgs = @("--host=$HostName", "--port=$Port", "--user=$User", "--default-character-set=utf8mb4")
    $recreateSQL = "DROP DATABASE IF EXISTS ``$Database``; CREATE DATABASE ``$Database`` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
    & $tool @serverArgs --execute=$recreateSQL
    if ($LASTEXITCODE -ne 0) { throw "Failed to recreate restore database" }
    $arguments = $serverArgs + @("--database=$Database")
    $process = Start-Process -FilePath $tool -ArgumentList $arguments -NoNewWindow -Wait -PassThru -RedirectStandardInput $resolvedInput -RedirectStandardError $stderr
    if ($process.ExitCode -ne 0) {
        $message = if (Test-Path $stderr) { [System.IO.File]::ReadAllText($stderr) } else { "unknown mysql restore failure" }
        throw "mysql restore failed: $message"
    }
} finally {
    $env:MYSQL_PWD = $previousMySQLPassword
    Remove-Item -LiteralPath $stderr -Force -ErrorAction SilentlyContinue
}
Write-Host "MySQL restore completed into: $Database"
