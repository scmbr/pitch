//go:build windows

package main

import (
	"syscall"

	"github.com/scmbr/pitch/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

const primaryLanguageRussian = 0x19

var getUserDefaultUILanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// systemLanguage returns "ru" for a Russian Windows UI and "en" otherwise.
func systemLanguage() string {
	langID, _, _ := getUserDefaultUILanguage.Call()
	if langID&0x3ff == primaryLanguageRussian {
		return settings.LanguageRU
	}
	return settings.LanguageEN
}

// showErrorBox opens a native message box; it works before any window exists.
func showErrorBox(_ string, title, body string) {
	w32.MessageBox(0, body, title, w32.MB_OK|w32.MB_ICONERROR)
}
