#requires -Version 5.1
# Deterministic checks using local fixtures; no real network queries or Pester.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$observer = Join-Path $PSScriptRoot 'netwatch.ps1'
$testDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ('pitch-netwatch-test-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $testDirectory

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}

$created = [datetime]'2026-01-01T00:00:00Z'
$fixtureProcesses = @(
    [pscustomobject]@{ ProcessId = 4244; ParentProcessId = 4243; Name = 'grandchild.exe'; CreationDate = $created.AddSeconds(2) }
    [pscustomobject]@{ ProcessId = 4242; ParentProcessId = 1; Name = 'pitch.exe'; CreationDate = $created }
    [pscustomobject]@{ ProcessId = 4243; ParentProcessId = 4242; Name = 'msedgewebview2.exe'; CreationDate = $created.AddSeconds(1) }
    [pscustomobject]@{ ProcessId = 4245; ParentProcessId = 4242; Name = 'old-parent-pid.exe'; CreationDate = $created.AddSeconds(-1) }
    [pscustomobject]@{ ProcessId = 9999; ParentProcessId = 1; Name = 'unrelated.exe'; CreationDate = $created }
)
$failTcp = $false
$replaceRoot = $false
$fixtureState = @{ CimCalls = 0 }

function Get-CimInstance {
    [CmdletBinding()]
    param([string]$ClassName)
    $fixtureState.CimCalls++
    if ($replaceRoot -and $fixtureState.CimCalls -gt 1) {
        [pscustomobject]@{ ProcessId = 4242; ParentProcessId = 1; Name = 'replacement.exe'; CreationDate = $created.AddMinutes(1) }
    } else {
        $fixtureProcesses
    }
}

function Get-NetTCPConnection {
    [CmdletBinding()]
    param()
    if ($failTcp) { throw 'Simulated access denied' }
    foreach ($ownerId in @(4242, 4243, 4244, 4245, 9999)) {
        [pscustomobject]@{
            OwningProcess = $ownerId; LocalAddress = '127.0.0.1'; LocalPort = 1234
            RemoteAddress = '203.0.113.10'; RemotePort = 443; State = 'Established'
        }
    }
}

function Get-NetUDPEndpoint {
    [CmdletBinding()]
    param()
    [pscustomobject]@{ OwningProcess = 4244; LocalAddress = '0.0.0.0'; LocalPort = 1234 }
    [pscustomobject]@{ OwningProcess = 9999; LocalAddress = '0.0.0.0'; LocalPort = 5678 }
}

try {
    $normalPath = Join-Path $testDirectory 'normal.json'
    $null = & $observer -ProcessId 4242 -DurationSeconds 1 -IntervalMilliseconds 1000 -OutputPath $normalPath
    $report = Get-Content -LiteralPath $normalPath -Raw | ConvertFrom-Json
    Assert-True (-not $report.Incomplete) 'A successful observation was marked incomplete.'
    Assert-True ($report.Samples.Count -gt 0) 'No samples were saved.'
    $sample = $report.Samples[0]
    Assert-True (($sample.Processes.ProcessId -join ',') -eq '4242,4243,4244') 'Descendants were missed, or an unrelated/stale-parent process was included.'
    Assert-True (($sample.TCP.OwningProcess -join ',') -eq '4242,4243,4244') 'TCP ownership filtering failed.'
    Assert-True ($sample.UDP.Count -eq 1 -and $sample.UDP[0].OwningProcess -eq 4244) 'UDP ownership filtering failed.'

    $oldReport = [System.IO.File]::ReadAllText($normalPath)
    $rejected = $false
    try {
        $null = & $observer -ProcessId 4242 -DurationSeconds 1 -OutputPath $normalPath
    } catch { $rejected = $true }
    Assert-True $rejected 'An existing report was accepted for overwrite.'
    Assert-True ([System.IO.File]::ReadAllText($normalPath) -eq $oldReport) 'An existing report was modified.'

    $failTcp = $true
    $failurePath = Join-Path $testDirectory 'failure.json'
    $rejected = $false
    try {
        $null = & $observer -ProcessId 4242 -DurationSeconds 1 -OutputPath $failurePath
    } catch { $rejected = $true }
    Assert-True $rejected 'A query failure was silently treated as an empty observation.'
    $report = Get-Content -LiteralPath $failurePath -Raw | ConvertFrom-Json
    Assert-True ($report.Incomplete -and $report.Errors.Count -eq 1 -and $report.TerminationReason -eq 'query-failed') 'The query failure was not recorded.'
    $failTcp = $false

    $replaceRoot = $true
    $fixtureState.CimCalls = 0
    $reusedPath = Join-Path $testDirectory 'reused.json'
    $null = & $observer -ProcessId 4242 -DurationSeconds 1 -OutputPath $reusedPath
    $report = Get-Content -LiteralPath $reusedPath -Raw | ConvertFrom-Json
    Assert-True ($report.Incomplete -and $report.TerminationReason -eq 'root-exited' -and $report.Samples.Count -eq 0) 'A reused root PID was observed as the original process.'

    $replaceRoot = $false
    $missingPath = Join-Path $testDirectory 'missing.json'
    $rejected = $false
    try {
        $null = & $observer -ProcessId 8888 -DurationSeconds 1 -OutputPath $missingPath
    } catch { $rejected = $true }
    Assert-True ($rejected -and -not (Test-Path -LiteralPath $missingPath)) 'A nonexistent process produced a misleading report.'
    Write-Output 'netwatch: all checks passed'
} finally {
    Get-ChildItem -LiteralPath $testDirectory -File | ForEach-Object { Remove-Item -LiteralPath $_.FullName }
    Remove-Item -LiteralPath $testDirectory
}
