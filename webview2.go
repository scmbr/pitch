package main

import (
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// windowPermissions grants only what the training screen needs. Without an
// explicit policy WebView2 would decide by itself, so everything else is denied.
func windowPermissions() map[application.PermissionType]application.Permission {
	return map[application.PermissionType]application.Permission{
		application.PermissionCamera:        application.PermissionAllow,
		application.PermissionMicrophone:    application.PermissionAllow,
		application.PermissionGeolocation:   application.PermissionDeny,
		application.PermissionNotifications: application.PermissionDeny,
		application.PermissionClipboardRead: application.PermissionDeny,
	}
}

// windowsPermissions covers the WebView2 kind that has no cross-platform equivalent.
func windowsPermissions() map[application.CoreWebView2PermissionKind]application.CoreWebView2PermissionState {
	return map[application.CoreWebView2PermissionKind]application.CoreWebView2PermissionState{
		application.CoreWebView2PermissionKindOtherSensors: application.CoreWebView2PermissionStateDeny,
	}
}

// webviewOptions turns off the background services of the embedded browser.
// WebView2 shares one browser environment, so these apply to every window.
// The flags are Chromium switches and WebView2 may ignore some of them; the
// effect of each one is checked with the firewall measurement in the ADR.
func webviewOptions(dataDir string) application.WindowsOptions {
	return application.WindowsOptions{
		// Keeping the browser profile inside the data folder makes "delete
		// everything" a single action instead of leaving cache in %APPDATA%.
		WebviewUserDataPath: filepath.Join(dataDir, "webview"),
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
