package settings

import (
	"reflect"
	"sync"
	"testing"
)

func TestPreferencesValidation(t *testing.T) {
	if err := DefaultPreferences().Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Preferences){
		func(p *Preferences) { p.Speech = 0 },
		func(p *Preferences) { p.Preparation = -1 },
		func(p *Preferences) { p.Quality = "4k" },
		func(p *Preferences) { p.Theme = "invalid" },
		func(p *Preferences) { p.TopicWindow = 1001 },
		func(p *Preferences) { p.Shortcuts["topic"] = "Space" },
		func(p *Preferences) { p.Shortcuts["topic"] = "Ctrl+Ctrl+KeyR" },
		func(p *Preferences) { p.Shortcuts["topic"] = "Shift+Ctrl+KeyR" },
		func(p *Preferences) { p.Shortcuts["topic"] = "Tab" },
	} {
		p := DefaultPreferences()
		change(&p)
		if p.Validate() == nil {
			t.Fatalf("accepted invalid preferences: %+v", p)
		}
	}
}

func TestPreferencesRoundTripAndLegacyDefaults(t *testing.T) {
	s, _ := newTestStore(t)
	old, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(old.EffectivePreferences(), DefaultPreferences()) {
		t.Fatal("legacy defaults")
	}
	p := DefaultPreferences()
	p.Preparation = 0
	p.Shortcuts["toggle"] = "Ctrl+KeyP"
	if err := s.Update(func(v *Settings) error { v.Preferences = &p; return nil }); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || !reflect.DeepEqual(got.EffectivePreferences(), p) {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	copy := got.EffectivePreferences()
	copy.Shortcuts["toggle"] = ""
	if got.Preferences.Shortcuts["toggle"] == "" {
		t.Fatal("returned mutable shared preferences")
	}
}

func TestConcurrentUpdatesKeepAllChanges(t *testing.T) {
	s, _ := newTestStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Update(func(v *Settings) error { v.StorageLimitMB++; return nil }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := s.Load()
	if err != nil || got.StorageLimitMB != 20 {
		t.Fatalf("lost update: %+v %v", got, err)
	}
}
