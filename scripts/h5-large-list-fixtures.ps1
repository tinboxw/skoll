param(
    [ValidateSet("seed", "cleanup")]
    [string]$Action = "seed",
    [ValidateRange(100, 1000)]
    [int]$Rows = 125,
    [ValidateSet("zh-CN", "en-US")]
    [string]$Locale = "zh-CN",
    [string]$HostName = "127.0.0.1",
    [int]$Port = 3306,
    [string]$User = "root",
    [string]$Password = "root",
    [string]$Database = "skoll"
)

$ErrorActionPreference = "Stop"

function Decode-UTF8([string]$Value) {
    return [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($Value))
}

$messages = if ($Locale -eq "zh-CN") {
    @{
        Seed = Decode-UTF8 "5q2j5Zyo5YaZ5YWlIEg1IOWkp+WIl+ihqOa1j+iniOWZqOmqjOaUtuWkueWFty4uLg=="
        Cleanup = Decode-UTF8 "5q2j5Zyo5riF55CGIEg1IOWkp+WIl+ihqOa1j+iniOWZqOmqjOaUtuWkueWFty4uLg=="
        SeedDone = Decode-UTF8 "SDUg5aSn5YiX6KGo5rWP6KeI5Zmo6aqM5pS25aS55YW35bey5bCx57uq44CC"
        CleanupDone = Decode-UTF8 "SDUg5aSn5YiX6KGo5rWP6KeI5Zmo6aqM5pS25aS55YW35bey5riF55CG44CC"
    }
} else {
    @{ Seed = "Seeding H5 large-list browser fixtures..."; Cleanup = "Cleaning H5 large-list browser fixtures..."; SeedDone = "H5 large-list browser fixtures are ready."; CleanupDone = "H5 large-list browser fixtures were cleaned." }
}

Write-Host $(if ($Action -eq "seed") { $messages.Seed } else { $messages.Cleanup })

$statements = [Collections.Generic.List[string]]::new()
$statements.Add("START TRANSACTION;")
$statements.Add("DELETE FROM pharma_oa_customers WHERE id LIKE 'h5-list-%';")
$statements.Add("DELETE FROM pharma_oa_employees WHERE id LIKE 'h5-list-%';")

if ($Action -eq "seed") {
    $employeeValues = for ($index = 1; $index -le $Rows; $index++) {
        $number = $index.ToString("0000")
        "('h5-list-employee-$number','H5E$number','H5 Employee $number','quality','qa','','h5e$number@example.test','active','','[]',UTC_TIMESTAMP(3),UTC_TIMESTAMP(3),'h5-browser','h5-browser')"
    }
    $customerValues = for ($index = 1; $index -le $Rows; $index++) {
        $number = $index.ToString("0000")
        $region = @("East", "South", "West", "North")[($index - 1) % 4]
        "('h5-list-customer-$number','H5C$number','H5 Customer $number','$region','org-root','admin',3,'active','','[]','[]',UTC_TIMESTAMP(3),UTC_TIMESTAMP(3),'h5-browser','h5-browser')"
    }
    $statements.Add("INSERT INTO pharma_oa_employees (id,code,name,department_id,position_id,phone,email,status,leave_reason,certificates_json,created_at,updated_at,created_by,updated_by) VALUES " + ($employeeValues -join ",") + ";")
    $statements.Add("INSERT INTO pharma_oa_customers (id,code,name,region,organization_id,owner_id,rating,status,disable_reason,contacts_json,qualifications_json,created_at,updated_at,created_by,updated_by) VALUES " + ($customerValues -join ",") + ";")
}

$statements.Add("COMMIT;")
$statements.Add("SELECT (SELECT COUNT(*) FROM pharma_oa_employees WHERE id LIKE 'h5-list-%') AS employees, (SELECT COUNT(*) FROM pharma_oa_customers WHERE id LIKE 'h5-list-%') AS customers;")

$previousPassword = $env:MYSQL_PWD
try {
    $env:MYSQL_PWD = $Password
    ($statements -join [Environment]::NewLine) | mysql.exe --host=$HostName --port=$Port --user=$User --batch --skip-column-names $Database
    if ($LASTEXITCODE -ne 0) {
        throw "mysql fixture command failed with exit code $LASTEXITCODE"
    }
} finally {
    $env:MYSQL_PWD = $previousPassword
}

Write-Host $(if ($Action -eq "seed") { $messages.SeedDone } else { $messages.CleanupDone })
