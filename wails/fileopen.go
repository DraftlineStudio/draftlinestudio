package main

// OS file-open plumbing: paths arrive either as a launch argument (Windows/
// Linux file associations), via the single-instance handoff (double-clicking
// a file while Draftline is already running), or through macOS's open-file
// event. All three funnel into one pending slot + one frontend event; the
// frontend routes by extension through the normal open/import flows so the
// unsaved-changes dialog is always respected.

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Extensions the app can actually open. .storiverse and .pdf are deliberately
// absent: nothing can open a universe yet, and there is no PDF importer.
var openableExtensions = map[string]bool{
	".draftline": true,
	".epub":      true,
	".docx":      true,
}

var (
	pendingOpenMu   sync.Mutex
	pendingOpenPath string
)

func setPendingOpenPath(path string) {
	pendingOpenMu.Lock()
	defer pendingOpenMu.Unlock()
	pendingOpenPath = path
}

// TakePendingOpenPath hands the frontend the file the app was launched with,
// exactly once. Empty string when the app was started normally.
func (a *App) TakePendingOpenPath() string {
	pendingOpenMu.Lock()
	defer pendingOpenMu.Unlock()
	path := pendingOpenPath
	pendingOpenPath = ""
	return path
}

// launchFilePath picks the first openable, existing file from a launch
// argument list (excluding the executable itself). Relative paths resolve
// against workingDir when provided.
func launchFilePath(args []string, workingDir string) string {
	for _, arg := range args {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		if !openableExtensions[strings.ToLower(filepath.Ext(arg))] {
			continue
		}
		path := arg
		if !filepath.IsAbs(path) && workingDir != "" {
			path = filepath.Join(workingDir, path)
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// onSecondInstanceLaunch runs in the FIRST instance when a second one starts
// (e.g. the user double-clicked another book). Focus the window and forward
// the file to the frontend.
func (a *App) onSecondInstanceLaunch(data options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
	// Pass Args through unfiltered: Wails versions differ on whether the
	// executable path is included, and stripping the first element blindly
	// can discard the document path. launchFilePath's extension allowlist
	// already ignores an exe path, so filtering is unnecessary.
	if path := launchFilePath(data.Args, data.WorkingDirectory); path != "" {
		runtime.EventsEmit(a.ctx, "file:open", path)
	}
}

// onMacFileOpen handles macOS's open-file event, which can arrive before the
// frontend is ready (app launched by double-clicking a document).
func (a *App) onMacFileOpen(path string) {
	if !openableExtensions[strings.ToLower(filepath.Ext(path))] {
		return
	}
	if a.ctx == nil {
		setPendingOpenPath(path)
		return
	}
	runtime.EventsEmit(a.ctx, "file:open", path)
}
