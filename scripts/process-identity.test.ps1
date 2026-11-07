$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'process-identity.ps1')
$current=[Diagnostics.Process]::GetCurrentProcess()
$key=[PitchProcessIdentity]::Read($current.Id,$current.StartTime)
if([string]::IsNullOrEmpty($key)){throw 'Cannot read current process identity'}
if([PitchProcessIdentity]::Read($current.Id,$current.StartTime.AddSeconds(-1))){throw 'Accepted reused PID with a different creation time'}
if([PitchProcessIdentity]::Read(0,[datetime]::UtcNow)){throw 'Accepted invalid process'}
Write-Output 'Process identity checks passed'
