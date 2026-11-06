package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordingLifecycle(t *testing.T) {
	dir := t.TempDir()
	s := NewRecordingService(dir)
	id, err := s.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Begin(); err == nil {
		t.Fatal("allowed concurrent recording")
	}
	if err := s.Append("../escape", "YQ=="); err == nil {
		t.Fatal("accepted wrong token")
	}
	if err := s.Append(id, strings.Repeat("A", 512*1024+1)); err == nil {
		t.Fatal("accepted oversized chunk")
	}
	for _, text := range []string{"first", "second"} {
		if err := s.Append(id, base64.StdEncoding.EncodeToString([]byte(text))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Finish(id); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "videos", id+".webm"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "firstsecond" {
		t.Fatalf("chunks: %q", got)
	}
	if err := s.Abort(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "videos", id+".webm")); err != nil {
		t.Fatal("abort removed finalized recording")
	}
}

func TestRecordingAbortAndShutdown(t *testing.T) {
	s := NewRecordingService(t.TempDir())
	id, err := s.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Finish(id); err == nil {
		t.Fatal("accepted empty recording")
	}
	if err := s.Abort(id); err != nil {
		t.Fatal(err)
	}
	id, err = s.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.dir, id+".partial")); !os.IsNotExist(err) {
		t.Fatal("partial survived shutdown")
	}
}

func TestRecordingUnavailableFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "videos"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewRecordingService(dir)
	if _, err := s.Begin(); err == nil {
		t.Fatal("opened recording in a regular file")
	}
	got, err := os.ReadFile(filepath.Join(dir, "videos"))
	if err != nil || string(got) != "keep" {
		t.Fatal("changed existing file")
	}
}
