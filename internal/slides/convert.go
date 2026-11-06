// Package slides prepares offline presentation snapshots for local rendering.
package slides

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("libreOfficeMissing")

func FindLibreOffice() (string, error) {
	for _, name := range []string{"soffice.com", "soffice", "libreoffice"} {
		if path, e := exec.LookPath(name); e == nil {
			return path, nil
		}
	}
	for _, path := range []string{filepath.Join(os.Getenv("ProgramFiles"), "LibreOffice", "program", "soffice.com"), filepath.Join(os.Getenv("ProgramFiles(x86)"), "LibreOffice", "program", "soffice.com"), "/Applications/LibreOffice.app/Contents/MacOS/soffice"} {
		if f, e := os.Stat(path); e == nil && !f.IsDir() {
			return path, nil
		}
	}
	return "", ErrUnavailable
}

type relationship struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr,omitempty"`
}
type relationships struct {
	XMLName xml.Name       `xml:"http://schemas.openxmlformats.org/package/2006/relationships Relationships"`
	Items   []relationship `xml:"Relationship"`
}

// External OOXML relationships are removed from the conversion-only copy so
// linked images/documents cannot cause LibreOffice to fetch remote resources.
func SanitisePPTX(source, destination string) error {
	r, e := zip.OpenReader(source)
	if e != nil {
		return errors.New("invalidPresentation")
	}
	defer func() { _ = r.Close() }()
	if len(r.File) > 10000 {
		return errors.New("presentationTooLarge")
	}
	var total uint64
	hasSlides := false
	for _, f := range r.File {
		if f.UncompressedSize64 > 512*1024*1024-total {
			return errors.New("presentationTooLarge")
		}
		total += f.UncompressedSize64
		if strings.HasPrefix(f.Name, "ppt/slides/slide") {
			hasSlides = true
		}
		if total > 512*1024*1024 {
			return errors.New("presentationTooLarge")
		}
	}
	if !hasSlides {
		return errors.New("invalidPresentation")
	}
	out, e := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if e != nil {
		return e
	}
	w := zip.NewWriter(out)
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".rels") {
			e = w.Copy(f)
		} else {
			if f.UncompressedSize64 > 8*1024*1024 {
				e = errors.New("presentationTooLarge")
				break
			}
			var reader io.ReadCloser
			reader, e = f.Open()
			if e != nil {
				break
			}
			var rel relationships
			e = xml.NewDecoder(reader).Decode(&rel)
			_ = reader.Close()
			if e != nil {
				break
			}
			kept := rel.Items[:0]
			for _, entry := range rel.Items {
				target, parseErr := url.Parse(entry.Target)
				if parseErr == nil && !target.IsAbs() && !strings.HasPrefix(entry.Target, "//") && !strings.HasPrefix(entry.Target, `\\`) && !strings.EqualFold(entry.TargetMode, "External") {
					kept = append(kept, entry)
				}
			}
			rel.Items = kept
			var data []byte
			data, e = xml.Marshal(rel)
			if e != nil {
				break
			}
			var entry io.Writer
			entry, e = w.Create(f.Name)
			if e == nil {
				_, e = entry.Write(data)
			}
		}
		if e != nil {
			break
		}
	}
	return errors.Join(e, w.Close(), out.Close())
}

func Convert(ctx context.Context, source, destination string) error {
	program, e := FindLibreOffice()
	if e != nil {
		return e
	}
	work, e := os.MkdirTemp(filepath.Dir(destination), "convert-")
	if e != nil {
		return e
	}
	defer func() { _ = os.RemoveAll(work) }()
	input := filepath.Join(work, "slides.pptx")
	if e := SanitisePPTX(source, input); e != nil {
		return e
	}
	profile := filepath.Join(work, "profile", "user")
	if e := os.MkdirAll(profile, 0o700); e != nil {
		return e
	}
	const config = `<?xml version="1.0" encoding="UTF-8"?><oor:items xmlns:oor="http://openoffice.org/2001/registry"><item oor:path="/org.openoffice.Office.Common/Security/Scripting"><prop oor:name="MacroSecurityLevel" oor:op="fuse"><value>3</value></prop></item><item oor:path="/org.openoffice.Office.Common/Misc"><prop oor:name="CollectUsageInformation" oor:op="fuse"><value>false</value></prop></item><item oor:path="/org.openoffice.Office.Jobs/Jobs/UpdateCheck/Arguments"><prop oor:name="AutoCheckEnabled" oor:op="fuse"><value>false</value></prop></item></oor:items>`
	if e := os.WriteFile(filepath.Join(profile, "registrymodifications.xcu"), []byte(config), 0o600); e != nil {
		return e
	}
	profilePath := filepath.ToSlash(filepath.Dir(profile))
	if !strings.HasPrefix(profilePath, "/") {
		profilePath = "/" + profilePath
	}
	uri := (&url.URL{Scheme: "file", Path: profilePath}).String()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, "-env:UserInstallation="+uri, "--headless", "--norestore", "--nodefault", "--nolockcheck", "--nologo", "--convert-to", "pdf:impress_pdf_Export", "--outdir", work, input)
	configureCommand(cmd)
	if e := cmd.Run(); e != nil {
		return errors.New("conversionFailed")
	}
	return os.Rename(filepath.Join(work, "slides.pdf"), destination)
}
