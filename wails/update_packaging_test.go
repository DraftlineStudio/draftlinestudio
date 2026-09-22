package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The AppImage swap must leave the destination holding the complete new image,
// with the executable bit set, without ever holding the whole package in
// memory. It used to read the file into a []byte and write it back out, which
// for a package the updater allows up to 600 MiB is a large avoidable
// allocation immediately after the download and hash pass.
//
// Start is expected to fail here: the "AppImage" is not executable content.
// What this pins is the file that is left behind.
func TestReplacingAnAppImageLeavesTheCompleteNewOne(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "Draftline.AppImage")
	downloaded := filepath.Join(dir, "Draftline-new.AppImage")

	if err := os.WriteFile(current, []byte("the running image"), 0o755); err != nil {
		t.Fatal(err)
	}
	replacement := make([]byte, 1<<20)
	for i := range replacement {
		replacement[i] = byte(i % 251)
	}
	if err := os.WriteFile(downloaded, replacement, 0o644); err != nil {
		t.Fatal(err)
	}

	replaceAppImage(current, downloaded)

	got, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("the running image is gone: %v", err)
	}
	if len(got) != len(replacement) {
		t.Fatalf("swapped in %d bytes, the download is %d", len(got), len(replacement))
	}
	for i := range got {
		if got[i] != replacement[i] {
			t.Fatalf("the swapped image differs from the download at byte %d", i)
		}
	}
	if _, err := os.Stat(downloaded); !os.IsNotExist(err) {
		t.Error("the verified download should be cleaned up after the swap")
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.Name() != filepath.Base(current) {
				t.Errorf("a staging file was left behind: %s", e.Name())
			}
		}
	}
}
