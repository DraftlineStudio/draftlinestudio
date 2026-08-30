package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func isolateConfigDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
}

func writeProject(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSkipsMissingFile(t *testing.T) {
	isolateConfigDir(t)
	if err := Create(filepath.Join(t.TempDir(), "nope.draftline")); err != nil {
		t.Fatalf("Create on missing file should be a no-op: %v", err)
	}
}

func TestCreateRotatesAndCaps(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	for i := 1; i <= MaxBackups+2; i++ {
		writeProject(t, path, fmt.Sprintf("version-%d", i))
		if err := Create(path); err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
	}

	backups := List(path)
	if len(backups) != MaxBackups {
		t.Fatalf("expected %d backups, got %d", MaxBackups, len(backups))
	}
	// backup.1 must hold the most recent pre-save state (version-7).
	data, err := os.ReadFile(backups[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != fmt.Sprintf("version-%d", MaxBackups+2) {
		t.Fatalf("backup.1 content: %q", data)
	}
}

func TestRestoreRoundTrip(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	writeProject(t, path, "original")
	if err := Create(path); err != nil {
		t.Fatal(err)
	}
	writeProject(t, path, "modified")

	res := Restore(path, 1)
	if !res.Success {
		t.Fatalf("Restore failed: %s", res.Error)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("restored content: %q", data)
	}
	// Pre-restore state must itself have been backed up (reversible restore).
	backups := List(path)
	if len(backups) == 0 {
		t.Fatal("expected a pre-restore backup")
	}
	data, _ = os.ReadFile(backups[0].Path)
	if string(data) != "modified" {
		t.Fatalf("pre-restore backup content: %q", data)
	}
}

func TestRestoreInvalidNumber(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")
	writeProject(t, path, "x")

	if res := Restore(path, 0); res.Success {
		t.Fatal("number 0 should fail")
	}
	if res := Restore(path, MaxBackups+1); res.Success {
		t.Fatal("number beyond MaxBackups should fail")
	}
	if res := Restore(path, 2); res.Success {
		t.Fatal("missing backup should fail")
	}
	if res := Restore("", 1); res.Success {
		t.Fatal("empty path should fail")
	}
}
