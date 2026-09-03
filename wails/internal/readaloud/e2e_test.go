package readaloud

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestInstallRealBundle downloads the actual pinned bundle (~130 MB) and so
// verifies every manifest checksum against the live sources. It is skipped
// unless READALOUD_E2E=1; set READALOUD_E2E_DIR to install into a specific
// directory (e.g. the real model cache) instead of a throwaway temp dir.
func TestInstallRealBundle(t *testing.T) {
	if os.Getenv("READALOUD_E2E") == "" {
		t.Skip("set READALOUD_E2E=1 to download and verify the real pinned bundle")
	}
	dir := os.Getenv("READALOUD_E2E_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	var last Progress
	if err := Install(context.Background(), dir, func(p Progress) { last = p }); err != nil {
		t.Fatalf("install: %v", err)
	}
	if last.OverallTotal != TotalBytes() {
		t.Errorf("progress total %d, want %d", last.OverallTotal, TotalBytes())
	}
	status := Check(dir)
	if !status.Installed {
		t.Errorf("bundle not fully installed; missing: %v", status.Missing)
	}
	t.Logf("bundle verified in %s (%d bytes)", dir, status.BytesOnDisk)
}

// TestFreshInstallLifecycle walks the full install lifecycle against real
// downloads: not-installed → install (with manifest) → verified → deliberate
// corruption → repair re-downloads only the damaged file → verified →
// remove → confirmed empty. Gated like the bundle test; set
// READALOUD_E2E_DIR to exercise the real cache directory.
func TestFreshInstallLifecycle(t *testing.T) {
	if os.Getenv("READALOUD_E2E") == "" {
		t.Skip("set READALOUD_E2E=1 to run the real download lifecycle")
	}
	base := os.Getenv("READALOUD_E2E_DIR")
	if base == "" {
		base = filepath.Join(t.TempDir(), "kokoro")
	}

	// 1. Fresh state.
	if err := Remove(base); err != nil {
		t.Fatalf("initial clean: %v", err)
	}
	if v := Verify(base); v.Installed || v.Verified {
		t.Fatalf("fresh dir must verify as not installed: %+v", v)
	}
	t.Log("fresh: not installed ✓")

	// 2. Install.
	if err := Install(context.Background(), base, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if readInstalledManifest(base) == nil {
		t.Fatal("install must write manifest.json")
	}
	v := Verify(base)
	if !v.Verified {
		t.Fatalf("post-install verify failed: %+v", v)
	}
	t.Logf("installed: v%s, %d bytes, all hashes match ✓", v.Version, v.Bytes)

	// 3. Corrupt one file (right size, wrong bytes) and confirm detection.
	victim := GroupManifest(GroupCore)[0]
	victimPath := filepath.Join(base, filepath.FromSlash(victim.Name))
	if err := os.WriteFile(victimPath, make([]byte, victim.Bytes), 0644); err != nil {
		t.Fatal(err)
	}
	v = Verify(base)
	if v.Verified || len(v.Corrupt) != 1 || v.Corrupt[0] != victim.Name {
		t.Fatalf("corruption not detected: %+v", v)
	}
	t.Logf("corruption detected: %v ✓", v.Corrupt)

	// 4. Repair: Install must re-download only the damaged file.
	if err := Install(context.Background(), base, nil); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if v = Verify(base); !v.Verified {
		t.Fatalf("post-repair verify failed: %+v", v)
	}
	t.Log("repair re-downloaded the damaged file; verified ✓")

	// 5. Remove and confirm nothing is left.
	if err := Remove(base); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("model directory still exists after remove")
	}
	if v = Verify(base); v.Installed {
		t.Fatalf("post-remove verify must report not installed: %+v", v)
	}
	t.Log("removed: directory gone, verify reports not installed ✓")
}
