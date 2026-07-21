param(
    [ValidateSet("Doing", "Review", "Done")]
    [string]$ExpectedWorkItemStatus = "Doing",

    [ValidateSet("Doing", "Done")]
    [string]$ExpectedParentStatus = "Doing"
)

$ErrorActionPreference = "Stop"

function Assert-Condition {
    param(
        [bool]$Condition,
        [string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

function Read-JsonFile {
    param([string]$Path)

    Assert-Condition (Test-Path -LiteralPath $Path -PathType Leaf) "Missing evidence: $Path"
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$currentDir = Join-Path $repoRoot "docs/refactor/current"
$workItemsPath = Join-Path $currentDir "hardening_work_items_2026-07-18.md"
$taskBoardPath = Join-Path $currentDir "hardening_task_board_2026-07-18.md"
$acceptancePath = Join-Path $currentDir "hardening_acceptance_log_2026-07-18.md"

$workItems = Get-Content -LiteralPath $workItemsPath -Raw -Encoding UTF8
$taskBoard = Get-Content -LiteralPath $taskBoardPath -Raw -Encoding UTF8
$acceptance = Get-Content -LiteralPath $acceptancePath -Raw -Encoding UTF8

$rows = [regex]::Matches($workItems, '(?m)^\| (H\d-\d{2}) \|.*?\| (Todo|Doing|Review|Done|Failed|Blocked) \|$')
Assert-Condition ($rows.Count -eq 25) "Expected 25 hardening Work Items, found $($rows.Count)."

$seen = @{}
foreach ($row in $rows) {
    $id = $row.Groups[1].Value
    $status = $row.Groups[2].Value
    Assert-Condition (-not $seen.ContainsKey($id)) "Duplicate Work Item row: $id"
    $seen[$id] = $status

    $expected = if ($id -eq "H6-03") { $ExpectedWorkItemStatus } else { "Done" }
    Assert-Condition ($status -eq $expected) "Work Item $id is $status; expected $expected."
}

foreach ($id in 1..6) {
    $parentId = "H$id"
    $match = [regex]::Match($taskBoard, "(?m)^\| $parentId \|.*?\| (Todo|Doing|Review|Done|Failed|Blocked) \|$")
    Assert-Condition $match.Success "Missing parent task row: $parentId"
    $expected = if ($parentId -eq "H6") { $ExpectedParentStatus } else { "Done" }
    Assert-Condition ($match.Groups[1].Value -eq $expected) "Parent task $parentId is $($match.Groups[1].Value); expected $expected."
}

$acceptedIds = [regex]::Matches($acceptance, '(?m)^## (H\d-\d{2})\b') |
    ForEach-Object { $_.Groups[1].Value } |
    Sort-Object -Unique
$requiredAcceptance = $seen.Keys | Where-Object {
    $_ -ne "H6-03" -or $ExpectedWorkItemStatus -eq "Done"
}
foreach ($id in $requiredAcceptance) {
    Assert-Condition ($acceptedIds -contains $id) "Acceptance log is missing $id."
}

$evidenceDir = Join-Path $currentDir "evidence"
$matrix = Read-JsonFile (Join-Path $evidenceDir "h4-04/matrix.json")
Assert-Condition ($matrix.summary.total -eq 28) "H4-04 matrix must contain 28 checks."
Assert-Condition ($matrix.summary.passed -eq 28) "H4-04 matrix contains failed checks."

foreach ($imageName in @(
    "zh-CN-desktop-matrix.png",
    "zh-CN-mobile-matrix.png",
    "en-US-desktop-matrix.png",
    "en-US-mobile-matrix.png"
)) {
    $imagePath = Join-Path $evidenceDir "h4-04/$imageName"
    Assert-Condition ((Test-Path -LiteralPath $imagePath -PathType Leaf) -and ((Get-Item -LiteralPath $imagePath).Length -gt 0)) "Missing H4-04 image: $imageName"
}

$passedEvidence = @(
    "h5-01/mysql-query-benchmark.json",
    "h5-02/browser-matrix.json",
    "h5-03/pharma-job-soak.json",
    "h5-03/business-event-soak.json",
    "h5-04/performance-baseline.json",
    "h6-01/deployment-recovery-smoke.json",
    "h6-03/runtime-review.json"
)
foreach ($relativePath in $passedEvidence) {
    $evidence = Read-JsonFile (Join-Path $evidenceDir $relativePath)
    Assert-Condition ($evidence.passed -eq $true) "Evidence did not pass: $relativePath"
}

foreach ($reportPath in @(
    (Join-Path $currentDir "performance_capacity_baseline_2026-07-21.md"),
    (Join-Path $currentDir "deployment_recovery_rehearsal_2026-07-21.md"),
    (Join-Path $repoRoot "docs/user/release-boundaries.md"),
    (Join-Path $repoRoot "docs/user/release-boundaries.en.md")
)) {
    Assert-Condition (Test-Path -LiteralPath $reportPath -PathType Leaf) "Missing release report: $reportPath"
}

if ($ExpectedWorkItemStatus -eq "Done") {
    foreach ($closeoutPath in @(
        (Join-Path $currentDir "hardening_closeout_2026-07-21.md"),
        (Join-Path $evidenceDir "h6-03/final-quality-gate.json"),
        (Join-Path $evidenceDir "h6-03/README.md")
    )) {
        Assert-Condition (Test-Path -LiteralPath $closeoutPath -PathType Leaf) "Missing closeout artifact: $closeoutPath"
    }

    $finalGate = Read-JsonFile (Join-Path $evidenceDir "h6-03/final-quality-gate.json")
    Assert-Condition ($finalGate.passed -eq $true) "Final quality gate evidence did not pass."
    Assert-Condition ($finalGate.locale -eq "zh-CN") "Final quality gate must use the Chinese-default locale."
    Assert-Condition ($finalGate.summary.acceptedWorkItems -eq 25) "Final quality gate must report 25 accepted Work Items."
    Assert-Condition (@($finalGate.gates | Where-Object { $_.passed -ne $true }).Count -eq 0) "Final quality gate contains a failed gate."

    foreach ($markdownPath in @(
        (Join-Path $currentDir "hardening_closeout_2026-07-21.md"),
        (Join-Path $evidenceDir "h6-03/README.md")
    )) {
        $markdown = Get-Content -LiteralPath $markdownPath -Raw -Encoding UTF8
        foreach ($link in [regex]::Matches($markdown, '\]\(([^)]+)\)')) {
            $target = $link.Groups[1].Value.Split('#')[0]
            if ([string]::IsNullOrWhiteSpace($target) -or $target -match '^[a-z]+:') {
                continue
            }
            $resolved = Join-Path (Split-Path -Parent $markdownPath) ([Uri]::UnescapeDataString($target))
            Assert-Condition (Test-Path -LiteralPath $resolved) "Broken local link in $markdownPath`: $target"
        }
    }
}

$archiveChanges = & git -C $repoRoot diff --name-only -- "docs/refactor/old/"
if ($LASTEXITCODE -ne 0) {
    throw "Unable to inspect archive changes."
}
Assert-Condition (-not $archiveChanges) "Archived refactor evidence was modified."

Write-Host "H6-03 closeout consistency audit passed: $($rows.Count) Work Items, $($acceptedIds.Count) accepted IDs."
