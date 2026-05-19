param(
    [string]$BasePrefix = "http://127.0.0.1:8080/skoll",
    [string]$Account = "admin",
    [string]$Password = "Admin@123456",
    [int]$TimeoutSec = 8,
    [string]$ExportDir = "tmp"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "`n=== $Message ===" -ForegroundColor Cyan
}

function Invoke-Json {
    param(
        [string]$Method,
        [string]$Uri,
        [object]$Body,
        [hashtable]$Headers
    )

    $args = @{
        Method      = $Method
        Uri         = $Uri
        TimeoutSec  = $TimeoutSec
        ErrorAction = "Stop"
    }

    if ($null -ne $Body) {
        $args.ContentType = "application/json"
        $args.Body = ($Body | ConvertTo-Json -Depth 8)
    }

    if ($null -ne $Headers -and $Headers.Count -gt 0) {
        $args.Headers = $Headers
    }

    return Invoke-RestMethod @args
}

function Mask-Token {
    param([string]$Token)
    if ([string]::IsNullOrWhiteSpace($Token)) {
        return ""
    }
    if ($Token.Length -le 12) {
        return $Token
    }
    return "$($Token.Substring(0, 8))...$($Token.Substring($Token.Length - 4))"
}

try {
    $healthUri = "$BasePrefix/health"
    $loginUri = "$BasePrefix/v1/auth/login"
    $meUri = "$BasePrefix/v1/auth/me"
    $auditUri = "$BasePrefix/v1/audit?limit=20"
    $exportUri = "$BasePrefix/v1/audit/export?limit=20"

    Write-Step "1) Health Check"
    $health = Invoke-Json -Method "GET" -Uri $healthUri -Body $null -Headers @{}
    Write-Host "Health response: $($health | ConvertTo-Json -Depth 5)"

    Write-Step "2) Login"
    $login = Invoke-Json -Method "POST" -Uri $loginUri -Body @{ account = $Account; password = $Password } -Headers @{}
    $token = [string]$login.data.token
    if ([string]::IsNullOrWhiteSpace($token)) {
        throw "login succeeded but token is empty"
    }
    Write-Host "Login ok. token=$((Mask-Token -Token $token)) user=$($login.data.user.account)"

    $headers = @{ Authorization = "Bearer $token" }

    Write-Step "3) Current User"
    $me = Invoke-Json -Method "GET" -Uri $meUri -Body $null -Headers $headers
    Write-Host "Me: $($me | ConvertTo-Json -Depth 5)"

    Write-Step "4) Audit Query"
    $audit = Invoke-Json -Method "GET" -Uri $auditUri -Body $null -Headers $headers
    $count = 0
    if ($audit.data -is [System.Collections.IEnumerable]) {
        foreach ($item in $audit.data) { $count++ }
    }
    Write-Host "Audit records fetched: $count"

    Write-Step "5) Audit Export"
    if (!(Test-Path $ExportDir)) {
        New-Item -ItemType Directory -Path $ExportDir | Out-Null
    }
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $exportPath = Join-Path $ExportDir "audit-smoke-$timestamp.csv"
    Invoke-WebRequest -Method GET -Uri $exportUri -Headers $headers -OutFile $exportPath -TimeoutSec $TimeoutSec -UseBasicParsing -ErrorAction Stop | Out-Null
    $size = (Get-Item $exportPath).Length
    Write-Host "Export saved: $exportPath ($size bytes)"

    Write-Step "Result"
    Write-Host "Smoke check passed." -ForegroundColor Green
    exit 0
}
catch {
    Write-Host "`nSmoke check failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
