package readaloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeArtifact(name string, body []byte, server string) Artifact {
	sum := sha256.Sum256(body)
	return Artifact{
		Name:   name,
		URL:    server + "/" + name,
		SHA256: hex.EncodeToString(sum[:]),
		Bytes:  int64(len(body)),
	}
}

func serveFiles(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInstallHappyPathAndProgress(t *testing.T) {
	files := map[string][]byte{
		"hf/config.json":  []byte(`{"model_type":"test"}`),
		"hf/model.onnx":   make([]byte, 1<<20),
		"ort/runtime.mjs": []byte("export default 1"),
	}
	srv := serveFiles(t, files)
	manifest := []Artifact{
		fakeArtifact("hf/config.json", files["hf/config.json"], srv.URL),
		fakeArtifact("hf/model.onnx", files["hf/model.onnx"], srv.URL),
		fakeArtifact("ort/runtime.mjs", files["ort/runtime.mjs"], srv.URL),
	}
	dir := t.TempDir()

	var lastOverall int64
	progressCalls := 0
	err := installManifest(context.Background(), dir, manifest, func(p Progress) {
		progressCalls++
		if p.OverallReceived < lastOverall {
			t.Errorf("overall progress went backwards: %d after %d", p.OverallReceived, lastOverall)
		}
		lastOverall = p.OverallReceived
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if progressCalls == 0 {
		t.Fatal("expected at least one progress callback")
	}
	var wantTotal int64
	for _, a := range manifest {
		wantTotal += a.Bytes
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(a.Name)))
		if err != nil {
			t.Fatalf("read installed %s: %v", a.Name, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != a.SHA256 {
			t.Errorf("%s content mismatch after install", a.Name)
		}
	}
	if lastOverall != wantTotal {
		t.Errorf("final overall progress %d, want %d", lastOverall, wantTotal)
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "*", "*.partial")); len(entries) != 0 {
		t.Errorf("leftover partial files: %v", entries)
	}
}

func TestInstallChecksumMismatchRejected(t *testing.T) {
	files := map[string][]byte{"hf/model.onnx": []byte("actual bytes")}
	srv := serveFiles(t, files)
	art := fakeArtifact("hf/model.onnx", files["hf/model.onnx"], srv.URL)
	art.SHA256 = strings.Repeat("0", 64)
	dir := t.TempDir()

	err := installManifest(context.Background(), dir, []Artifact{art}, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "hf", "model.onnx")); !os.IsNotExist(statErr) {
		t.Error("mismatched file must not be installed")
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "hf", "*.partial")); len(entries) != 0 {
		t.Errorf("leftover partial files: %v", entries)
	}
}

func TestInstallOversizedBodyRejected(t *testing.T) {
	files := map[string][]byte{"hf/model.onnx": make([]byte, 4096)}
	srv := serveFiles(t, files)
	art := fakeArtifact("hf/model.onnx", files["hf/model.onnx"], srv.URL)
	art.Bytes = 100 // pinned smaller than the served body

	err := installManifest(context.Background(), t.TempDir(), []Artifact{art}, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds pinned size") {
		t.Fatalf("expected oversize rejection, got %v", err)
	}
}

func TestInstallCancelLeavesNoPartial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 512*1024))
		w.(http.Flusher).Flush()
		cancel()
		<-blocked // hold the connection open until the client gives up
	}))
	defer srv.Close()
	defer close(blocked)

	body := make([]byte, 2<<20)
	art := fakeArtifact("hf/model.onnx", body, srv.URL)
	dir := t.TempDir()

	err := installManifest(ctx, dir, []Artifact{art}, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "hf", "*")); len(entries) != 0 {
		t.Errorf("cancelled download left files behind: %v", entries)
	}
}

func TestInstallResumeSkipsVerifiedFiles(t *testing.T) {
	hits := map[string]int{}
	files := map[string][]byte{
		"hf/config.json": []byte(`{"ok":true}`),
		"hf/model.onnx":  make([]byte, 2048),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		hits[name]++
		_, _ = w.Write(files[name])
	}))
	defer srv.Close()
	manifest := []Artifact{
		fakeArtifact("hf/config.json", files["hf/config.json"], srv.URL),
		fakeArtifact("hf/model.onnx", files["hf/model.onnx"], srv.URL),
	}
	dir := t.TempDir()

	// Pre-install the first file as a completed earlier run would have.
	if err := installManifest(context.Background(), dir, manifest[:1], nil); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	if err := installManifest(context.Background(), dir, manifest, nil); err != nil {
		t.Fatalf("resume install: %v", err)
	}
	if hits["hf/config.json"] != 1 {
		t.Errorf("verified file was re-downloaded %d times", hits["hf/config.json"])
	}
	if hits["hf/model.onnx"] != 1 {
		t.Errorf("missing file should download exactly once, got %d", hits["hf/model.onnx"])
	}
}

func TestCheckAndRemove(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kokoro")
	bundle := Manifest()
	status := Check(dir)
	if status.Installed || len(status.Missing) != len(bundle) {
		t.Fatalf("empty dir must report every native file missing, got %+v", status)
	}
	if status.BytesTotal != TotalBytes() {
		t.Errorf("BytesTotal = %d, want %d", status.BytesTotal, TotalBytes())
	}
	// Fabricate a correctly sized file for the first artifact: Check judges by size.
	first := bundle[0]
	path := filepath.Join(dir, filepath.FromSlash(first.Name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, first.Bytes), 0644); err != nil {
		t.Fatal(err)
	}
	status = Check(dir)
	if len(status.Missing) != len(bundle)-1 || status.BytesOnDisk != first.Bytes {
		t.Errorf("after one file: %+v", status)
	}

	if err := Remove(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("dir should be gone after Remove")
	}
	if err := Remove(filepath.Join(t.TempDir(), "not-the-model-dir")); err == nil {
		t.Error("Remove must refuse unexpected directory names")
	}
}

func TestVerifyStates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kokoro")

	// Empty dir: not installed, everything missing.
	result := Verify(dir)
	if result.Installed || result.Verified || len(result.Missing) != len(Manifest()) {
		t.Fatalf("empty dir: %+v", result)
	}

	// A native file with the pinned size but wrong content: corrupt, not verified.
	first := Manifest()[0]
	path := filepath.Join(dir, filepath.FromSlash(first.Name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, first.Bytes), 0644); err != nil {
		t.Fatal(err)
	}
	result = Verify(dir)
	if !result.Installed || result.Verified {
		t.Errorf("corrupt install must be installed-but-unverified: %+v", result)
	}
	if len(result.Corrupt) != 1 || result.Corrupt[0] != first.Name {
		t.Errorf("expected exactly %s corrupt, got %v", first.Name, result.Corrupt)
	}
	// Size mismatch also reads as corrupt.
	if err := os.WriteFile(path, make([]byte, first.Bytes+5), 0644); err != nil {
		t.Fatal(err)
	}
	if result = Verify(dir); len(result.Corrupt) != 1 {
		t.Errorf("size mismatch must be corrupt: %+v", result)
	}
}

func TestInstalledManifestRoundTrip(t *testing.T) {
	// Install a fake bundle, then confirm the real manifest writer records
	// only files that hash-match the compiled pins (none here) while the
	// file itself is valid JSON with provenance fields.
	dir := t.TempDir()
	if err := WriteInstalledManifest(dir); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	manifest := readInstalledManifest(dir)
	if manifest == nil {
		t.Fatal("manifest unreadable after write")
	}
	if manifest.Version != bundleVersion || manifest.ModelID == "" || manifest.InstalledAt == "" {
		t.Errorf("manifest metadata incomplete: %+v", manifest)
	}
	if len(manifest.Files) != 0 {
		t.Errorf("no pinned files exist in a temp dir; recorded %d", len(manifest.Files))
	}
}

func TestInstallWritesManifest(t *testing.T) {
	files := map[string][]byte{"hf/config.json": []byte(`{"ok":true}`)}
	srv := serveFiles(t, files)
	dir := t.TempDir()
	if err := installManifest(context.Background(), dir, []Artifact{
		fakeArtifact("hf/config.json", files["hf/config.json"], srv.URL),
	}, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if readInstalledManifest(dir) == nil {
		t.Error("successful install must leave a manifest.json")
	}
}

func TestLoopbackServer(t *testing.T) {
	dir := t.TempDir()
	base, err := StartServer(dir)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	if !strings.HasPrefix(base, "http://127.0.0.1:") {
		t.Fatalf("server must bind loopback, got %s", base)
	}
	parsedBase, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	if len(strings.TrimPrefix(parsedBase.Path, "/")) != 32 {
		t.Fatalf("server URL must carry a 128-bit capability path, got %q", parsedBase.Path)
	}
	unscoped := parsedBase.Scheme + "://" + parsedBase.Host + NativeSynthesisPath
	if resp, err := http.Get(unscoped); err != nil {
		t.Fatalf("unscoped get: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("unscoped request: got %d, want 404", resp.StatusCode)
		}
	}

	resp, err := http.Get(base + "/readaloud-models/ort/runtime.wasm")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("obsolete browser model route: got %d, want 404", resp.StatusCode)
	}
	preflight, err := http.NewRequest(http.MethodOptions, base+NativeSynthesisPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp, err := http.DefaultClient.Do(preflight); err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("preflight: got %d, want 204", resp.StatusCode)
		}
		if methods := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "POST") {
			t.Errorf("preflight methods %q do not permit native synthesis POST", methods)
		}
		if exposed := resp.Header.Get("Access-Control-Expose-Headers"); !strings.Contains(exposed, "X-Draftline-Sample-Rate") {
			t.Errorf("preflight does not expose native sample rate: %q", exposed)
		}
	}
}

func TestManifestPinsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range Manifest() {
		if seen[a.Name] {
			t.Errorf("duplicate artifact %s", a.Name)
		}
		seen[a.Name] = true
		if len(a.SHA256) != 64 {
			t.Errorf("%s: malformed sha256 %q", a.Name, a.SHA256)
		}
		if a.Bytes <= 0 {
			t.Errorf("%s: non-positive size", a.Name)
		}
		if !strings.Contains(a.URL, nativeRevision) && !strings.Contains(a.URL, nativeInt8Revision) &&
			!strings.Contains(a.URL, sherpaVersion) && a.URL != espeakBundleURL {
			t.Errorf("%s: URL %s is not pinned to a revision", a.Name, a.URL)
		}
		if strings.Contains(a.URL, "/resolve/main/") {
			t.Errorf("%s: mutable branch URL forbidden", a.Name)
		}
		if a.Group != GroupNative {
			t.Errorf("%s: unknown group %q", a.Name, a.Group)
		}
	}
	if fmt.Sprintf("%d", TotalBytes()) == "0" {
		t.Error("TotalBytes must be positive")
	}
	if NativeSupported() && len(GroupManifest(GroupNative)) < 5 {
		t.Error("native bundle must include support files and both platform libraries")
	}
	var nativeSum int64
	for _, a := range GroupManifest(GroupNative) {
		nativeSum += a.Bytes
	}
	if TotalBytes() != nativeSum {
		t.Errorf("TotalBytes must price the native bundle: got %d, want %d", TotalBytes(), nativeSum)
	}
}
