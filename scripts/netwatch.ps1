#requires -Version 5.1
<#
.SYNOPSIS
Samples TCP connections and UDP endpoints for a process and its current descendants.
.DESCRIPTION
Development-only observer. Does not launch processes or change firewall settings.
Polling can miss short-lived connections; UDP endpoints do not expose destinations.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateRange(1, 2147483647)]
    [int]$ProcessId,

    [ValidateRange(1, 3600)]
    [int]$DurationSeconds = 30,

    [ValidateRange(100, 60000)]
    [int]$IntervalMilliseconds = 500,

    [ValidateNotNullOrEmpty()]
    [string]$OutputPath = (Join-Path $PSScriptRoot ('../data/network/' + [guid]::NewGuid().ToString('N') + '.json'))
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'process-identity.ps1')

if (Test-Path -LiteralPath $OutputPath) {
    throw "Output already exists: $OutputPath"
}

$initialProcesses = @(Get-CimInstance -ClassName Win32_Process)
$root = $initialProcesses | Where-Object { $_.ProcessId -eq $ProcessId } | Select-Object -First 1
if ($null -eq $root) {
    throw "Process $ProcessId is not running."
}
$rootCreated = $root.CreationDate
$samples = [System.Collections.Generic.List[object]]::new()
$queryErrors = [System.Collections.Generic.List[object]]::new()
$startedAt = [datetime]::UtcNow
$watch = [System.Diagnostics.Stopwatch]::StartNew()
$terminationReason = 'duration'

while ($watch.Elapsed.TotalSeconds -lt $DurationSeconds) {
    $sampleTime = [datetime]::UtcNow.ToString('o')
    try {
        $processes = @(Get-CimInstance -ClassName Win32_Process)
        $currentRoot = $processes | Where-Object { $_.ProcessId -eq $ProcessId } | Select-Object -First 1
        # Windows reuses PIDs. Never attach to a replacement process.
        if ($null -eq $currentRoot -or $currentRoot.CreationDate -ne $rootCreated) {
            $terminationReason = 'root-exited'
            break
        }

        $tree = @{}
        $tree[$ProcessId] = $currentRoot
        do {
            $added = $false
            foreach ($process in $processes) {
                $parentId = [int]$process.ParentProcessId
                $childId = [int]$process.ProcessId
                if (-not $tree.ContainsKey($childId) -and $tree.ContainsKey($parentId) -and
                    $process.CreationDate -ge $tree[$parentId].CreationDate) {
                    $tree[$childId] = $process
                    $added = $true
                }
            }
        } while ($added)

        $tcp = @(Get-NetTCPConnection -ErrorAction Stop | Where-Object { $tree.ContainsKey([int]$_.OwningProcess) })
        $udp = @(Get-NetUDPEndpoint -ErrorAction Stop | Where-Object { $tree.ContainsKey([int]$_.OwningProcess) })
        $samples.Add([pscustomobject]@{
            TimestampUtc = $sampleTime
            FinishedAtUtc = [datetime]::UtcNow.ToString('o')
            Processes = @($tree.Values | Sort-Object ProcessId | Select-Object ProcessId, ParentProcessId, Name, CreationDate, @{Name='ProcessStartKey';Expression={ [PitchProcessIdentity]::Read([int]$_.ProcessId,[datetime]$_.CreationDate) }})
            TCP = @($tcp | Select-Object OwningProcess, LocalAddress, LocalPort, RemoteAddress, RemotePort, State)
            UDP = @($udp | Select-Object OwningProcess, LocalAddress, LocalPort)
        })
    } catch {
        $queryErrors.Add([pscustomobject]@{ TimestampUtc = $sampleTime; Message = $_.Exception.Message })
        $terminationReason = 'query-failed'
        break
    }

    $remainingMs = ($DurationSeconds * 1000) - $watch.Elapsed.TotalMilliseconds
    if ($remainingMs -gt 0) {
        Start-Sleep -Milliseconds ([int][math]::Min($IntervalMilliseconds, [math]::Ceiling($remainingMs)))
    }
}

$report = [pscustomobject]@{
    RootProcessId = $ProcessId
    RootProcessName = $root.Name
    RootCreatedAt = $rootCreated
    StartedAtUtc = $startedAt.ToString('o')
    FinishedAtUtc = [datetime]::UtcNow.ToString('o')
    RequestedDurationSeconds = $DurationSeconds
    IntervalMilliseconds = $IntervalMilliseconds
    TerminationReason = $terminationReason
    Incomplete = ($terminationReason -ne 'duration')
    Limitations = @('Polling may miss short-lived connections and processes.', 'UDP endpoints do not expose remote destinations.', 'An empty report does not prove the absence of network traffic.')
    Samples = $samples.ToArray()
    Errors = $queryErrors.ToArray()
}

$absoluteOutput = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($OutputPath)
$null = [System.IO.Directory]::CreateDirectory([System.IO.Path]::GetDirectoryName($absoluteOutput))
# CreateNew protects existing reports even if another writer wins the race.
$stream = [System.IO.File]::Open($absoluteOutput, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write)
$writer = [System.IO.StreamWriter]::new($stream, [System.Text.UTF8Encoding]::new($false))
try {
    $writer.WriteLine(($report | ConvertTo-Json -Depth 8))
} finally {
    $writer.Dispose()
}
Write-Output $absoluteOutput
if ($queryErrors.Count -gt 0) {
    throw "Network observation failed; the incomplete report is saved to $absoluteOutput"
}
