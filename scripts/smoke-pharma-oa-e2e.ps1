param(
    [int]$Count = 1,
    [ValidateSet("zh-CN", "en-US")]
    [string]$Locale = "zh-CN"
)

$ErrorActionPreference = "Stop"

$utf8 = [Text.Encoding]::UTF8
$messages = @{
    "zh-CN" = @{
        Start = $utf8.GetString([Convert]::FromBase64String("PT09IOWMu+iNryBPQSDlrozmlbTnq6/liLDnq6/pqozmlLYgPT09"))
        Passed = $utf8.GetString([Convert]::FromBase64String("5Yy76I2vIE9BIOWujOaVtOerr+WIsOerr+mqjOaUtumAmui/h+OAgg=="))
    }
    "en-US" = @{
        Start = "=== Pharma OA full end-to-end acceptance smoke ==="
        Passed = "Pharma OA full end-to-end acceptance smoke passed."
    }
}

Write-Host $messages[$Locale].Start
& go test ./tests/integration -run "^TestPharmaOAEndToEndAcceptanceSmoke$" -count ([string]$Count) -v
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host $messages[$Locale].Passed
