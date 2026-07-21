param()

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot

function Read-UTF8([string]$RelativePath) {
    $path = Join-Path $repoRoot $RelativePath
    if (-not (Test-Path -LiteralPath $path)) { throw "Required document is missing: $RelativePath" }
    return [System.IO.File]::ReadAllText($path, [System.Text.Encoding]::UTF8)
}

function Assert-Contains([string]$Content, [string]$Marker, [string]$Document) {
    if (-not $Content.Contains($Marker)) { throw "$Document is missing required marker: $Marker" }
}

function Assert-LocalLinks([string]$RelativePath) {
    $path = Join-Path $repoRoot $RelativePath
    $content = Read-UTF8 $RelativePath
    foreach ($match in [regex]::Matches($content, '\[[^\]]+\]\(([^)]+)\)')) {
        $target = $match.Groups[1].Value.Trim('<', '>')
        if ($target -match '^(https?://|mailto:|#)') { continue }
        $target = ($target -split '#')[0]
        $resolved = [System.IO.Path]::GetFullPath((Join-Path (Split-Path -Parent $path) $target))
        if (-not (Test-Path -LiteralPath $resolved)) { throw "Broken local link in $RelativePath`: $target" }
    }
}

$zhPath = "docs/user/release-boundaries.md"
$enPath = "docs/user/release-boundaries.en.md"
$zh = Read-UTF8 $zhPath
$en = Read-UTF8 $enPath
$zhSections = [regex]::Matches($zh, '(?m)^## ').Count
$enSections = [regex]::Matches($en, '(?m)^## ').Count
if ($zhSections -lt 7 -or $zhSections -ne $enSections) {
    throw "Bilingual boundary section count mismatch: zh=$zhSections en=$enSections"
}

foreach ($marker in @('GSP', 'GMP', 'DEMO-*', 'SKOLL_SECURITY_JWT_SECRET', 'MIT License', 'SBOM', '/app/data', '/app/plugins', 'SECURITY.md', 'memory', 'mysql', 'postgres')) {
    Assert-Contains $zh $marker $zhPath
    Assert-Contains $en $marker $enPath
}
Assert-Contains $zh 'release-boundaries.en.md' $zhPath
Assert-Contains $en 'release-boundaries.md' $enPath

$pluginZH = Read-UTF8 "plugins/pharma_oa/README.md"
$pluginEN = Read-UTF8 "plugins/pharma_oa/README.en.md"
Assert-Contains $pluginZH '../../docs/user/release-boundaries.md' 'plugins/pharma_oa/README.md'
Assert-Contains $pluginEN '../../docs/user/release-boundaries.en.md' 'plugins/pharma_oa/README.en.md'
if ($pluginZH.Contains('current Pharma OA business repositories are process-local') -or $pluginEN.Contains('Pharma OA repositories are currently process-local')) {
    throw "A stale process-local persistence boundary remains in the plugin guides"
}

$checklist = Read-UTF8 "docs/release-checklist.md"
foreach ($marker in @('Third-party licenses', 'Sample position', 'Demo data', 'Security channel', 'Support boundary', 'Bilingual boundary')) {
    Assert-Contains $checklist $marker 'docs/release-checklist.md'
}
if ($checklist.Contains('docs/refactor/old/')) { throw "Release checklist still directs current progress into archived evidence" }
$license = Read-UTF8 "LICENSE"
if (-not $license.StartsWith('MIT License')) { throw "Root LICENSE is not the documented MIT license" }

foreach ($document in @(
    $zhPath,
    $enPath,
    'docs/user/README.md',
    'plugins/pharma_oa/README.md',
    'plugins/pharma_oa/README.en.md',
    'docs/release-checklist.md'
)) {
    Assert-LocalLinks $document
}

Write-Host "H6-02 bilingual release boundary and link audit passed"
