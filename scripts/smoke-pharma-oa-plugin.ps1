param(
    [string]$Package = "./internal/plugin",
    [string]$Run = "TestPharmaOA",
    [int]$Count = 1,
    [ValidateSet("zh-CN", "en-US")]
    [string]$Locale = "zh-CN"
)

$ErrorActionPreference = "Stop"

$messages = if ($Locale -eq "en-US") {
    @{
        Start = "=== Pharma OA plugin lifecycle smoke ==="
        Passed = "Pharma OA plugin lifecycle smoke passed."
    }
} else {
    @{
        Start = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("PT09IOWMu+iNryBPQSDmj5Lku7bnlJ/lkb3lkajmnJ/pqozmlLYgPT09"))
        Passed = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String("5Yy76I2vIE9BIOaPkuS7tueUn+WRveWRqOacn+mqjOaUtumAmui/h+OAgg=="))
    }
}

Write-Host $messages.Start
$goArgs = @("test", $Package, "-run", $Run, "-count", ([string]$Count), "-v")
& go @goArgs
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host $messages.Passed
