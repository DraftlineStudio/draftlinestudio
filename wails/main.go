package main

import (
	"embed"
	"os"

	"draftline/internal/instancelock"
	"draftline/internal/platform"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// File passed by the OS (file association / "Open with") at launch.
	// Draftline is multi-instance; only the same BOOK is exclusive. If the
	// double-clicked book is already open in a living instance, hand focus
	// to that window and exit before any UI appears (Word-style).
	if wd, err := os.Getwd(); err == nil {
		if path := launchFilePath(os.Args[1:], wd); path != "" {
			if owner := instancelock.CurrentOwner(path); owner != nil {
				platform.FocusProcessWindow(owner.PID)
				return
			}
			setPendingOpenPath(path)
		}
	}

	err := wails.Run(&options.App{
		Title:     "Draftline",
		Width:     1400,
		Height:    900,
		MinWidth:  960,
		MinHeight: 600,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
			// Serves installed plugin frontend bundles at /plugins/<id>/…
			// so the webview can import them same-origin. Middleware, not
			// Handler: it must win before the dev server's SPA fallback.
			Middleware: app.pluginAssetMiddleware(),
		},
		// Dropping cover artwork onto the window hands Go the file's PATH,
		// which is the only form internal/coverart accepts. Without this the
		// webview would receive the file's contents instead, and a forty
		// megabyte TIFF would have to cross the JSON bridge to get back to
		// disk. See cover.go.
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		BackgroundColour: &options.RGBA{R: 43, G: 45, B: 48, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.onShutdown,
		Bind: []any{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Mac: &mac.Options{
			OnFileOpen: app.onMacFileOpen,
		},
		// No app-level SingleInstanceLock: Draftline allows multiple windows
		// (different books side by side, dev builds next to the installed
		// app). Exclusivity is per BOOK via internal/instancelock — opening
		// a book that's already open foregrounds its window instead.
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
