package main

import (
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMacBundlesDeclareMediaPermissionsAndExecutable(t *testing.T) {
	for _, path := range []string{"build/darwin/Info.plist", "build/darwin/Info.dev.plist"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(strings.NewReader(string(data)))
		values := map[string]string{}
		key := ""
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			if start.Name.Local == "key" {
				if err := decoder.DecodeElement(&key, &start); err != nil {
					t.Fatal(err)
				}
			}
			if start.Name.Local == "string" && key != "" {
				var value string
				if err := decoder.DecodeElement(&value, &start); err != nil {
					t.Fatal(err)
				}
				values[key] = value
				key = ""
			}
		}
		if values["CFBundleExecutable"] != "pitch" {
			t.Fatalf("%s points at wrong executable", path)
		}
		for _, key := range []string{"NSCameraUsageDescription", "NSMicrophoneUsageDescription"} {
			if values[key] == "" {
				t.Fatalf("%s missing %s", path, key)
			}
		}
	}
	for _, language := range []string{"en", "ru"} {
		data, err := os.ReadFile("build/darwin/" + language + ".lproj/InfoPlist.strings")
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"NSCameraUsageDescription", "NSMicrophoneUsageDescription"} {
			if !strings.Contains(string(data), key) {
				t.Fatalf("missing %s translation for %s", key, language)
			}
		}
	}
}
