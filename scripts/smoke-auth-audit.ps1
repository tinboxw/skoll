param(
    [string]$BasePrefix = "http://127.0.0.1:8080/skoll",
    [string]$Account = "admin",
    [string]$Password = "Admin@123456",
    [string]$BadPassword = "Skoll-Smoke-Wrong-Password",
    [int]$TimeoutSec = 8,
    [string]$ExportDir = "tmp",
    [string]$FixturePath = "docs/archive/refactor-2026-06-19/fixtures/m2_audit_smoke_events.json",
    [switch]$StrictFixtureAssertions
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

function Resolve-SmokePath {
    param([string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return $Path
    }
    $repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
    return Join-Path $repoRoot $Path
}

function Read-SmokeFixture {
    param([string]$Path)
    $resolved = Resolve-SmokePath -Path $Path
    if (!(Test-Path $resolved)) {
        throw "fixture not found: $resolved"
    }
    $fixture = Get-Content -Raw -Encoding utf8 $resolved | ConvertFrom-Json
    $required = @("login_failed", "forbidden", "plugin", "menu", "export")
    $ids = @($fixture.scenarios | ForEach-Object { [string]$_.id })
    $missing = @($required | Where-Object { $_ -notin $ids })
    if ($missing.Count -gt 0) {
        throw "fixture missing scenarios: $($missing -join ', ')"
    }
    return $fixture
}

function Get-AuditItems {
    param([object]$AuditResponse)
    if ($null -eq $AuditResponse) {
        return @()
    }
    if ($null -ne $AuditResponse.data -and $null -ne $AuditResponse.data.items) {
        return @($AuditResponse.data.items)
    }
    if ($null -ne $AuditResponse.data -and $AuditResponse.data -is [System.Collections.IEnumerable] -and !($AuditResponse.data -is [string])) {
        return @($AuditResponse.data)
    }
    return @()
}

function Assert-ContainsToken {
    param(
        [string]$Text,
        [string]$Token,
        [string]$Context
    )
    if ($Text.IndexOf($Token, [System.StringComparison]::OrdinalIgnoreCase) -lt 0) {
        throw "$Context missing token: $Token"
    }
}

function Get-ObjectProperty {
    param(
        [object]$Object,
        [string]$Name
    )
    if ($null -eq $Object) {
        return $null
    }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return $null
    }
    return $property.Value
}

function Invoke-ExpectedLoginFailure {
    param([string]$LoginUri)
    try {
        $null = Invoke-Json -Method "POST" -Uri $LoginUri -Body @{ account = $Account; password = $BadPassword } -Headers @{}
        throw "bad password login unexpectedly succeeded"
    }
    catch {
        if ($_.Exception.Message -eq "bad password login unexpectedly succeeded") {
            throw
        }
        Write-Host "Expected login failure triggered for audit scenario login_failed."
    }
}

function Assert-Scenario {
    param(
        [object]$Scenario,
        [string]$BasePrefix,
        [hashtable]$Headers,
        [string]$ExportDir
    )
    $eventId = [string]$Scenario.event.id
    $query = [string]$Scenario.expectedQueries.list
    if ([string]::IsNullOrWhiteSpace($query)) {
        throw "scenario $($Scenario.id) is missing expectedQueries.list"
    }
    if ($query.IndexOf("limit=", [System.StringComparison]::OrdinalIgnoreCase) -lt 0) {
        $query = "$query&limit=50"
    }
    $listUri = "$BasePrefix/v1/audit?$query"
    $list = Invoke-Json -Method "GET" -Uri $listUri -Body $null -Headers $Headers
    $items = Get-AuditItems -AuditResponse $list
    $match = @($items | Where-Object {
        $itemID = [string](Get-ObjectProperty -Object $_ -Name "id")
        $metadata = Get-ObjectProperty -Object $_ -Name "metadata"
        $scenarioID = [string](Get-ObjectProperty -Object $metadata -Name "scenario")
        $itemID -eq $eventId -or $scenarioID -eq [string]$Scenario.id
    })
    $fixedFixtureMatch = $match.Count -gt 0
    if ($match.Count -eq 0 -and [string]$Scenario.id -eq "login_failed") {
        $match = @($items | Where-Object {
            [string](Get-ObjectProperty -Object $_ -Name "action") -eq [string]$Scenario.event.action -and
            [string](Get-ObjectProperty -Object $_ -Name "result") -eq [string]$Scenario.event.result -and
            [string](Get-ObjectProperty -Object $_ -Name "risk") -eq [string]$Scenario.event.risk
        })
    }
    if ($match.Count -eq 0) {
        $message = "scenario $($Scenario.id) not found by query: $query"
        if ($StrictFixtureAssertions) {
            throw $message
        }
        Write-Host "Warning: $message" -ForegroundColor Yellow
        return
    }

    $detailUri = "$BasePrefix/v1/audit/$([System.Uri]::EscapeDataString([string]$match[0].id))"
    $detail = Invoke-Json -Method "GET" -Uri $detailUri -Body $null -Headers $Headers
    if ($null -eq $detail.data.item.sourceData) {
        throw "scenario $($Scenario.id) detail missing sourceData"
    }

    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $exportPath = Join-Path $ExportDir "audit-smoke-$($Scenario.id)-$timestamp.csv"
    Invoke-WebRequest -Method GET -Uri "$BasePrefix/v1/audit/export?$query" -Headers $Headers -OutFile $exportPath -TimeoutSec $TimeoutSec -UseBasicParsing -ErrorAction Stop | Out-Null
    $csv = Get-Content -Raw -Encoding utf8 $exportPath
    Assert-ContainsToken -Text $csv -Token "eventId,sourceData" -Context "scenario $($Scenario.id) export"
    if ($fixedFixtureMatch) {
        foreach ($token in @($Scenario.expectedQueries.exportContains)) {
            Assert-ContainsToken -Text $csv -Token ([string]$token) -Context "scenario $($Scenario.id) export"
        }
    } else {
        Assert-ContainsToken -Text $csv -Token ([string]$match[0].id) -Context "scenario $($Scenario.id) runtime export"
    }
    Write-Host "Scenario $($Scenario.id) passed."
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
    Write-Step "0) Fixture Validation"
    $fixture = Read-SmokeFixture -Path $FixturePath
    Write-Host "Fixture scenarios: $(@($fixture.scenarios).Count)"

    $healthUri = "$BasePrefix/health"
    $loginUri = "$BasePrefix/v1/auth/login"
    $meUri = "$BasePrefix/v1/auth/me"
    $auditUri = "$BasePrefix/v1/audit?limit=20"
    $exportUri = "$BasePrefix/v1/audit/export?limit=20"

    Write-Step "1) Health Check"
    $health = Invoke-Json -Method "GET" -Uri $healthUri -Body $null -Headers @{}
    Write-Host "Health response: $($health | ConvertTo-Json -Depth 5)"

    Write-Step "2) Login Failure Audit Trigger"
    Invoke-ExpectedLoginFailure -LoginUri $loginUri

    Write-Step "3) Login"
    $login = Invoke-Json -Method "POST" -Uri $loginUri -Body @{ account = $Account; password = $Password } -Headers @{}
    $token = [string]$login.data.token
    if ([string]::IsNullOrWhiteSpace($token)) {
        throw "login succeeded but token is empty"
    }
    Write-Host "Login ok. token=$((Mask-Token -Token $token)) user=$($login.data.user.account)"

    $headers = @{ Authorization = "Bearer $token" }

    Write-Step "4) Current User"
    $me = Invoke-Json -Method "GET" -Uri $meUri -Body $null -Headers $headers
    Write-Host "Me: $($me | ConvertTo-Json -Depth 5)"

    Write-Step "5) Audit Query"
    $audit = Invoke-Json -Method "GET" -Uri $auditUri -Body $null -Headers $headers
    $count = @(Get-AuditItems -AuditResponse $audit).Count
    Write-Host "Audit records fetched: $count"

    Write-Step "6) Audit Export"
    if (!(Test-Path $ExportDir)) {
        New-Item -ItemType Directory -Path $ExportDir | Out-Null
    }
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $exportPath = Join-Path $ExportDir "audit-smoke-$timestamp.csv"
    Invoke-WebRequest -Method GET -Uri $exportUri -Headers $headers -OutFile $exportPath -TimeoutSec $TimeoutSec -UseBasicParsing -ErrorAction Stop | Out-Null
    $size = (Get-Item $exportPath).Length
    Write-Host "Export saved: $exportPath ($size bytes)"

    Write-Step "7) Fixture Scenario Assertions"
    foreach ($scenario in @($fixture.scenarios)) {
        Assert-Scenario -Scenario $scenario -BasePrefix $BasePrefix -Headers $headers -ExportDir $ExportDir
    }

    Write-Step "Result"
    Write-Host "Smoke check passed." -ForegroundColor Green
    exit 0
}
catch {
    Write-Host "`nSmoke check failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
