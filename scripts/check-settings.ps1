#requires -Version 5.1
<#
.SYNOPSIS
Checks real Windows settings UI and restart using an isolated diagnostic build.
.DESCRIPTION
Uses a Go overlay to inject check-settings.js and receive its result through
the native WebView bridge. The normal source and application have no test hook.
All test data remains under the ignored data/ directory. No debugging port.
#>
[CmdletBinding()]
param()
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$runRoot = Join-Path $repo ('data/ui-' + [guid]::NewGuid().ToString('N').Substring(0, 12))
$null = New-Item -ItemType Directory -Path $runRoot
$encoding = [System.Text.UTF8Encoding]::new($false)

$main = Get-Content -LiteralPath (Join-Path $repo 'main.go') -Raw -Encoding UTF8
foreach ($marker in @('"embed"', 'Name:        "Pitch",', 'app.Window.NewWithOptions(', 'return app.Run()')) {
    if (-not $main.Contains($marker)) { throw "Diagnostic overlay needs updating: missing $marker" }
}
$main = $main.Replace('"embed"', '"embed"' + "`n" + @'
    "encoding/json"
    "path/filepath"
    "strings"
    "time"
    "github.com/wailsapp/wails/v3/pkg/events"
'@)
$main = $main.Replace('Name:        "Pitch",', 'Name:        "Pitch",' + "`n" + @'
        RawMessageHandler: func(_ application.Window, message string, _ *application.OriginInfo) {
            if strings.HasPrefix(message, "pitch-stage-one:") {
                _ = os.WriteFile(filepath.Join(dataDir, "gui-result.json"), []byte(strings.TrimPrefix(message, "pitch-stage-one:")), 0600)
                go func() { time.Sleep(time.Second); application.Get().Quit() }()
            }
        },
'@)
$main = $main.Replace('app.Window.NewWithOptions(', 'window := app.Window.NewWithOptions(')
$main = $main.Replace('return app.Run()', @'
    window.OnWindowEvent(events.Windows.WebViewNavigationCompleted, func(*application.WindowEvent) {
        script, err := os.ReadFile(os.Getenv("PITCH_CHECK_SCRIPT"))
        if err != nil { return }
        config, _ := json.Marshal(map[string]string{"expectedDir": dataDir, "nextDir": os.Getenv("PITCH_CHECK_NEXT_DIR")})
        window.ExecJS("window.__pitchCheck=" + string(config) + ";\n" + string(script))
    })
    return app.Run()
'@)
$sourcePath = Join-Path $runRoot 'diagnostic-main.txt'
[System.IO.File]::WriteAllText($sourcePath, $main, $encoding)
$overlayPath = Join-Path $runRoot 'overlay.json'
$overlay = @{ Replace = @{ (Join-Path $repo 'main.go') = $sourcePath } } | ConvertTo-Json
[System.IO.File]::WriteAllText($overlayPath, $overlay, $encoding)
$binary = Join-Path $runRoot 'pitch.exe'
Push-Location $repo
try {
    & go build -overlay $overlayPath -tags production -ldflags '-H windowsgui' -o $binary .
    if ($LASTEXITCODE -ne 0) { throw 'Diagnostic build failed' }
} finally { Pop-Location }

$null = New-Item -ItemType Directory -Path (Join-Path $runRoot 'config'), (Join-Path $runRoot 'home')
$defaultDir = Join-Path (Join-Path $runRoot 'home') 'Pitch'
$changedDir = Join-Path $runRoot 'changed-data'
$savedEnvironment = @{}
foreach ($key in @('APPDATA', 'USERPROFILE', 'PITCH_CHECK_SCRIPT', 'PITCH_CHECK_NEXT_DIR')) {
    $savedEnvironment[$key] = [Environment]::GetEnvironmentVariable($key)
}
$env:APPDATA = Join-Path $runRoot 'config'
$env:USERPROFILE = Join-Path $runRoot 'home'
$env:PITCH_CHECK_SCRIPT = Join-Path $PSScriptRoot 'check-settings.js'
$app = $null
try {
    foreach ($phase in @('edit', 'restart')) {
        if ($phase -eq 'restart') { Start-Sleep -Seconds 2 }
        $activeDir = if ($phase -eq 'edit') { $defaultDir } else { $changedDir }
        $env:PITCH_CHECK_NEXT_DIR = if ($phase -eq 'edit') { $changedDir } else { '' }
        $app = Start-Process -FilePath $binary -WindowStyle Hidden -PassThru
        $null = $app.Handle
        $resultPath = Join-Path $activeDir 'gui-result.json'
        $deadline = [datetime]::UtcNow.AddSeconds(60)
        while (-not (Test-Path -LiteralPath $resultPath) -and -not $app.HasExited -and [datetime]::UtcNow -lt $deadline) {
            Start-Sleep -Milliseconds 100
        }
        if (-not (Test-Path -LiteralPath $resultPath)) { throw "Missing $phase result; see $runRoot" }
        $result = Get-Content -LiteralPath $resultPath -Raw -Encoding UTF8 | ConvertFrom-Json
        if (-not $result.passed) { throw $result.error }
        if (-not $app.WaitForExit(10000)) { throw 'Diagnostic application did not close' }
        if ($app.ExitCode -ne 0) { throw "Diagnostic application exited with $($app.ExitCode)" }
        $settingsPath = Join-Path (Join-Path $runRoot 'config') 'Pitch/settings.json'
        $settings = Get-Content -LiteralPath $settingsPath -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($settings.language -ne 'en' -or $settings.dataDir -ne $changedDir) { throw 'Saved settings differ from the UI result' }
        if (-not (Test-Path -LiteralPath (Join-Path $activeDir 'pitch.sqlite'))) { throw 'Database missing from active data folder' }
        Write-Output ("Settings UI {0}: passed" -f $phase)
    }
} finally {
    if ($null -ne $app -and -not $app.HasExited) { Stop-Process -Id $app.Id }
    foreach ($key in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($key, $savedEnvironment[$key]) }
}
Write-Output "Reports: $runRoot"
