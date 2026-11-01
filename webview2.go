package main

import "github.com/wailsapp/wails/v3/pkg/application"

// webviewOptions turns off the background services of the embedded browser.
// WebView2 shares one browser environment, so these apply to every window.
// The flags are Chromium switches and WebView2 may ignore some of them; the
// effect of each one is checked with the firewall measurement in the ADR.
func webviewOptions() application.WindowsOptions {
	return application.WindowsOptions{
		AdditionalBrowserArgs: []string{
			"--disable-background-networking",
			"--disable-component-update",
			"--disable-sync",
			"--no-pings",
			"--disable-domain-reliability",
			"--disable-breakpad",
		},
		DisabledFeatures: []string{
			"OptimizationHints",
			"OptimizationGuideModelDownloading",
			"Translate",
			"MediaRouter",
		},
	}
}
