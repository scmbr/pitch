//go:build !windows

package main

import (
	"log"
	"os"
	"strings"

	"github.com/scmbr/pitch/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func systemLanguage() string {
	if strings.HasPrefix(os.Getenv("LANG"), "ru") {
		return settings.LanguageRU
	}
	return settings.LanguageEN
}

// The native dialog needs a running platform event loop. No WebView, settings
// store or database is created, since one of those may have caused the failure.
func showErrorBox(lang, title, body string) {
	app := application.New(application.Options{Name: "Pitch"})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		dialog := app.Dialog.Error().SetTitle(title).SetMessage(body)
		dialog.AddButton(startupCloseLabel(lang)).SetAsDefault().SetAsCancel().OnClick(app.Quit)
		dialog.Show()
	})
	if err := app.Run(); err != nil {
		log.Printf("show startup error: %v; %s: %s", err, title, body)
	}
}

func startupCloseLabel(lang string) string {
	if lang == settings.LanguageRU {
		return "Закрыть"
	}
	return "Close"
}
