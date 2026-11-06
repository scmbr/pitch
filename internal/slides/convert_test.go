package slides

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PITCH_OFFICE_HELPER") == "1" {
		out := ""
		profile := ""
		for i, arg := range os.Args {
			if arg == "--outdir" && i+1 < len(os.Args) {
				out = os.Args[i+1]
			}
			if strings.HasPrefix(arg, "-env:UserInstallation=") {
				profile = strings.TrimPrefix(arg, "-env:UserInstallation=")
			}
		}
		u, e := url.Parse(profile)
		if e != nil || u.Scheme != "file" || out == "" {
			os.Exit(21)
		}
		path := u.Path
		if runtime.GOOS == "windows" {
			path = strings.TrimPrefix(path, "/")
		}
		data, e := os.ReadFile(filepath.Join(filepath.FromSlash(path), "user", "registrymodifications.xcu"))
		if e != nil || !strings.Contains(string(data), "MacroSecurityLevel") || !strings.Contains(string(data), "AutoCheckEnabled") {
			os.Exit(22)
		}
		if e := os.WriteFile(filepath.Join(out, "slides.pdf"), []byte("%PDF-1.7\nconverted"), 0o600); e != nil {
			os.Exit(23)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestConvertRunsIsolatedProcessAndCollectsPDF(t *testing.T) {
	dir := t.TempDir()
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	name := "soffice"
	if runtime.GOOS == "windows" {
		name = "soffice.com"
	}
	in, e := os.Open(executable)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = in.Close() }()
	out, e := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if e != nil {
		t.Fatal(e)
	}
	_, e = io.Copy(out, in)
	closeErr := out.Close()
	if e != nil || closeErr != nil {
		t.Fatalf("helper: %v %v", e, closeErr)
	}
	t.Setenv("PATH", dir)
	t.Setenv("PITCH_OFFICE_HELPER", "1")
	source := filepath.Join(dir, "talk.pptx")
	file, e := os.Create(source)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(file)
	w, e := z.Create("ppt/slides/slide1.xml")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := w.Write([]byte("<slide/>")); e != nil {
		t.Fatal(e)
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	if e := file.Close(); e != nil {
		t.Fatal(e)
	}
	pdf := filepath.Join(dir, "document.pdf")
	if e := Convert(context.Background(), source, pdf); e != nil {
		t.Fatal(e)
	}
	if data, e := os.ReadFile(pdf); e != nil || !strings.HasPrefix(string(data), "%PDF-") {
		t.Fatal("conversion result missing")
	}
	files, e := os.ReadDir(dir)
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "convert-") {
			t.Fatal("conversion profile leaked")
		}
	}
}

func TestSanitiseExternalRelationships(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "talk.pptx")
	out, e := os.Create(source)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(out)
	for name, content := range map[string]string{
		"ppt/slides/slide1.xml":            "<slide/>",
		"ppt/slides/_rels/slide1.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="keep" Type="image" Target="../media/image.png"/><Relationship Id="web" Type="image" Target="https://example.invalid/image.png" TargetMode="External"/><Relationship Id="unc" Type="image" Target="//server/share"/><Relationship Id="malformed" Type="image" Target="https://example.invalid/"/></Relationships>`,
	} {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := w.Write([]byte(content)); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	if e := out.Close(); e != nil {
		t.Fatal(e)
	}
	destination := filepath.Join(dir, "sanitised.pptx")
	if e := SanitisePPTX(source, destination); e != nil {
		t.Fatal(e)
	}
	r, e := zip.OpenReader(destination)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".rels") {
			continue
		}
		stream, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		var rel relationships
		e = xml.NewDecoder(stream).Decode(&rel)
		_ = stream.Close()
		if e != nil {
			t.Fatal(e)
		}
		if len(rel.Items) != 1 || rel.Items[0].ID != "keep" {
			t.Fatalf("external relationship retained: %+v", rel)
		}
	}
}
