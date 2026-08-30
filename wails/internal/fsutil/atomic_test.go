package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomic_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	want := []byte("hello atomic world")
	if err := WriteFileAtomic(path, want, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("content mismatch: got %q", got)
	}
}

func TestWriteFileAtomic_OverwriteReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.bin")
	if err := os.WriteFile(path, []byte("old contents that are longer"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Fatalf("expected full replacement, got %q", got)
	}
}

func TestWriteFileAtomic_FailureLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	// Make the target an existing non-empty directory so the final rename fails.
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(filepath.Join(target, "child"), 0755); err != nil {
		t.Fatal(err)
	}
	err := WriteFileAtomic(target, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error when target is a non-empty directory")
	}
	// Original directory must be untouched.
	if _, statErr := os.Stat(filepath.Join(target, "child")); statErr != nil {
		t.Fatalf("original target was damaged: %v", statErr)
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFileAtomic_TempCleanedUpOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(filepath.Join(target, "child"), 0755); err != nil {
		t.Fatal(err)
	}
	_ = WriteFileAtomic(target, []byte("data"), 0644)
	assertNoTempFiles(t, dir)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".draftline-tmp-") {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}
}
