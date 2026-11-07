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
param([ValidateSet('settings', 'training', 'projects', 'impromptu')][string]$Scenario = 'settings')
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
if ($Scenario -eq 'projects') {
    & (Join-Path $PSScriptRoot 'make-presentation-fixture.ps1') -Directory $runRoot
    $marker = 'backend.NewProjectService(projectStore, pickPresentationFile)'
    if (-not $main.Contains($marker)) { throw 'Project service overlay marker missing' }
    $main = $main.Replace($marker, 'backend.NewProjectService(projectStore, func(_ string, pattern string) (string,error) { if pattern == "*.pdf" { return os.Getenv("PITCH_CHECK_PDF"),nil }; return os.Getenv("PITCH_CHECK_PPTX"),nil })')
}
if ($Scenario -eq 'impromptu') {
    $topics = "topic,tag`n`"Build a repair cafe, together`",community`nExplain a useful mistake,learning`nPitch a quiet workspace,work`n"
    [IO.File]::WriteAllText((Join-Path $runRoot 'topics.csv'), $topics, $encoding)
    $marker = 'backend.NewTopicService(topicStore, pickPresentationFile)'
    if (-not $main.Contains($marker)) { throw 'Topic service overlay marker missing' }
    $main = $main.Replace($marker, 'backend.NewTopicService(topicStore, func(_ string, _ string) (string,error) { return os.Getenv("PITCH_CHECK_TOPICS"),nil })')
}
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
if ($Scenario -ne 'settings') {
    $webviewPath = Join-Path $repo 'webview2.go'
    $webview = Get-Content -LiteralPath $webviewPath -Raw -Encoding UTF8
    $marker = 'AdditionalBrowserArgs: []string{'
    if (-not $webview.Contains($marker)) { throw 'WebView overlay marker missing' }
    $webview = $webview.Replace($marker, $marker + '"--use-fake-device-for-media-stream", "--use-fake-ui-for-media-stream",')
    $webviewSource = Join-Path $runRoot 'diagnostic-webview.txt'
    [System.IO.File]::WriteAllText($webviewSource, $webview, $encoding)
    $overlay = @{ Replace = @{ (Join-Path $repo 'main.go') = $sourcePath; $webviewPath = $webviewSource } } | ConvertTo-Json
}
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
foreach ($key in @('APPDATA', 'USERPROFILE', 'PITCH_CHECK_SCRIPT', 'PITCH_CHECK_NEXT_DIR', 'PITCH_CHECK_PDF', 'PITCH_CHECK_PPTX', 'PITCH_CHECK_TOPICS')) {
    $savedEnvironment[$key] = [Environment]::GetEnvironmentVariable($key)
}
$env:APPDATA = Join-Path $runRoot 'config'
$env:USERPROFILE = Join-Path $runRoot 'home'
$env:PITCH_CHECK_SCRIPT = Join-Path $PSScriptRoot ('check-' + $Scenario + '.js')
$env:PITCH_CHECK_PDF = Join-Path $runRoot 'talk.pdf'
$env:PITCH_CHECK_PPTX = Join-Path $runRoot 'talk.pptx'
$env:PITCH_CHECK_TOPICS = Join-Path $runRoot 'topics.csv'
$app = $null
try {
    $phases = if ($Scenario -ne 'settings') { @($Scenario) } else { @('edit', 'restart') }
    foreach ($phase in $phases) {
        if ($phase -eq 'restart') { Start-Sleep -Seconds 2 }
        $activeDir = if ($phase -eq 'restart') { $changedDir } else { $defaultDir }
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
        if ($Scenario -eq 'settings') {
            $settingsPath = Join-Path (Join-Path $runRoot 'config') 'Pitch/settings.json'
            $settings = Get-Content -LiteralPath $settingsPath -Raw -Encoding UTF8 | ConvertFrom-Json
            if ($settings.language -ne 'en' -or $settings.dataDir -ne $changedDir) { throw 'Saved settings differ from the UI result' }
        } else {
            $videos = @(Get-ChildItem -LiteralPath (Join-Path $activeDir 'videos') -Filter '*.webm')
            if ($videos.Count -ne $result.expectedVideos) { throw 'Unexpected number of remaining videos' }
            if (@(Get-ChildItem -LiteralPath (Join-Path $activeDir 'thumbs') -Filter '*.jpg').Count -ne $result.expectedVideos) { throw 'Unexpected thumbnail count' }
            if (@(Get-ChildItem -LiteralPath (Join-Path $activeDir 'videos') -Filter '*.partial').Count -ne 0) { throw 'Partial recording leaked' }
        }
        if (-not (Test-Path -LiteralPath (Join-Path $activeDir 'pitch.sqlite'))) { throw 'Database missing from active data folder' }
        Write-Output ("Settings UI {0}: passed" -f $phase)
    }
} finally {
    if ($null -ne $app -and -not $app.HasExited) { Stop-Process -Id $app.Id }
    foreach ($key in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($key, $savedEnvironment[$key]) }
}
Write-Output "Reports: $runRoot"
