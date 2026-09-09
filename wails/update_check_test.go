package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseReleaseTag(t *testing.T) {
	cases := []struct {
		tag     string
		version string
		label   string
		ok      bool
	}{
		{"v0.17.02563", "0.17.02563", "0.17.02563", true},
		{"0.17.02563", "0.17.02563", "0.17.02563", true},
		{"v0.18.02600-beta", "0.18.02600", "0.18.02600-beta", true},
		{"v1.0.0-rc.1", "1.0.0", "1.0.0-rc.1", true},
		{"nightly", "", "", false},
		{"v1.2", "", "", false},
	}
	for _, c := range cases {
		version, label, ok := parseReleaseTag(c.tag)
		if version != c.version || label != c.label || ok != c.ok {
			t.Fatalf("parseReleaseTag(%q) = %q, %q, %v", c.tag, version, label, ok)
		}
	}
}

func TestVersionNewer(t *testing.T) {
	if !versionNewer("0.17.02564", "0.17.02563") {
		t.Fatal("higher build must be newer")
	}
	if versionNewer("0.17.02563", "0.17.02563") {
		t.Fatal("same version is not newer")
	}
	if versionNewer("0.17.02562", "0.17.02563") {
		t.Fatal("lower build is not newer")
	}
	if !versionNewer("0.18.00001", "0.17.09999") {
		t.Fatal("minor version outranks build number")
	}
	if versionNewer("garbage", "0.17.02563") {
		t.Fatal("unparseable versions are never offered")
	}
}

func TestPickReleaseAsset(t *testing.T) {
	release := &githubRelease{Assets: []githubReleaseAsset{
		{Name: "Draftline-0.18.02600-beta-windows-amd64-setup.exe"},
		{Name: "Draftline-0.18.02600-beta-windows-amd64-portable.zip"},
		{Name: "Draftline-0.18.02600-beta-macos-universal.dmg"},
		{Name: "Draftline-0.18.02600-beta-linux-x86_64.AppImage"},
		{Name: "SHA256SUMS.txt"},
	}}
	cases := []struct {
		goos, goarch, want string
	}{
		{"windows", "amd64", "Draftline-0.18.02600-beta-windows-amd64-setup.exe"},
		{"darwin", "arm64", "Draftline-0.18.02600-beta-macos-universal.dmg"},
		{"darwin", "amd64", "Draftline-0.18.02600-beta-macos-universal.dmg"},
		{"linux", "amd64", "Draftline-0.18.02600-beta-linux-x86_64.AppImage"},
	}
	for _, c := range cases {
		asset := pickReleaseAsset(release, c.goos, c.goarch)
		if asset == nil || asset.Name != c.want {
			t.Fatalf("pickReleaseAsset(%s/%s) = %v, want %s", c.goos, c.goarch, asset, c.want)
		}
	}
	if pickReleaseAsset(release, "linux", "arm64") != nil {
		t.Fatal("platforms without a published package must not match another platform's asset")
	}
	// The portable zip must never shadow the installer.
	if asset := pickReleaseAsset(release, "windows", "amd64"); asset == nil || asset.Name != cases[0].want {
		t.Fatalf("windows picked %v", asset)
	}
}

func TestDiscardNoteReportsWhatHappened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rejected-download.exe")
	if err := os.WriteFile(path, []byte("bad bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if note := discardNote(path); note != " The file was discarded." {
		t.Fatalf("removable file: %q", note)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("the rejected file must actually be deleted")
	}
	// Already-gone files still count as discarded, not as a failure.
	if note := discardNote(path); note != " The file was discarded." {
		t.Fatalf("missing file: %q", note)
	}
	// A path that cannot be removed (its parent does not permit it — use a
	// directory with contents, which os.Remove refuses) must tell the user
	// the file is still there.
	dir := filepath.Join(t.TempDir(), "held")
	if err := os.MkdirAll(filepath.Join(dir, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if note := discardNote(dir); note == " The file was discarded." {
		t.Fatal("an undeletable path must not be reported as discarded")
	}
}

func TestParseSHA256Sums(t *testing.T) {
	content := "" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  Draftline-1.0.0-windows-amd64-setup.exe\n" +
		"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB *Draftline-1.0.0-macos-universal.dmg\n" +
		"not-a-checksum-line\n"
	sums := parseSHA256Sums(content)
	if sums["Draftline-1.0.0-windows-amd64-setup.exe"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("plain entry: %#v", sums)
	}
	if sums["Draftline-1.0.0-macos-universal.dmg"] != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("binary-mode entry must strip the asterisk and lowercase: %#v", sums)
	}
	if len(sums) != 2 {
		t.Fatalf("junk lines must be ignored: %#v", sums)
	}
}
