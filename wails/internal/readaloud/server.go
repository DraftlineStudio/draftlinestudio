package readaloud

import (
	"fmt"
	"net"
	"net/http"
)

// Loopback model server. WebView2's custom-scheme asset handler does not
// reliably intercept requests initiated inside nested workers (the ONNX
// runtime's pthread workers), so the runtime and model files are served from
// a real HTTP origin on 127.0.0.1 instead. The server binds an ephemeral
// port at first use; the frontend asks for the URL via a binding.
//
// Header policy: the loopback origin is cross-origin to the wails.localhost
// document, so under the app's COEP require-corp everything served here
// carries Cross-Origin-Resource-Policy: cross-origin plus permissive CORS
// (module imports and wasm fetches are CORS-gated cross-origin). COEP is set
// so worker scripts loaded from here remain usable inside the isolated
// agent cluster. Bound strictly to 127.0.0.1; the paths are read-only,
// traversal-proof, and extension-whitelisted (same serving core as the
// asset-handler fallback).
func StartServer(dir string) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("read aloud model server: %w", err)
	}
	inner := fileHandler(dir)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Range, Content-Type")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin")
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		inner.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	return fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port), nil
}
