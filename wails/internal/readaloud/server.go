package readaloud

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

const NativeSynthesisPath = "/readaloud-native/synthesize"

type nativeServer struct {
	dir  string
	mu   sync.Mutex
	pool *nativeEnginePool
}

type nativeEnginePool struct {
	engines    chan *NativeEngine
	all        []*NativeEngine
	sampleRate int
}

func (s *nativeServer) ensurePool(singleThread bool) (*nativeEnginePool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pool != nil {
		return s.pool, nil
	}
	count := min(3, max(1, runtime.NumCPU()/4))
	// Kokoro scales better across independent lookahead requests than through
	// a large intra-op thread count. Two threads per session matched the
	// three-thread throughput on the 12-thread reference machine with less CPU.
	threads := min(2, max(1, runtime.NumCPU()-1))
	if singleThread {
		count, threads = 1, 1
	}

	created := make([]*NativeEngine, count)
	errs := make(chan error, count)
	var wait sync.WaitGroup
	for i := range created {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			engine, err := NewNativeEngine(s.dir, threads)
			if err != nil {
				errs <- err
				return
			}
			created[index] = engine
		}(i)
	}
	wait.Wait()
	close(errs)
	if err := <-errs; err != nil {
		for _, engine := range created {
			if engine != nil {
				engine.Close()
			}
		}
		return nil, err
	}
	pool := &nativeEnginePool{engines: make(chan *NativeEngine, count), all: created, sampleRate: created[0].SampleRate()}
	for _, engine := range created {
		pool.engines <- engine
	}
	s.pool = pool
	return pool, nil
}

func (s *nativeServer) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pool == nil {
		return
	}
	for _, engine := range s.pool.all {
		engine.Close()
	}
	s.pool = nil
}

var runningNativeServers struct {
	sync.Mutex
	servers []*nativeServer
}

// ShutdownNative releases model mappings and shared-library handles. It waits
// for any callback currently leaving native code before returning, allowing
// Windows to remove downloaded DLLs safely as well as Unix platforms.
func ShutdownNative() {
	runningNativeServers.Lock()
	servers := append([]*nativeServer(nil), runningNativeServers.servers...)
	runningNativeServers.Unlock()
	for _, server := range servers {
		server.close()
	}
}

func (s *nativeServer) serveSynthesis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		Text    string  `json:"text"`
		Voice   string  `json:"voice"`
		Speed   float32 `json:"speed"`
		Threads int     `json:"threads"`
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || strings.TrimSpace(request.Text) == "" {
		http.Error(w, "invalid synthesis request", http.StatusBadRequest)
		return
	}
	pool, err := s.ensurePool(request.Threads == 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Draftline-Sample-Rate", fmt.Sprintf("%d", pool.sampleRate))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	var engine *NativeEngine
	select {
	case engine = <-pool.engines:
	case <-r.Context().Done():
		return
	}
	defer func() { pool.engines <- engine }()
	err = engine.Synthesize(r.Context(), request.Text, request.Voice, request.Speed, func(block []byte) error {
		if _, writeErr := w.Write(block); writeErr != nil {
			return writeErr
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	// Headers have already been sent so a late native/cancellation error is
	// represented by a truncated response; fetch rejects on a broken socket.
	_ = err
}

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
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		_ = listener.Close()
		return "", fmt.Errorf("read aloud model server token: %w", err)
	}
	pathPrefix := "/" + hex.EncodeToString(secret)
	inner := fileHandler(dir)
	native := &nativeServer{dir: dir}
	runningNativeServers.Lock()
	runningNativeServers.servers = append(runningNativeServers.servers, native)
	runningNativeServers.Unlock()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The random capability path prevents unrelated web content from using
		// Draftline's loopback synthesizer or probing its downloaded model files.
		// A new value is generated for every app process and is never persisted.
		if !strings.HasPrefix(r.URL.Path, pathPrefix+"/") {
			http.NotFound(w, r)
			return
		}
		request := r.Clone(r.Context())
		requestURL := *r.URL
		requestURL.Path = strings.TrimPrefix(r.URL.Path, pathPrefix)
		requestURL.RawPath = ""
		request.URL = &requestURL
		r = request

		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, HEAD, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Range, Content-Type")
		h.Set("Cross-Origin-Resource-Policy", "cross-origin")
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == NativeSynthesisPath {
			native.serveSynthesis(w, r)
			return
		}
		inner.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	return fmt.Sprintf("http://127.0.0.1:%d%s", listener.Addr().(*net.TCPAddr).Port, pathPrefix), nil
}
