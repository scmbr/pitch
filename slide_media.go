package main

import (
	"github.com/scmbr/pitch/internal/projects"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func slideHandler(store *projects.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/slides/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/slides/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		id, e := strconv.ParseInt(parts[0], 10, 64)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		page := 0
		mime := "application/pdf"
		if parts[1] != "document.pdf" {
			if !strings.HasSuffix(parts[1], ".png") {
				http.NotFound(w, r)
				return
			}
			page, e = strconv.Atoi(strings.TrimSuffix(parts[1], ".png"))
			if e != nil || page < 1 {
				http.NotFound(w, r)
				return
			}
			mime = "image/png"
		}
		path, e := store.MediaPath(id, page)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		file, e := os.Open(path)
		if e != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()
		stat, e := file.Stat()
		if e != nil || !stat.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, parts[1], stat.ModTime(), file)
	})
}
