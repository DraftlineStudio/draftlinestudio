package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func TestInstalledNpmPackageVersion(t *testing.T) {
	prefix := t.TempDir()
	parts := []string{prefix}
	if goruntime.GOOS != "windows" {
		parts = append(parts, "lib")
	}
	parts = append(parts, "node_modules", "@openai", "codex")
	pkgDir := filepath.Join(parts...)
	if err := os.MkdirAll(pkgDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"version":"0.151.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := installedNpmPackageVersion(prefix, "@openai", "codex"); got != "0.151.0" {
		t.Fatalf("installed version = %q", got)
	}
	if got := installedNpmPackageVersion(prefix, "missing"); got != "" {
		t.Fatalf("missing package returned version %q", got)
	}
}

func TestSecurePath(t *testing.T) {
	dest := t.TempDir()
	cases := []struct {
		rel string
		ok  bool
	}{
		{"bin/node", true},
		{"lib/node_modules/npm/README.md", true},
		{"../evil.txt", false},
		{"..", false},
		{"a/../../evil.txt", false},
		{"/etc/passwd", false},
	}
	if goruntime.GOOS == "windows" {
		cases = append(cases,
			struct {
				rel string
				ok  bool
			}{`C:\evil.txt`, false},
			struct {
				rel string
				ok  bool
			}{`..\evil.txt`, false},
		)
	}
	for _, c := range cases {
		_, err := securePath(dest, c.rel)
		if c.ok && err != nil {
			t.Errorf("securePath(%q) unexpected error: %v", c.rel, err)
		}
		if !c.ok && err == nil {
			t.Errorf("securePath(%q) should have been rejected", c.rel)
		}
	}
}

func TestMaskMode(t *testing.T) {
	if m := maskMode(0o4755); m&os.ModeSetuid != 0 {
		t.Fatal("setuid must be stripped")
	}
	if m := maskMode(0o755); m != 0o755 {
		t.Fatalf("plain mode altered: %o", m)
	}
	if m := maskMode(0); m&0o400 == 0 {
		t.Fatal("mode must be readable")
	}
}

// buildTestZip builds a zip whose entries all sit under a fake top-level dir
// (mirroring Node's layout) plus any raw-named extras.
func buildTestZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractNodeZip_RejectsTraversal(t *testing.T) {
	dest := t.TempDir()
	data := buildTestZip(t, map[string]string{
		"node-vX-win-x64/../../evil.txt": "pwned",
	})
	if err := extractNodeZip(data, dest); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evil.txt")); err == nil {
		t.Fatal("traversal file was written outside dest")
	}
}

func TestExtractNodeZip_ExtractsNormalEntries(t *testing.T) {
	dest := t.TempDir()
	data := buildTestZip(t, map[string]string{
		"node-vX-win-x64/node.exe":          "binary",
		"node-vX-win-x64/npm.cmd":           "script",
		"node-vX-win-x64/node_modules/a.js": "js",
	})
	if err := extractNodeZip(data, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "node.exe"))
	if err != nil || string(got) != "binary" {
		t.Fatalf("node.exe: %v %q", err, got)
	}
}

type tarEntry struct {
	name     string
	typeflag byte
	content  string
	linkname string
	mode     int64
}

func buildTestTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Mode:     mode,
			Size:     int64(len(e.content)),
			Linkname: e.linkname,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractNodeTarGz_RejectsEscapingSymlink(t *testing.T) {
	dest := t.TempDir()
	data := buildTestTarGz(t, []tarEntry{
		{name: "node-vX/bin/evil", typeflag: tar.TypeSymlink, linkname: "../../../../etc"},
	})
	if err := extractNodeTarGz(data, dest); err == nil {
		t.Fatal("expected escaping symlink rejection")
	}
}

func TestExtractNodeTarGz_RejectsAbsoluteSymlink(t *testing.T) {
	dest := t.TempDir()
	data := buildTestTarGz(t, []tarEntry{
		{name: "node-vX/bin/evil", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
	})
	if err := extractNodeTarGz(data, dest); err == nil {
		t.Fatal("expected absolute symlink rejection")
	}
}

func TestExtractNodeTarGz_AcceptsInDestRelativeSymlink(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	dest := t.TempDir()
	data := buildTestTarGz(t, []tarEntry{
		{name: "node-vX/lib/node_modules/npm/bin/npm-cli.js", typeflag: tar.TypeReg, content: "js"},
		{name: "node-vX/bin/npm", typeflag: tar.TypeSymlink, linkname: "../lib/node_modules/npm/bin/npm-cli.js"},
	})
	if err := extractNodeTarGz(data, dest); err != nil {
		t.Fatalf("in-dest relative symlink must be accepted (Node needs it): %v", err)
	}
}

func TestExtractNodeTarGz_RejectsTraversalAndHardlink(t *testing.T) {
	dest := t.TempDir()
	if err := extractNodeTarGz(buildTestTarGz(t, []tarEntry{
		{name: "node-vX/../../evil", typeflag: tar.TypeReg, content: "pwned"},
	}), dest); err == nil {
		t.Fatal("expected traversal rejection")
	}
	if err := extractNodeTarGz(buildTestTarGz(t, []tarEntry{
		{name: "node-vX/link", typeflag: tar.TypeLink, linkname: "somewhere"},
	}), dest); err == nil {
		t.Fatal("expected hardlink rejection")
	}
}

func TestExtractNodeTarGz_MasksModeBits(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("unix permission bits not meaningful on Windows")
	}
	dest := t.TempDir()
	data := buildTestTarGz(t, []tarEntry{
		{name: "node-vX/bin/suid", typeflag: tar.TypeReg, content: "x", mode: 0o4755},
	})
	if err := extractNodeTarGz(data, dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "bin", "suid"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSetuid != 0 {
		t.Fatal("setuid bit survived extraction")
	}
}
