package readaloud

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
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

// TestNativeSynthesisRealBundle installs the pinned platform runtime into an
// existing verified bundle, loads it without CGO, and measures two consecutive
// syntheses. It is opt-in because it downloads roughly 40-70 MB in addition
// to the shared fp32 model when those files are absent.
func TestNativeSynthesisRealBundle(t *testing.T) {
	if os.Getenv("READALOUD_NATIVE_E2E") == "" {
		t.Skip("set READALOUD_NATIVE_E2E=1 and READALOUD_E2E_DIR to test native synthesis")
	}
	dir := os.Getenv("READALOUD_E2E_DIR")
	if dir == "" {
		t.Fatal("READALOUD_E2E_DIR is required so the native test can reuse the installed fp32 model")
	}
	if err := InstallNative(context.Background(), dir, nil); err != nil {
		t.Fatalf("install native bundle: %v", err)
	}
	engine, err := NewNativeEngine(dir, min(2, max(1, runtime.NumCPU()-1)))
	if err != nil {
		t.Fatalf("load native engine: %v", err)
	}
	defer engine.Close()
	texts := []string{
		"A short sentence ended. Then the longer sentence arrived with enough detail to test whether continuous narration could stay ahead without an awkward pause.",
		"Hanlon crossed the room, checked the map, and quietly asked Ruiz whether the signal had moved east of the river.",
		"The answer arrived through the radio a moment later, carrying more static than certainty.",
	}
	for i, text := range texts {
		started := time.Now()
		bytes := 0
		if err := engine.Synthesize(context.Background(), text, "af_heart", 1.2, func(block []byte) error {
			bytes += len(block)
			return nil
		}); err != nil {
			t.Fatalf("synthesis %d: %v", i+1, err)
		}
		audioSeconds := float64(bytes/4) / float64(engine.SampleRate())
		wall := time.Since(started)
		t.Logf("native synthesis %d: wall=%s audio=%.2fs RTF=%.3f", i+1, wall, audioSeconds, wall.Seconds()/audioSeconds)
	}

	// The controller requests several lookahead sentences concurrently. Two
	// independent native sessions test aggregate throughput without relying on
	// unsafe re-entry into one ONNX session.
	second, err := NewNativeEngine(dir, 2)
	if err != nil {
		t.Fatalf("load second native engine: %v", err)
	}
	defer second.Close()
	third, err := NewNativeEngine(dir, 2)
	if err != nil {
		t.Fatalf("load third native engine: %v", err)
	}
	defer third.Close()
	parallelStarted := time.Now()
	var wg sync.WaitGroup
	var totalBytes int
	var resultMu sync.Mutex
	for i, engine := range []*NativeEngine{engine, second, third} {
		wg.Add(1)
		go func(i int, engine *NativeEngine) {
			defer wg.Done()
			localBytes := 0
			if err := engine.Synthesize(context.Background(), texts[i], "af_heart", 1.2, func(block []byte) error { localBytes += len(block); return nil }); err != nil {
				t.Errorf("parallel synthesis %d: %v", i+1, err)
			}
			resultMu.Lock()
			totalBytes += localBytes
			resultMu.Unlock()
		}(i, engine)
	}
	wg.Wait()
	combinedAudio := float64(totalBytes/4) / float64(engine.SampleRate())
	wall := time.Since(parallelStarted)
	t.Logf("three-session native pipeline: wall=%s combined-audio=%.2fs aggregate-RTF=%.3f", wall, combinedAudio, wall.Seconds()/combinedAudio)
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
