param(
    [switch]$RequireExternalDatabases,
    [switch]$Full
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repoRoot

function Invoke-Gate {
    param(
        [string]$Name,
        [string[]]$Arguments
    )

    Write-Host "[database acceptance] $Name"
    & go @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

try {
    if ($RequireExternalDatabases) {
        if ([string]::IsNullOrWhiteSpace($env:SKOLL_TEST_MYSQL_DSN)) {
            throw "Strict mode requires SKOLL_TEST_MYSQL_DSN"
        }
        if ([string]::IsNullOrWhiteSpace($env:SKOLL_TEST_POSTGRES_DSN)) {
            throw "Strict mode requires SKOLL_TEST_POSTGRES_DSN"
        }
    }

    Invoke-Gate "schema/model/migration completeness and upgrade" @(
        "test", "./internal/store/sql/gormrepo",
        "-run", "TestPharmaSchemaBaselineHasModelsAndMigrations|TestPharmaSchemaUpgradePreservesExistingMasterData|TestPharmaMigrationSQLMySQL|TestPharmaMigrationSQLPostgreSQL",
        "-count=1", "-v"
    )
    Invoke-Gate "seed, restart, backup, and restore" @(
        "test", "./internal/service/pharmaoa",
        "-run", "TestPharmaDatabaseAcceptanceSeedRestartAndBackupRestore",
        "-count=1", "-v"
    )
    Invoke-Gate "SQLite/MySQL/PostgreSQL repository contracts" @(
        "test", "./internal/store/sql/gormrepo",
        "-run", "TestPharmaMasterRepositories|TestPharmaOrderRepositories|TestPharmaWorkflowRecordRepositories",
        "-count=1", "-v"
    )

    if ($Full) {
        Invoke-Gate "full Go regression" @("test", "./...", "-count=1")
        Invoke-Gate "Go vet" @("vet", "./...")
    }

    if ($RequireExternalDatabases) {
        Write-Host "[database acceptance] all gates passed, including MySQL and PostgreSQL."
    }
    else {
        Write-Host "[database acceptance] local gates passed; external database suites may have been skipped."
    }
}
finally {
    Pop-Location
}
