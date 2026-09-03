package main

import (
	"embed"
	"net/http"
	"os"

	"draftline/internal/readaloud"

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
	if wd, err := os.Getwd(); err == nil {
		if path := launchFilePath(os.Args[1:], wd); path != "" {
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
			// Fallback for paths missing from the embedded frontend: serves the
			// downloaded Read Aloud voice model read-only from the user cache.
			Handler: readaloud.NewHandler(readAloudModelDir()),
			// Cross-origin isolation gives the webview SharedArrayBuffer, which
			// lets the Read Aloud ONNX runtime run real WASM threads instead of
			// being silently capped at one. Everything the app loads is
			// same-origin (embedded assets + the model handler), so COEP
			// require-corp blocks nothing.
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
					w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
					w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
					next.ServeHTTP(w, r)
				})
			},
		},
		BackgroundColour: &options.RGBA{R: 43, G: 45, B: 48, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    true,
		},
		Mac: &mac.Options{
			OnFileOpen: app.onMacFileOpen,
		},
		// Double-clicking a document while Draftline is running focuses the
		// existing window and forwards the file instead of starting a second
		// process.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "com.draftline.app.single-instance",
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
