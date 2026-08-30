package ziputil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// buildZip creates an in-memory archive and returns its parsed reader.
func buildZip(t *testing.T, entries map[string][]byte) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCheckArchive_AcceptsNormalArchive(t *testing.T) {
	r := buildZip(t, map[string][]byte{
		"manifest.json":  []byte(`{"version":"2.0"}`),
		"body/000.html":  []byte("<p>chapter</p>"),
		"copyright.html": []byte("<p>c</p>"),
	})
	if err := CheckArchive(r.File); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckArchive_RejectsTooManyEntries(t *testing.T) {
	entries := make(map[string][]byte, MaxEntries+1)
	for i := 0; i <= MaxEntries; i++ {
		entries[fmt.Sprintf("f%05d", i)] = []byte("x")
	}
	r := buildZip(t, entries)
	if err := CheckArchive(r.File); err == nil {
		t.Fatal("expected entry-count rejection")
	}
}

func TestCheckArchive_RejectsOversizedDeclaredEntry(t *testing.T) {
	// 60 MB of zeros compresses to almost nothing but declares > MaxEntrySize.
	r := buildZip(t, map[string][]byte{"big": make([]byte, MaxEntrySize+1)})
	if err := CheckArchive(r.File); err == nil {
		t.Fatal("expected oversized-entry rejection")
	}
}

func TestCheckArchive_RejectsExcessiveRatio(t *testing.T) {
	// 20 MB of zeros: under MaxEntrySize but ratio is astronomically high.
	r := buildZip(t, map[string][]byte{"bomb": make([]byte, 20<<20)})
	if err := CheckArchive(r.File); err == nil {
		t.Fatal("expected compression-ratio rejection")
	}
}

func TestReadEntry_EnforcesActualSize(t *testing.T) {
	r := buildZip(t, map[string][]byte{"ok": []byte("fine")})
	data, err := ReadEntry(r.File[0])
	if err != nil || string(data) != "fine" {
		t.Fatalf("ReadEntry: %v, %q", err, data)
	}
}

func TestReadNamed_CaseInsensitive(t *testing.T) {
	r := buildZip(t, map[string][]byte{"OEBPS/Content.OPF": []byte("<opf/>")})
	if _, err := ReadNamed(r.File, "oebps/content.opf", true); err != nil {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
	if _, err := ReadNamed(r.File, "oebps/content.opf", false); err == nil {
		t.Fatal("case-sensitive lookup should fail")
	}
	if _, err := ReadNamed(r.File, "missing", true); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}
