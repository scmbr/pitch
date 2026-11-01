package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataDir, err := resolveDataDir()
	if err != nil {
		log.Fatalf("cannot prepare the data folder: %v", err)
	}

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

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
