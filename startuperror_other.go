//go:build !windows

package main

import (
	"log"
	"os"
	"strings"

	"github.com/scmbr/pitch/internal/settings"
)

func systemLanguage() string {
	if strings.HasPrefix(os.Getenv("LANG"), "ru") {
		return settings.LanguageRU
	}
	return settings.LanguageEN
}

// showErrorBox has no native fallback outside Windows yet, so the error only reaches the log.
func showErrorBox(title, body string) {
	log.Printf("%s: %s", title, body)
}
