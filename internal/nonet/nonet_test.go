// Package nonet guards the offline promise: code under internal/ must not
// import anything that can open a network connection.
package nonet

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var forbidden = []string{
	"net",
	"net/http",
	"net/rpc",
	"net/smtp",
	"net/textproto",
	"crypto/tls",
	"golang.org/x/net",
}

// allowed are net/* packages that only parse values and never open sockets.
var allowed = []string{"net/url", "net/netip"}

func isForbidden(path string) bool {
	for _, a := range allowed {
		if path == a {
			return false
		}
	}
	for _, f := range forbidden {
		if path == f || strings.HasPrefix(path, f+"/") {
			return true
		}
	}
	return false
}

func TestInternalPackagesDoNotImportNetworkPackages(t *testing.T) {
	root := filepath.Join("..")
	fset := token.NewFileSet()
	checked := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		checked++
		for _, imp := range file.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if isForbidden(importPath) {
				t.Errorf("%s imports %q, which can reach the network", path, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no Go files found under internal/, the test is looking in the wrong place")
	}
}

func TestIsForbidden(t *testing.T) {
	tests := map[string]bool{
		"net":                    true,
		"net/http":               true,
		"net/http/httputil":      true,
		"golang.org/x/net/proxy": true,
		"crypto/tls":             true,
		"net/url":                false,
		"os":                     false,
		"network":                false,
	}
	for path, want := range tests {
		if got := isForbidden(path); got != want {
			t.Errorf("isForbidden(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestMain(m *testing.M) {
	if _, err := os.Stat(filepath.Join("..", "settings")); err != nil {
		panic("run this test from internal/nonet: " + err.Error())
	}
	os.Exit(m.Run())
}
