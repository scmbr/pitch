package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/scmbr/pitch/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dataDir, err := resolveDataDir()
	if err != nil {
		return fmt.Errorf("cannot prepare the data folder: %w", err)
	}

	db, err := storage.Open(context.Background(), storage.Path(dataDir))
	if err != nil {
		return fmt.Errorf("cannot open the database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("closing the database: %v", err)
		}
	}()

	app := application.New(application.Options{
		Name:        "Pitch",
		Description: "Offline pitch practice trainer",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
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
