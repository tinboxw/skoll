Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ports = @(8080, 5173)
foreach ($port in $ports) {
    $listeners = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    foreach ($listener in $listeners) {
        try {
            Stop-Process -Id $listener.OwningProcess -Force -ErrorAction Stop
            Write-Host ("Stopped PID={0} on port {1}" -f $listener.OwningProcess, $port)
        } catch {
            Write-Host ("Failed to stop PID={0} on port {1}: {2}" -f $listener.OwningProcess, $port, $_.Exception.Message)
        }
    }
}

Write-Host "Done."
