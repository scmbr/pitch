package main

import (
	"errors"
	"log"

	"github.com/scmbr/pitch/internal/settings"
)

// resolveDataDir returns the data folder chosen in the saved settings, created
// and checked for write access. A changed folder takes effect on the next start
// because the browser profile and the database are opened from it at startup.
func resolveDataDir() (string, error) {
	configPath, err := settings.ConfigPath()
	if err != nil {
		return "", err
	}
	defaultDir, err := settings.DefaultDataDir()
	if err != nil {
		return "", err
	}

	current, err := settings.NewStore(configPath, settings.Settings{DataDir: defaultDir}).Load()
	if errors.Is(err, settings.ErrCorrupt) {
		log.Printf("settings file is corrupt, using defaults: %v", err)
	} else if err != nil {
		return "", err
	}

	if err := settings.PrepareDataDir(current.DataDir); err != nil {
		return "", err
	}
	return current.DataDir, nil
}
