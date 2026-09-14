package main

import (
	"os"
	"path/filepath"
	"strings"
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
		{Name: "Draftline-0.18.02600-beta-linux-x86_64.deb"},
		{Name: "Draftline-0.18.02600-beta-linux-x86_64.rpm"},
		{Name: "SHA256SUMS.txt"},
	}}
	cases := []struct {
		goos, goarch, linuxKind, want string
	}{
		{"windows", "amd64", "", "Draftline-0.18.02600-beta-windows-amd64-setup.exe"},
		{"darwin", "arm64", "", "Draftline-0.18.02600-beta-macos-universal.dmg"},
		{"darwin", "amd64", "", "Draftline-0.18.02600-beta-macos-universal.dmg"},
		// Linux gets the package matching how this copy was installed.
		{"linux", "amd64", "appimage", "Draftline-0.18.02600-beta-linux-x86_64.AppImage"},
		{"linux", "amd64", "deb", "Draftline-0.18.02600-beta-linux-x86_64.deb"},
		{"linux", "amd64", "rpm", "Draftline-0.18.02600-beta-linux-x86_64.rpm"},
	}
	for _, c := range cases {
		asset := pickReleaseAsset(release, c.goos, c.goarch, c.linuxKind)
		if asset == nil || asset.Name != c.want {
			t.Fatalf("pickReleaseAsset(%s/%s/%q) = %v, want %s", c.goos, c.goarch, c.linuxKind, asset, c.want)
		}
	}
	if pickReleaseAsset(release, "linux", "arm64", "appimage") != nil {
		t.Fatal("platforms without a published package must not match another platform's asset")
	}
	// A Linux build that did not come from a release package (source build,
	// hand-unpacked tarball) has no in-app update path.
	if pickReleaseAsset(release, "linux", "amd64", "") != nil {
		t.Fatal("an unknown Linux install kind must not be offered a package")
	}
	// The portable zip must never shadow the installer.
	if asset := pickReleaseAsset(release, "windows", "amd64", ""); asset == nil || asset.Name != cases[0].want {
		t.Fatalf("windows picked %v", asset)
	}
}

// The terminal script must install with the detected package manager and,
// on failure, keep its window open rather than vanishing with the error.
func TestPackageInstallScript(t *testing.T) {
	script := packageInstallScript("/home/w/.cache/Draftline/updates/Draftline-0.19.02606-linux-x86_64.deb")
	for _, want := range []string{
		`PKG="/home/w/.cache/Draftline/updates/Draftline-0.19.02606-linux-x86_64.deb"`,
		"sudo apt-get install -y \"$PKG\"",
		"sudo dnf install -y \"$PKG\"",
		"sudo zypper --non-interactive install --allow-unsigned-rpm \"$PKG\"",
		"(setsid draftline >/dev/null 2>&1 &)",
		"read -r _",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("install script missing %q:\n%s", want, script)
		}
	}
}

func TestLinuxAssetSuffix(t *testing.T) {
	for kind, want := range map[string]string{
		"appimage": "-linux-x86_64.AppImage",
		"deb":      "-linux-x86_64.deb",
		"rpm":      "-linux-x86_64.rpm",
		"":         "",
		"snap":     "",
	} {
		if got := linuxAssetSuffix(kind); got != want {
			t.Fatalf("linuxAssetSuffix(%q) = %q, want %q", kind, got, want)
		}
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
