package readaloud

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func pcmDiagnostics(t *testing.T, label string, pcm []byte) {
	t.Helper()
	if len(pcm)%4 != 0 {
		t.Fatalf("%s: PCM byte count %d is not float32 aligned", label, len(pcm))
	}
	peak, sumSquares, clipped, invalid := float64(0), float64(0), 0, 0
	firstSignal, lastSignal := -1, -1
	for offset := 0; offset < len(pcm); offset += 4 {
		sample := float64(math.Float32frombits(binary.LittleEndian.Uint32(pcm[offset : offset+4])))
		if math.IsNaN(sample) || math.IsInf(sample, 0) {
			invalid++
			continue
		}
		absolute := math.Abs(sample)
		if absolute > peak {
			peak = absolute
		}
		if absolute >= 1 {
			clipped++
		}
		sumSquares += sample * sample
		if absolute >= 0.002 {
			index := offset / 4
			if firstSignal == -1 {
				firstSignal = index
			}
			lastSignal = index
		}
	}
	samples := max(1, len(pcm)/4-invalid)
	rms := math.Sqrt(sumSquares / float64(samples))
	leadingMS, trailingMS := float64(firstSignal)/24, float64(samples-1-lastSignal)/24
	t.Logf("%s: samples=%d peak=%.5f rms=%.5f clipped=%d invalid=%d leading=%.1fms trailing=%.1fms sha256=%x", label, len(pcm)/4, peak, rms, clipped, invalid, leadingMS, trailingMS, sha256.Sum256(pcm))
	if invalid != 0 || peak > 1.001 {
		t.Fatalf("%s: malformed PCM (peak %.5f, invalid %d)", label, peak, invalid)
	}
}

// TestInstallRealBundle downloads the actual pinned native bundle and so
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

// TestNativeLoopbackRealBundle exercises the exact cross-origin HTTP contract
// used by the Wails WebView, including the exposed sample-rate header.
func TestNativeLoopbackRealBundle(t *testing.T) {
	if os.Getenv("READALOUD_NATIVE_E2E") == "" {
		t.Skip("set READALOUD_NATIVE_E2E=1 and READALOUD_E2E_DIR to test native synthesis")
	}
	dir := os.Getenv("READALOUD_E2E_DIR")
	if dir == "" {
		t.Fatal("READALOUD_E2E_DIR is required for the native test")
	}
	if err := InstallNative(context.Background(), dir, nil); err != nil {
		t.Fatalf("install native bundle: %v", err)
	}
	base, err := StartServer(dir)
	if err != nil {
		t.Fatalf("start loopback: %v", err)
	}
	t.Cleanup(ShutdownNative)
	body := strings.NewReader(`{"text":"The native interface reports its sample rate.","voice":"af_heart","speed":1.2,"threads":0}`)
	req, err := http.NewRequest(http.MethodPost, base+NativeSynthesisPath, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "wails://wails.localhost")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("native request: %v", err)
	}
	defer resp.Body.Close()
	pcm, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read PCM: %v", err)
	}
	if resp.StatusCode != http.StatusOK || len(pcm) == 0 {
		t.Fatalf("native response status=%d bytes=%d", resp.StatusCode, len(pcm))
	}
	if got := resp.Header.Get("X-Draftline-Sample-Rate"); got != "24000" {
		t.Fatalf("sample rate = %q, want 24000", got)
	}
	if exposed := resp.Header.Get("Access-Control-Expose-Headers"); !strings.Contains(exposed, "X-Draftline-Sample-Rate") {
		t.Fatalf("sample-rate header is not exposed to the WebView: %q", exposed)
	}
	pcmDiagnostics(t, "loopback", pcm)
}

// TestNativeSynthesisRealBundle installs the pinned platform runtime and model,
// loads them without CGO, and measures consecutive syntheses.
func TestNativeSynthesisRealBundle(t *testing.T) {
	if os.Getenv("READALOUD_NATIVE_E2E") == "" {
		t.Skip("set READALOUD_NATIVE_E2E=1 and READALOUD_E2E_DIR to test native synthesis")
	}
	dir := os.Getenv("READALOUD_E2E_DIR")
	if dir == "" {
		t.Fatal("READALOUD_E2E_DIR is required for the native test")
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

	// Repeat requests through the same serialized engine and validate each
	// waveform. Kokoro itself may vary slightly between generations; waveform
	// identity is not a supported contract.
	pcm := make([][]byte, 2)
	for i := range pcm {
		if err := engine.Synthesize(context.Background(), texts[0], "af_heart", 1.2, func(block []byte) error {
			pcm[i] = append(pcm[i], block...)
			return nil
		}); err != nil {
			t.Fatalf("serialized determinism synthesis %d: %v", i+1, err)
		}
		pcmDiagnostics(t, fmt.Sprintf("serialized synthesis %d", i+1), pcm[i])
	}
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
	victim := Manifest()[0]
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
