package main

// Read Aloud plugin backend: thin bound methods over internal/readaloud.
// Heavy lifting (pinned manifest, verified download, and native synthesis)
// lives in the package; this file wires it to the frontend and Wails events.

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"draftline/internal/readaloud"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// readAloudModelDir follows the setup.go convention: large downloaded
// binaries live under the user cache directory, small JSON state under the
// config directory.
func readAloudModelDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir, _ = os.UserConfigDir()
	}
	return filepath.Join(dir, "draftline", "models", "kokoro")
}

var readAloudDownload struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

var readAloudServer struct {
	mu  sync.Mutex
	url string
}

// ReadAloudServerURL starts (once) and returns the authenticated loopback
// synthesis service. An empty result is fatal to playback; there is no
// browser or remote fallback.
func (a *App) ReadAloudServerURL() string {
	readAloudServer.mu.Lock()
	defer readAloudServer.mu.Unlock()
	if readAloudServer.url != "" {
		return readAloudServer.url
	}
	url, err := readaloud.StartServer(readAloudModelDir())
	if err != nil {
		return ""
	}
	readAloudServer.url = url
	return url
}

// VerifyReadAloudModel re-hashes every installed file against the pinned
// manifest. Used on plugin enable and before first playback; a corrupt
// result must block model loading and offer repair instead.
func (a *App) VerifyReadAloudModel() readaloud.VerifyResult {
	return readaloud.Verify(readAloudModelDir())
}

var readAloudMemLog struct {
	mu   sync.Mutex
	stop chan struct{}
}

// StartReadAloudMemLog samples WebView2/app process RSS every 2 s while
// playback runs, emitting lines onto the shared diagnostics channel. Safety
// cap of 5 minutes in case the stop call never arrives.
func (a *App) StartReadAloudMemLog() {
	readAloudMemLog.mu.Lock()
	defer readAloudMemLog.mu.Unlock()
	if readAloudMemLog.stop != nil {
		return
	}
	stop := make(chan struct{})
	readAloudMemLog.stop = stop
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		deadline := time.After(5 * time.Minute)
		for {
			select {
			case <-stop:
				return
			case <-deadline:
				a.StopReadAloudMemLog()
				return
			case <-ticker.C:
				for _, line := range readaloud.MemorySnapshot() {
					runtime.EventsEmit(a.ctx, "readaloud:diag", line)
				}
			}
		}
	}()
}

// StopReadAloudMemLog ends RSS sampling.
func (a *App) StopReadAloudMemLog() {
	readAloudMemLog.mu.Lock()
	defer readAloudMemLog.mu.Unlock()
	if readAloudMemLog.stop != nil {
		close(readAloudMemLog.stop)
		readAloudMemLog.stop = nil
	}
}

// ShutdownReadAloudNative releases every hot ONNX session without removing
// its installed files. The next play recreates the bounded pool lazily.
func (a *App) ShutdownReadAloudNative() {
	readaloud.ShutdownNative()
}

// ReadAloudStatus reports whether the voice model bundle is fully installed.
func (a *App) ReadAloudStatus() readaloud.Status {
	return readaloud.Check(readAloudModelDir())
}

// DownloadReadAloudNative installs the platform runtime and the CPU-optimized
// native model/support files.
func (a *App) DownloadReadAloudNative() {
	readAloudDownload.mu.Lock()
	if readAloudDownload.cancel != nil {
		readAloudDownload.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	readAloudDownload.cancel = cancel
	readAloudDownload.mu.Unlock()

	go func() {
		err := readaloud.InstallNative(ctx, readAloudModelDir(), func(p readaloud.Progress) {
			runtime.EventsEmit(a.ctx, "readaloud:progress", p)
		})
		readAloudDownload.mu.Lock()
		readAloudDownload.cancel = nil
		readAloudDownload.mu.Unlock()
		payload := map[string]any{"ok": err == nil, "group": readaloud.GroupNative}
		if err != nil {
			payload["error"] = err.Error()
		}
		runtime.EventsEmit(a.ctx, "readaloud:done", payload)
	}()
}

// CancelReadAloudDownload aborts an in-flight download. Completed files stay
// on disk, so a later download resumes where it left off.
func (a *App) CancelReadAloudDownload() {
	readAloudDownload.mu.Lock()
	defer readAloudDownload.mu.Unlock()
	if readAloudDownload.cancel != nil {
		readAloudDownload.cancel()
	}
}

// RemoveReadAloudModel deletes the downloaded bundle from disk.
func (a *App) RemoveReadAloudModel() error {
	a.CancelReadAloudDownload()
	return readaloud.Remove(readAloudModelDir())
}
