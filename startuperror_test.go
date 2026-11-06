package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/scmbr/pitch/internal/settings"
	"github.com/scmbr/pitch/internal/storage"
)

func TestStartupLanguage(t *testing.T) {
	for _, test := range []struct{ preference, system, want string }{
		{settings.LanguageRU, settings.LanguageEN, settings.LanguageRU},
		{settings.LanguageEN, settings.LanguageRU, settings.LanguageEN},
		{settings.LanguageAuto, settings.LanguageRU, settings.LanguageRU},
		{"", settings.LanguageEN, settings.LanguageEN},
		{"unknown", settings.LanguageRU, settings.LanguageRU},
	} {
		if got := startupLanguage(test.preference, test.system); got != test.want {
			t.Errorf("startupLanguage(%q, %q) = %q, want %q", test.preference, test.system, got, test.want)
		}
	}
}

func TestStartupMessage(t *testing.T) {
	const settingsPath = `C:\cfg\settings.json`
	cause := errors.New("disk offline")

	tests := []struct {
		name         string
		err          error
		ruTitle      string
		enTitle      string
		bodyContains []string
	}{
		{
			name:         "data folder",
			err:          fmt.Errorf("%w: %w", errDataFolder, cause),
			ruTitle:      "Не удалось подготовить папку данных",
			enTitle:      "Cannot prepare the data folder",
			bodyContains: []string{settingsPath, "disk offline"},
		},
		{
			name:         "database",
			err:          fmt.Errorf("%w: %w", errDatabase, cause),
			ruTitle:      "Не удалось открыть базу данных",
			enTitle:      "Cannot open the database",
			bodyContains: []string{"disk offline"},
		},
		{
			name:    "database from a newer version",
			err:     fmt.Errorf("%w: %w", errDatabase, storage.ErrDatabaseNewer),
			ruTitle: "Pitch не может открыть данные",
			enTitle: "Pitch cannot open your data",
		},
		{
			name:         "anything else",
			err:          cause,
			ruTitle:      "Не удалось запустить Pitch",
			enTitle:      "Pitch could not start",
			bodyContains: []string{"disk offline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruTitle, ruBody := startupMessage(settings.LanguageRU, tt.err, settingsPath)
			enTitle, enBody := startupMessage(settings.LanguageEN, tt.err, settingsPath)

			if ruTitle != tt.ruTitle || enTitle != tt.enTitle {
				t.Errorf("titles = %q / %q, want %q / %q", ruTitle, enTitle, tt.ruTitle, tt.enTitle)
			}
			for _, want := range tt.bodyContains {
				if !strings.Contains(ruBody, want) || !strings.Contains(enBody, want) {
					t.Errorf("body lacks %q:\nru: %s\nen: %s", want, ruBody, enBody)
				}
			}
		})
	}
}
