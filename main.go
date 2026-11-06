package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	backend "github.com/scmbr/pitch/internal/app"
	"github.com/scmbr/pitch/internal/attempts"
	"github.com/scmbr/pitch/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		reportStartupError(err)
		os.Exit(1)
	}
}

func run() error {
	store, err := newSettingsStore()
	if err != nil {
		return fmt.Errorf("%w: %w", errDataFolder, err)
	}
	dataDir, err := resolveDataDir(store)
	if err != nil {
		return fmt.Errorf("%w: %w", errDataFolder, err)
	}

	db, err := storage.Open(context.Background(), storage.Path(dataDir))
	if err != nil {
		return fmt.Errorf("%w: %w", errDatabase, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("closing the database: %v", err)
		}
	}()

	attemptStore := attempts.NewStore(db, dataDir, store)
	if err := attemptStore.Recover(); err != nil {
		log.Printf("recovering attempts: %v", err)
	}
	app := application.New(application.Options{
		Name:        "Pitch",
		Description: "Offline pitch practice trainer",
		Services: []application.Service{
			application.NewService(backend.NewSettingsService(store, dataDir, pickFolder)),
			application.NewService(backend.NewRecordingService(dataDir, attemptStore.Add)),
			application.NewService(backend.NewAttemptService(attemptStore, store)),
		},
		Assets: application.AssetOptions{
			Handler: mediaHandler(attemptStore, application.AssetFileServerFS(assets)),
		},
		Windows: webviewOptions(dataDir),
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "Pitch",
		Width:       1000,
		Height:      700,
		URL:         "/",
		Permissions: windowPermissions(),
		Windows: application.WindowsWindow{
			Permissions: windowsPermissions(),
		},
	})

	return app.Run()
}

// pickFolder shows the system folder chooser. It returns "" if the user cancels.
func pickFolder(startDir, title string) (string, error) {
	return application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle(title).
		SetDirectory(startDir).
		PromptForSingleSelection()
}
