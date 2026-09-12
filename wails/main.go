package main

import (
	"embed"
	"os"

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
			// Serves installed plugin frontend bundles at /plugins/<id>/…
			// so the webview can import them same-origin. Middleware, not
			// Handler: it must win before the dev server's SPA fallback.
			Middleware: app.pluginAssetMiddleware(),
		},
		BackgroundColour: &options.RGBA{R: 43, G: 45, B: 48, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdownPlugins,
		Bind: []any{
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
