package readaloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
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
	core := GroupManifest(GroupCore)
	status := Check(dir)
	if status.Installed || len(status.Missing) != len(core) {
		t.Fatalf("empty dir must report every core file missing, got %+v", status)
	}
	if status.GPUInstalled {
		t.Error("empty dir must report the GPU model missing")
	}
	if status.BytesTotal != TotalBytes() {
		t.Errorf("BytesTotal = %d, want %d", status.BytesTotal, TotalBytes())
	}
	if status.GPUBytesTotal <= 0 {
		t.Error("GPUBytesTotal must be positive")
	}

	// Fabricate a correctly sized file for the first artifact: Check judges by size.
	first := core[0]
	path := filepath.Join(dir, filepath.FromSlash(first.Name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, first.Bytes), 0644); err != nil {
		t.Fatal(err)
	}
	status = Check(dir)
	if len(status.Missing) != len(core)-1 || status.BytesOnDisk != first.Bytes {
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

func TestHandlerServesAndRefuses(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hf"), 0755); err != nil {
		t.Fatal(err)
	}
	wasm := []byte("\x00asm")
	if err := os.WriteFile(filepath.Join(dir, "hf", "model.wasm"), wasm, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("no"), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(dir)

	get := func(method, path string) *http.Response {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec.Result()
	}

	if resp := get("GET", HandlerPrefix+"hf/model.wasm"); resp.StatusCode != http.StatusOK {
		dump, _ := httputil.DumpResponse(resp, true)
		t.Fatalf("expected 200 for served wasm, got:\n%s", dump)
	} else if ct := resp.Header.Get("Content-Type"); ct != "application/wasm" {
		t.Errorf("wasm content type = %q", ct)
	}
	for name, path := range map[string]string{
		"traversal":          HandlerPrefix + "../secret.txt",
		"unknown extension":  HandlerPrefix + "secret.txt",
		"outside prefix":     "/settings.json",
		"missing file":       HandlerPrefix + "hf/nope.wasm",
		"bare prefix":        HandlerPrefix,
		"encoded traversal":  HandlerPrefix + "..%2fsecret.txt",
		"windows abs volume": HandlerPrefix + "C:/Windows/win.ini",
	} {
		if resp := get("GET", path); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s (%s): got %d, want 404", name, path, resp.StatusCode)
		}
	}
	if resp := get("POST", HandlerPrefix+"hf/model.wasm"); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST: got %d, want 405", resp.StatusCode)
	}
}

func TestVerifyStates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kokoro")

	// Empty dir: not installed, everything missing.
	result := Verify(dir)
	if result.Installed || result.Verified || len(result.Missing) != len(GroupManifest(GroupCore)) {
		t.Fatalf("empty dir: %+v", result)
	}

	// A core file with the pinned size but wrong content: corrupt, not verified.
	first := GroupManifest(GroupCore)[0]
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
	if err := os.MkdirAll(filepath.Join(dir, "ort"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ort", "runtime.wasm"), []byte("\x00asm"), 0644); err != nil {
		t.Fatal(err)
	}
	base, err := StartServer(dir)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	if !strings.HasPrefix(base, "http://127.0.0.1:") {
		t.Fatalf("server must bind loopback, got %s", base)
	}

	resp, err := http.Get(base + HandlerPrefix + "ort/runtime.wasm")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	for header, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Cross-Origin-Resource-Policy": "cross-origin",
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Content-Type":                 "application/wasm",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	if resp, err := http.Get(base + HandlerPrefix + "../secret.txt"); err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("traversal: got %d, want 404", resp.StatusCode)
		}
	}
	preflight, err := http.NewRequest(http.MethodOptions, base+HandlerPrefix+"ort/runtime.wasm", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp, err := http.DefaultClient.Do(preflight); err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("preflight: got %d, want 204", resp.StatusCode)
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
		if !strings.Contains(a.URL, kokoroRevision) && !strings.Contains(a.URL, ortVersion) {
			t.Errorf("%s: URL %s is not pinned to a revision", a.Name, a.URL)
		}
		if strings.Contains(a.URL, "/resolve/main/") {
			t.Errorf("%s: mutable branch URL forbidden", a.Name)
		}
		if a.Group != GroupCore && a.Group != GroupGPU {
			t.Errorf("%s: unknown group %q", a.Name, a.Group)
		}
	}
	if fmt.Sprintf("%d", TotalBytes()) == "0" {
		t.Error("TotalBytes must be positive")
	}
	if len(GroupManifest(GroupGPU)) == 0 || len(GroupManifest(GroupCore)) == 0 {
		t.Error("both artifact groups must be non-empty")
	}
	var coreSum int64
	for _, a := range GroupManifest(GroupCore) {
		coreSum += a.Bytes
	}
	if TotalBytes() != coreSum {
		t.Errorf("TotalBytes must price the core bundle only: got %d, want %d", TotalBytes(), coreSum)
	}
}
