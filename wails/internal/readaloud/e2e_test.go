package readaloud

import (
	"context"
	"os"
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
