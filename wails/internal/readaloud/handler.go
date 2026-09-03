package readaloud

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HandlerPrefix is the virtual URL root the webview fetches model files from.
const HandlerPrefix = "/readaloud-models/"

// contentTypes maps served extensions explicitly; application/wasm is required
// for WebAssembly.instantiateStreaming. Anything not listed is refused rather
// than guessed.
var contentTypes = map[string]string{
	".onnx": "application/octet-stream",
	".bin":  "application/octet-stream",
	".json": "application/json",
	".wasm": "application/wasm",
	".mjs":  "text/javascript",
}

// NewHandler serves the installed bundle read-only under HandlerPrefix. It is
// mounted as the Wails asset-server fallback handler, so it answers 404 for
// every path outside its prefix and never shadows embedded frontend assets.
func NewHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rel, ok := strings.CutPrefix(r.URL.Path, HandlerPrefix)
		if !ok || rel == "" {
			http.NotFound(w, r)
			return
		}
		clean, err := secureJoin(dir, rel)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct, ok := contentTypes[strings.ToLower(filepath.Ext(clean))]
		if !ok {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(clean)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ct)
		// The webview runs cross-origin isolated (COOP/COEP set by the main
		// asset middleware). The ONNX runtime spawns nested pthread workers
		// from the .mjs served here, and a worker script loaded into an
		// isolated agent cluster is BLOCKED unless its own response carries
		// COEP — so these headers must be present on this handler too, not
		// only on the embedded-asset chain.
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		http.ServeContent(w, r, "", info.ModTime(), f)
	})
}

// secureJoin resolves rel inside root, rejecting absolute paths, volume names,
// and any traversal outside root (same policy as setup.go's securePath).
func secureJoin(root, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", os.ErrPermission
	}
	joined := filepath.Join(root, rel)
	cleanRoot := filepath.Clean(root) + string(os.PathSeparator)
	if !strings.HasPrefix(joined, cleanRoot) {
		return "", os.ErrPermission
	}
	return joined, nil
}
