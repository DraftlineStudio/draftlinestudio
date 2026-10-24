package main

import (
	"embed"
	"os"
	"runtime"
	"strings"

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

// KWin on Wayland draws its own title bar and frame around the frameless
// GTK3 window: the decoration negotiation between GTK3 and KWin never
// settles on "none", so the custom title bar gets a second, native one
// around it. The X11 backend is immune — XWayland windows carry the
// undecorated hint KWin honors — so prefer it on KDE Wayland sessions.
// An explicit GDK_BACKEND always wins, and a session without XWayland
// (no DISPLAY) is left alone rather than broken. GNOME and X11 sessions
// never reach the Setenv.
func preferX11OnKDEWayland(getenv func(string) string) bool {
	if runtime.GOOS != "linux" || getenv("GDK_BACKEND") != "" || getenv("DISPLAY") == "" {
		return false
	}
	if getenv("XDG_SESSION_TYPE") != "wayland" && getenv("WAYLAND_DISPLAY") == "" {
		return false
	}
	return strings.Contains(strings.ToLower(getenv("XDG_CURRENT_DESKTOP")), "kde")
}

func main() {
	app := NewApp()

	if preferX11OnKDEWayland(os.Getenv) {
		os.Setenv("GDK_BACKEND", "x11")
	}

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
			// A pinch would scale the whole interface, WebView2's browser
			// default. Text size is the editor font size in settings.
			DisablePinchZoom: true,
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
