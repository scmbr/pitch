// Package settings stores user preferences in a small JSON file.
//
// The file lives in a fixed place (the OS config dir) because it holds the
// location of the data folder, so it cannot live inside that folder itself.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirName    = "Pitch"
	fileName      = "settings.json"
	probeName     = ".pitch-write-test"
	defaultFolder = "Pitch"
)

var (
	ErrDataDirEmpty       = errors.New("settings: data folder is empty")
	ErrDataDirNotAbsolute = errors.New("settings: data folder must be an absolute path")
	ErrDataDirNotWritable = errors.New("settings: data folder is not writable")
	ErrCorrupt            = errors.New("settings: settings file is corrupt")
	ErrLanguageUnknown    = errors.New("settings: unknown language")
)

// Interface languages. LanguageAuto follows the system language.
const (
	LanguageAuto = "auto"
	LanguageRU   = "ru"
	LanguageEN   = "en"
)

// Settings is everything the user can configure. New fields must keep a
// usable zero value so that older files still load.
type Settings struct {
	DataDir  string `json:"dataDir"`
	Language string `json:"language"`
}

// Validate checks the values without touching the disk.
func (s Settings) Validate() error {
	if s.DataDir == "" {
		return ErrDataDirEmpty
	}
	if !filepath.IsAbs(s.DataDir) {
		return fmt.Errorf("%w: %q", ErrDataDirNotAbsolute, s.DataDir)
	}
	switch s.Language {
	case LanguageAuto, LanguageRU, LanguageEN:
	default:
		return fmt.Errorf("%w: %q", ErrLanguageUnknown, s.Language)
	}
	return nil
}

// Store reads and writes one settings file.
type Store struct {
	path     string
	defaults Settings
}

// NewStore returns a store for the file at path. A missing file yields defaults.
func NewStore(path string, defaults Settings) *Store {
	if defaults.Language == "" {
		defaults.Language = LanguageAuto
	}
	return &Store{path: path, defaults: defaults}
}

// Path returns the settings file location.
func (s *Store) Path() string { return s.path }

// Load returns the saved settings, or the defaults if there is no file yet.
// On a corrupt file it returns the defaults together with ErrCorrupt and
// leaves the file untouched so the user can still recover it.
func (s *Store) Load() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.defaults, nil
	}
	if err != nil {
		return s.defaults, fmt.Errorf("settings: read %s: %w", s.path, err)
	}

	loaded := s.defaults
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s.defaults, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	if loaded.DataDir == "" {
		loaded.DataDir = s.defaults.DataDir
	}
	if loaded.Language == "" {
		loaded.Language = LanguageAuto
	}
	return loaded, nil
}

// Save validates and writes the settings. The file is replaced atomically so
// a crash cannot leave a half-written file.
func (s *Store) Save(v Settings) error {
	if v.Language == "" {
		v.Language = LanguageAuto
	}
	if err := v.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("settings: encode: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("settings: create %s: %w", filepath.Dir(s.path), err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), fileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("settings: write %s: %w", s.path, err)
	}
	tmpName := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("settings: write %s: %w", s.path, err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("settings: replace %s: %w", s.path, err)
	}
	return nil
}

// ConfigPath returns the fixed settings file location: <user config dir>/Pitch/settings.json.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("settings: locate config dir: %w", err)
	}
	return filepath.Join(dir, appDirName, fileName), nil
}

// DefaultDataDir returns <home>/Pitch. It is deliberately outside the
// roaming profile, where videos would be copied between machines.
func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("settings: locate home dir: %w", err)
	}
	return filepath.Join(home, defaultFolder), nil
}

// PrepareDataDir creates the folder if needed and checks that files can be written to it.
func PrepareDataDir(dir string) error {
	if err := (Settings{DataDir: dir, Language: LanguageAuto}).Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("%w: %v", ErrDataDirNotWritable, err)
	}
	// A fixed name could overwrite a user's file or collide with another
	// running instance. Only remove the unique file created by this check.
	probe, err := os.CreateTemp(dir, probeName+".*")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDataDirNotWritable, err)
	}
	if err := errors.Join(probe.Close(), os.Remove(probe.Name())); err != nil {
		return fmt.Errorf("%w: %v", ErrDataDirNotWritable, err)
	}
	return nil
}
