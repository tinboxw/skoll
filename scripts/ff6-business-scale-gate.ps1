$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$temporaryRoot = Join-Path (Split-Path -Parent $repoRoot) (".skoll-ff6-business-scale-" + [guid]::NewGuid().ToString("N"))
$binarySuffix = if ([System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT) { ".exe" } else { "" }
$qualityBinary = Join-Path $temporaryRoot ("generator-quality" + $binarySuffix)
$startedAt = Get-Date

function Invoke-Gate {
    param(
        [string]$Name,
        [string]$WorkingDirectory,
        [string]$Command,
        [string[]]$Arguments
    )

    Write-Host ""
    Write-Host "==> $Name"
    Push-Location $WorkingDirectory
    try {
        & $Command @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$Name failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }
}

New-Item -ItemType Directory -Path $temporaryRoot | Out-Null
try {
    Invoke-Gate `
        -Name "Multi-plugin load, attack, outage, recovery, and isolation" `
        -WorkingDirectory $repoRoot `
        -Command "go" `
        -Arguments @(
            "test", "./internal/plugin",
            "-run", "^TestBusinessScaleFrameworkAcceptance$",
            "-count=20", "-v", "-timeout=5m"
        )

    $previousEquipmentE2E = $env:SKOLL_EQUIPMENT_PLUGIN_E2E
    $previousPharmaE2E = $env:SKOLL_PHARMA_OA_E2E
    $env:SKOLL_EQUIPMENT_PLUGIN_E2E = "1"
    $env:SKOLL_PHARMA_OA_E2E = "1"
    try {
        Invoke-Gate `
            -Name "Current plugin process and interoperability E2E" `
            -WorkingDirectory $repoRoot `
            -Command "go" `
            -Arguments @(
                "test", "./internal/plugin", "./internal/bootstrap",
                "-run", "^(TestReliableCurrentEventInteroperabilityEndToEnd|TestEquipmentMaintenancePackagedLifecycleE2E|TestPharmaOAPackagedBusinessLifecycleE2E|TestIndependentPluginDataStoreProcessLifecycleE2E|TestIndependentPluginGovernedWorkflowProcessE2E|TestPluginRuntimeMilestoneEndToEnd|TestPluginRequestQuotaIsolatesConcurrentPluginsAndRecovers)$",
                "-count=1", "-v", "-timeout=15m"
            )
    }
    finally {
        $env:SKOLL_EQUIPMENT_PLUGIN_E2E = $previousEquipmentE2E
        $env:SKOLL_PHARMA_OA_E2E = $previousPharmaE2E
    }

    Invoke-Gate `
        -Name "Business-scale race detector" `
        -WorkingDirectory $repoRoot `
        -Command "go" `
        -Arguments @(
            "test", "-race", "./internal/plugin",
            "-run", "^TestBusinessScaleFrameworkAcceptance$",
            "-count=1", "-v", "-timeout=5m"
        )

    Invoke-Gate `
        -Name "Compile zero-edit generated-plugin quality gate" `
        -WorkingDirectory $repoRoot `
        -Command "go" `
        -Arguments @(
            "test", "-c", "-tags=pluginquality",
            "-o", $qualityBinary,
            "./internal/service/generator"
        )

    Invoke-Gate `
        -Name "Zero-edit generated-plugin quality gate" `
        -WorkingDirectory (Join-Path $repoRoot "internal/service/generator") `
        -Command $qualityBinary `
        -Arguments @(
            "-test.run", "^TestGeneratedPluginZeroEditQualityGate$",
            "-test.count=1", "-test.v", "-test.timeout=15m"
        )

    Invoke-Gate `
        -Name "Full Go repository regression" `
        -WorkingDirectory $repoRoot `
        -Command "go" `
        -Arguments @("test", "./...", "-count=1", "-timeout=30m")

    $publicOpenAPI = Get-FileHash (Join-Path $repoRoot "docs/api/openapi.yaml") -Algorithm SHA256
    $embeddedOpenAPI = Get-FileHash (Join-Path $repoRoot "internal/handler/http/openapi.yaml") -Algorithm SHA256
    if ($publicOpenAPI.Hash -ne $embeddedOpenAPI.Hash) {
        throw "OpenAPI copies differ."
    }

    Invoke-Gate `
        -Name "Repository whitespace validation" `
        -WorkingDirectory $repoRoot `
        -Command "git" `
        -Arguments @("diff", "--check")

    $elapsed = (Get-Date) - $startedAt
    Write-Host ""
    Write-Host "FF6 business-scale quality gate passed in $([math]::Round($elapsed.TotalSeconds, 1)) seconds."
    Write-Host "OpenAPI SHA256: $($publicOpenAPI.Hash)"
}
finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        Remove-Item -LiteralPath $temporaryRoot -Recurse -Force
    }
}
