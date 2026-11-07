package settings

import (
	"errors"
	"regexp"
	"strings"
)

var ErrPreferences = errors.New("settings: invalid preferences")

type Preferences struct {
	Preparation  int               `json:"preparation"`
	Speech       int               `json:"speech"`
	CameraID     string            `json:"cameraId"`
	MicrophoneID string            `json:"microphoneId"`
	Quality      string            `json:"quality"`
	Theme        string            `json:"theme"`
	TopicWindow  int               `json:"topicWindow"`
	Shortcuts    map[string]string `json:"shortcuts"`
}

func DefaultPreferences() Preferences {
	return Preferences{Preparation: 30, Speech: 120, Quality: "720p", Theme: "auto", TopicWindow: 20, Shortcuts: map[string]string{
		"toggle": "Space", "skip": "Enter", "abort": "Escape", "topic": "KeyR", "previous": "ArrowLeft", "next": "ArrowRight", "fullscreen": "KeyF", "history": "Ctrl+KeyH",
	}}
}

var shortcutKey = regexp.MustCompile(`^(Key[A-Z]|Digit[0-9]|Space|Enter|Escape|ArrowLeft|ArrowRight|ArrowUp|ArrowDown|F([1-9]|1[0-2]))$`)

func (p Preferences) Validate() error {
	if p.Preparation < 0 || p.Preparation > 3600 || p.Speech < 1 || p.Speech > 86400 || p.TopicWindow < 0 || p.TopicWindow > 1000 || len(p.CameraID) > 4096 || len(p.MicrophoneID) > 4096 {
		return ErrPreferences
	}
	if p.Quality != "720p" && p.Quality != "480p" {
		return ErrPreferences
	}
	if p.Theme != "auto" && p.Theme != "light" && p.Theme != "dark" {
		return ErrPreferences
	}
	defaults := DefaultPreferences().Shortcuts
	if len(p.Shortcuts) != len(defaults) {
		return ErrPreferences
	}
	used := map[string]bool{}
	for action, key := range p.Shortcuts {
		if _, ok := defaults[action]; !ok {
			return ErrPreferences
		}
		if key == "" {
			continue
		}
		if used[key] {
			return ErrPreferences
		}
		used[key] = true
		parts := strings.Split(key, "+")
		if !shortcutKey.MatchString(parts[len(parts)-1]) {
			return ErrPreferences
		}
		last := -1
		for _, modifier := range parts[:len(parts)-1] {
			order := -1
			for index, name := range []string{"Ctrl", "Alt", "Shift", "Meta"} {
				if modifier == name {
					order = index
				}
			}
			if order <= last {
				return ErrPreferences
			}
			last = order
		}
	}
	return nil
}

func (s Settings) EffectivePreferences() Preferences {
	if s.Preferences == nil || s.Preferences.Validate() != nil {
		return DefaultPreferences()
	}
	p := *s.Preferences
	p.Shortcuts = make(map[string]string, len(s.Preferences.Shortcuts))
	for k, v := range s.Preferences.Shortcuts {
		p.Shortcuts[k] = v
	}
	return p
}
