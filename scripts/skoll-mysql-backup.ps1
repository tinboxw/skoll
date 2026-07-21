param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9_]+$')]
    [string]$Database,
    [Parameter(Mandatory = $true)]
    [string]$OutputPath,
    [string]$HostName = "127.0.0.1",
    [ValidateRange(1, 65535)]
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$PasswordEnvironmentVariable = "SKOLL_MYSQL_PASSWORD"
)

$ErrorActionPreference = "Stop"
if ($Database -in @("mysql", "information_schema", "performance_schema", "sys")) {
    throw "System databases cannot be backed up by this script"
}
$password = [Environment]::GetEnvironmentVariable($PasswordEnvironmentVariable)
if ([string]::IsNullOrWhiteSpace($password)) { throw "Set $PasswordEnvironmentVariable before backup" }
$tool = (Get-Command mysqldump -ErrorAction Stop).Source
$resolvedOutput = [System.IO.Path]::GetFullPath($OutputPath)
$parent = Split-Path -Parent $resolvedOutput
New-Item -ItemType Directory -Path $parent -Force | Out-Null
$stderr = "$resolvedOutput.stderr"
$previousMySQLPassword = $env:MYSQL_PWD
$env:MYSQL_PWD = $password
try {
    $helpText = (& $tool --help) -join "`n"
    $arguments = @(
        "--host=$HostName", "--port=$Port", "--user=$User",
        "--default-character-set=utf8mb4", "--single-transaction",
        "--routines", "--triggers", "--events", "--skip-comments"
    )
    if ($helpText -match '(?m)^\s*--column-statistics') { $arguments += "--skip-column-statistics" }
    $arguments += $Database
    $process = Start-Process -FilePath $tool -ArgumentList $arguments -NoNewWindow -Wait -PassThru -RedirectStandardOutput $resolvedOutput -RedirectStandardError $stderr
    if ($process.ExitCode -ne 0) {
        $message = if (Test-Path $stderr) { [System.IO.File]::ReadAllText($stderr) } else { "unknown mysqldump failure" }
        throw "mysqldump failed: $message"
    }
} finally {
    $env:MYSQL_PWD = $previousMySQLPassword
    Remove-Item -LiteralPath $stderr -Force -ErrorAction SilentlyContinue
}
if (-not (Test-Path -LiteralPath $resolvedOutput) -or (Get-Item -LiteralPath $resolvedOutput).Length -eq 0) {
    throw "Backup output is empty"
}
$hash = (Get-FileHash -LiteralPath $resolvedOutput -Algorithm SHA256).Hash.ToLowerInvariant()
$metadata = [ordered]@{
    schemaVersion = "skoll.mysql-backup.v1"
    database = $Database
    generatedAt = (Get-Date).ToUniversalTime().ToString("o")
    sha256 = $hash
    bytes = (Get-Item -LiteralPath $resolvedOutput).Length
}
$metadata | ConvertTo-Json | Set-Content -LiteralPath "$resolvedOutput.metadata.json" -Encoding utf8
Write-Host "MySQL backup completed: $resolvedOutput"
