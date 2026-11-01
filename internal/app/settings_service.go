// Package app holds the methods the frontend can call.
//
// Errors are returned as codes, not texts: the frontend owns all user-facing
// strings so they can be translated in one place.
package app

import (
	"errors"

	"github.com/scmbr/pitch/internal/settings"
)

// Error and warning codes understood by the frontend.
const (
	CodeDataDirEmpty       = "dataDirEmpty"
	CodeDataDirNotAbsolute = "dataDirNotAbsolute"
	CodeDataDirNotWritable = "dataDirNotWritable"
	CodeLanguageUnknown    = "languageUnknown"
	CodeSaveFailed         = "saveFailed"
	CodeSettingsCorrupt    = "settingsCorrupt"
)

// SettingsView is what the settings screen shows.
type SettingsView struct {
	DataDir  string `json:"dataDir"`
	Language string `json:"language"`
	// ActiveDataDir is the folder this run started with. A saved change to
	// DataDir takes effect after a restart.
	ActiveDataDir   string `json:"activeDataDir"`
	RestartRequired bool   `json:"restartRequired"`
	// Warning is a code for a problem that did not stop the app, or empty.
	Warning string `json:"warning"`
}

// SaveResult reports whether settings were saved; ErrorCode is empty on success.
type SaveResult struct {
	OK        bool   `json:"ok"`
	ErrorCode string `json:"errorCode"`
}

// SettingsService exposes reading and saving settings to the frontend.
type SettingsService struct {
	store         *settings.Store
	activeDataDir string
	pickFolder    func(startDir, title string) (string, error)
}

// NewSettingsService creates the service. pickFolder shows a native folder
// chooser and returns "" if the user cancels.
func NewSettingsService(store *settings.Store, activeDataDir string, pickFolder func(startDir, title string) (string, error)) *SettingsService {
	return &SettingsService{store: store, activeDataDir: activeDataDir, pickFolder: pickFolder}
}

// Get returns the saved settings. A corrupt file is reported as a warning and defaults are shown.
func (s *SettingsService) Get() (SettingsView, error) {
	current, err := s.store.Load()
	warning := ""
	switch {
	case errors.Is(err, settings.ErrCorrupt):
		warning = CodeSettingsCorrupt
	case err != nil:
		return SettingsView{}, err
	}
	return SettingsView{
		DataDir:         current.DataDir,
		Language:        current.Language,
		ActiveDataDir:   s.activeDataDir,
		RestartRequired: current.DataDir != s.activeDataDir,
		Warning:         warning,
	}, nil
}

// Save checks that the data folder is usable now, so a bad choice is caught here
// and not at the next start, and then writes the settings.
func (s *SettingsService) Save(dataDir, language string) SaveResult {
	next := settings.Settings{DataDir: dataDir, Language: language}
	if err := next.Validate(); err != nil {
		return SaveResult{ErrorCode: errorCode(err)}
	}
	if err := settings.PrepareDataDir(dataDir); err != nil {
		return SaveResult{ErrorCode: errorCode(err)}
	}
	if err := s.store.Save(next); err != nil {
		return SaveResult{ErrorCode: errorCode(err)}
	}
	return SaveResult{OK: true}
}

// PickDataDir lets the user choose a folder. The title comes from the frontend
// so it is translated. It returns "" if they cancel.
func (s *SettingsService) PickDataDir(startDir, title string) (string, error) {
	return s.pickFolder(startDir, title)
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, settings.ErrDataDirEmpty):
		return CodeDataDirEmpty
	case errors.Is(err, settings.ErrDataDirNotAbsolute):
		return CodeDataDirNotAbsolute
	case errors.Is(err, settings.ErrDataDirNotWritable):
		return CodeDataDirNotWritable
	case errors.Is(err, settings.ErrLanguageUnknown):
		return CodeLanguageUnknown
	default:
		return CodeSaveFailed
	}
}
