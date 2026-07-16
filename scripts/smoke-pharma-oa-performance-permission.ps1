param(
    [int]$Count = 1,
    [ValidateSet("zh-CN", "en-US")]
    [string]$Locale = "zh-CN"
)

$ErrorActionPreference = "Stop"

$messages = if ($Locale -eq "en-US") {
    @{
        Start = "=== Pharma OA performance and permission acceptance ==="
        Passed = "Pharma OA performance and permission acceptance passed."
    }
} else {
    @{
        Start = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("PT09IOWMu+iNryBPQSDmgKfog73kuI7mnYPpmZDpqozmlLYgPT09"))
        Passed = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("5Yy76I2vIE9BIOaAp+iDveS4juadg+mZkOmqjOaUtumAmui/h+OAgg=="))
    }
}

function Invoke-GoGate {
    param([string[]]$Arguments)

    & go @Arguments
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}

Write-Host $messages.Start
Invoke-GoGate -Arguments @("test", "./internal/service/pharmaoa", "-run", "^TestPharmaOALargeListPaginationAndCommonQueries$", "-count", ([string]$Count), "-v")
Invoke-GoGate -Arguments @("test", "./internal/bootstrap", "-run", "Test(LoadAuthPolicy|AuthPolicy|PharmaOACriticalPermissionMapping|AuthGuardMiddlewareEnforcesPharmaOACriticalPermission)", "-count", ([string]$Count), "-v")
Invoke-GoGate -Arguments @("test", "./internal/handler/http/v1/pharmaoa", "-run", "Test(ActorIDFromRequestPrefersJWTSubject|CustomerScopeFromRequestUsesJWTIdentity|NormalizePagination)", "-count", ([string]$Count), "-v")
Invoke-GoGate -Arguments @("test", "./tests/benchmark", "-run", "^$", "-bench", "BenchmarkPharmaOA", "-benchtime", "100x", "-count", ([string]$Count))
Write-Host $messages.Passed
