package main

// Read Aloud plugin backend: thin bound methods over internal/readaloud.
// Heavy lifting (pinned manifest, verified download, asset handler) lives in
// the package; this file only wires it to the frontend and the Wails event
// bus. Synthesis itself runs in the webview — no audio work happens in Go.

import (
	"context"
	"os"
	"path/filepath"
	"sync"

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

// ReadAloudStatus reports whether the voice model bundle is fully installed.
func (a *App) ReadAloudStatus() readaloud.Status {
	return readaloud.Check(readAloudModelDir())
}

// DownloadReadAloudModel starts (or resumes) the pinned core-bundle download
// in the background. Progress arrives on "readaloud:progress" and completion
// on "readaloud:done". A second call while a download runs is a no-op.
func (a *App) DownloadReadAloudModel() {
	a.downloadReadAloudGroup(readaloud.GroupCore)
}

// DownloadReadAloudGPUModel downloads the optional full-precision model for
// the WebGPU fast path (~311 MB), with the same verification and resume
// semantics as the core bundle.
func (a *App) DownloadReadAloudGPUModel() {
	a.downloadReadAloudGroup(readaloud.GroupGPU)
}

func (a *App) downloadReadAloudGroup(group string) {
	readAloudDownload.mu.Lock()
	if readAloudDownload.cancel != nil {
		readAloudDownload.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	readAloudDownload.cancel = cancel
	readAloudDownload.mu.Unlock()

	go func() {
		err := readaloud.InstallGroup(ctx, readAloudModelDir(), group, func(p readaloud.Progress) {
			runtime.EventsEmit(a.ctx, "readaloud:progress", p)
		})

		readAloudDownload.mu.Lock()
		readAloudDownload.cancel = nil
		readAloudDownload.mu.Unlock()

		payload := map[string]any{"ok": err == nil, "group": group}
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
