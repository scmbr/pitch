package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/scmbr/pitch/internal/settings"
	"github.com/scmbr/pitch/internal/storage"
)

var (
	errDataFolder = errors.New("data folder")
	errDatabase   = errors.New("database")
)

// startupMessage describes a failure that happens before the window exists.
// The frontend cannot show it, so these few strings live here in both languages.
func startupMessage(lang string, err error, settingsPath string) (title, body string) {
	ru := lang == settings.LanguageRU

	switch {
	case errors.Is(err, storage.ErrDatabaseNewer):
		if ru {
			return "Pitch не может открыть данные",
				"Данные созданы более новой версией Pitch. Установите свежую версию приложения."
		}
		return "Pitch cannot open your data",
			"Your data was created by a newer version of Pitch. Please install the latest version."
	case errors.Is(err, errDatabase):
		if ru {
			return "Не удалось открыть базу данных",
				fmt.Sprintf("Проверьте, что диск подключён и папка доступна для записи.\n\n%v", err)
		}
		return "Cannot open the database",
			fmt.Sprintf("Check that the drive is connected and the folder is writable.\n\n%v", err)
	case errors.Is(err, errDataFolder):
		if ru {
			return "Не удалось подготовить папку данных",
				fmt.Sprintf("Проверьте, что диск подключён и в папку можно записывать. "+
					"Чтобы вернуть папку по умолчанию, удалите файл настроек:\n%s\n\n%v", settingsPath, err)
		}
		return "Cannot prepare the data folder",
			fmt.Sprintf("Check that the drive is connected and the folder is writable. "+
				"To go back to the default folder, delete the settings file:\n%s\n\n%v", settingsPath, err)
	default:
		if ru {
			return "Не удалось запустить Pitch", err.Error()
		}
		return "Pitch could not start", err.Error()
	}
}

// reportStartupError shows the failure to the user and also writes it to the log.
func reportStartupError(err error) {
	log.Printf("startup failed: %v", err)

	settingsPath, pathErr := settings.ConfigPath()
	if pathErr != nil {
		settingsPath = "settings.json"
	}
	title, body := startupMessage(systemLanguage(), err, settingsPath)
	showErrorBox(title, body)
}
