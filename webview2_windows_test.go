//go:build windows

package main

import (
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestBrowserArgumentsKeepResolverRuleIntact(t *testing.T) {
	// Wails joins these strings before passing them to WebView2. A resolver
	// rule with unquoted spaces becomes "MAP" plus stray positional arguments.
	options := webviewOptions(t.TempDir())
	command, err := syscall.UTF16PtrFromString("webview.exe " + strings.Join(options.AdditionalBrowserArgs, " "))
	if err != nil {
		t.Fatal(err)
	}
	var count int32
	args, err := syscall.CommandLineToArgv(command, &count)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := syscall.LocalFree(syscall.Handle(unsafe.Pointer(args))); err != nil {
			t.Errorf("free parsed command line: %v", err)
		}
	}()
	resolverFound := false
	for _, arg := range args[1:count] {
		value := syscall.UTF16ToString(arg[:])
		if !strings.HasPrefix(value, "--") {
			t.Errorf("browser rule escaped its argument: %q", value)
		}
		if rule, ok := strings.CutPrefix(value, "--host-resolver-rules="); ok {
			resolverFound = true
			if !strings.HasPrefix(rule, "MAP * ^NOTFOUND,") || !strings.Contains(rule, "EXCLUDE wails.localhost") {
				t.Errorf("external DNS restriction or Wails exception was lost: %q", rule)
			}
		}
	}
	if !resolverFound {
		t.Fatal("external DNS resolver restriction is missing")
	}
}
