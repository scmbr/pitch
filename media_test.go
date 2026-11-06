package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scmbr/pitch/internal/attempts"
)

func TestMediaHandlerRangeAndPathIsolation(t *testing.T) {
	dir := t.TempDir()
	token := strings.Repeat("a", 32)
	if e := os.Mkdir(filepath.Join(dir, "videos"), 0o700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "videos", token+".webm"), []byte("0123456789"), 0o600); e != nil {
		t.Fatal(e)
	}
	handler := mediaHandler(attempts.NewStore(nil, dir, nil), http.NotFoundHandler())
	r := httptest.NewRequest("GET", "/media/"+token+".webm", nil)
	r.Header.Set("Range", "bytes=2-5")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 206 || w.Body.String() != "2345" || w.Header().Get("Content-Type") != "video/webm" {
		t.Fatalf("range: %d %q", w.Code, w.Body.String())
	}
	for _, path := range []string{"/media/../../settings.json", "/media/invalid.webm", "/media/" + token + ".json"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatalf("exposed %s", path)
		}
	}
}
