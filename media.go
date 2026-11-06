package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/scmbr/pitch/internal/attempts"
)

// This handler serves only local generated media. It has no listening socket or
// HTTP client: Wails dispatches requests from the application's own origin.
func mediaHandler(store *attempts.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/media/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/media/")
		thumbnail := strings.HasSuffix(name, ".jpg")
		suffix := ".webm"
		contentType := "video/webm"
		if thumbnail {
			suffix = ".jpg"
			contentType = "image/jpeg"
		}
		if !strings.HasSuffix(name, suffix) {
			http.NotFound(w, r)
			return
		}
		path, err := store.MediaPath(strings.TrimSuffix(name, suffix), thumbnail)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		file, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()
		stat, err := file.Stat()
		if err != nil || !stat.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, stat.ModTime(), file)
	})
}
