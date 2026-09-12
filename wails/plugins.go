package main

// Plugin platform bindings: the generic bridge between the webview and
// installed plugins. The host never loads plugin code into this process —
// frontend bundles are served to the webview (see pluginAssets in main.go)
// and backend logic runs in supervised sidecar executables
// (internal/plugins). These are the only bound methods the platform needs;
// individual plugins never add bindings of their own.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"draftline/internal/plugins"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// pluginInvokeTimeout bounds a single sidecar call. Long-running work
// (downloads, synthesis) reports through notifications, not a held call.
const pluginInvokeTimeout = 120 * time.Second

type pluginHost struct {
	sup *plugins.Supervisor

	mu        sync.Mutex
	devDir    string
	installed map[string]plugins.Installed
}

// pluginDevDir resolves the development override root: the --plugin-dev flag
// wins over the DRAFTLINE_PLUGIN_DEV environment variable.
func pluginDevDir(args []string) string {
	for i, arg := range args {
		if arg == "--plugin-dev" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(arg, "--plugin-dev="); ok {
			return v
		}
	}
	return os.Getenv("DRAFTLINE_PLUGIN_DEV")
}

func (a *App) initPlugins(devDir string) {
	a.plugins = &pluginHost{
		devDir:    devDir,
		installed: map[string]plugins.Installed{},
	}
	a.plugins.sup = plugins.NewSupervisor(
		func(pluginID, event string, payload json.RawMessage) {
			// Forward sidecar notifications to the webview. Payload stays a
			// JSON value; unmarshal so Wails re-marshals it cleanly.
			var data any
			if len(payload) > 0 {
				if err := json.Unmarshal(payload, &data); err != nil {
					data = string(payload)
				}
			}
			runtime.EventsEmit(a.ctx, "plugin:"+pluginID+":"+event, data)
		},
		func(pluginID, line string) {
			log.Printf("[plugin %s] %s", pluginID, line)
		},
	)
	a.refreshPlugins()
}

func (a *App) refreshPlugins() []plugins.Installed {
	h := a.plugins
	found := plugins.Discover(plugins.Roots(h.devDir))
	h.mu.Lock()
	h.installed = map[string]plugins.Installed{}
	for _, inst := range found {
		h.installed[inst.Manifest.ID] = inst
	}
	h.mu.Unlock()
	return found
}

func (a *App) installedPlugin(id string) (plugins.Installed, bool) {
	h := a.plugins
	h.mu.Lock()
	inst, ok := h.installed[id]
	h.mu.Unlock()
	return inst, ok
}

// PluginInfo is the listing the frontend renders slots from.
type PluginInfo struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Publisher   string                `json:"publisher"`
	Description string                `json:"description"`
	APIVersion  int                   `json:"api_version"`
	Supported   bool                  `json:"supported"` // api_version matches this host
	Permissions []string              `json:"permissions"`
	Activation  []string              `json:"activation"`
	Contributes plugins.Contributions `json:"contributes"`
	HasFrontend bool                  `json:"has_frontend"`
	// FrontendURL is the webview path of the entry module ("" without one).
	FrontendURL string `json:"frontend_url"`
	HasSidecar  bool   `json:"has_sidecar"` // for the running platform
	Running     bool   `json:"running"`
	Root        string `json:"root"` // "dev" | "shared" | "user"
	LoadError   string `json:"load_error"`
}

// PluginList rescans the plugin roots and returns every installed plugin,
// including broken ones (load_error set) so the UI can surface them.
func (a *App) PluginList() []PluginInfo {
	found := a.refreshPlugins()
	out := make([]PluginInfo, 0, len(found))
	for _, inst := range found {
		m := inst.Manifest
		info := PluginInfo{
			ID:          m.ID,
			Name:        m.Name,
			Version:     m.Version,
			Publisher:   m.Publisher,
			Description: m.Description,
			APIVersion:  m.APIVersion,
			Supported:   m.APIVersion == plugins.APIVersion,
			Permissions: m.Permissions,
			Activation:  m.Activation,
			Contributes: m.Contributes,
			HasFrontend: m.Frontend != nil,
			HasSidecar:  inst.SidecarPath() != "",
			Running:     a.plugins.sup.Running(m.ID),
			Root:        inst.Root,
			LoadError:   inst.LoadError,
		}
		if m.Frontend != nil {
			info.FrontendURL = "/plugins/" + m.ID + "/" + m.Frontend.Entry
		}
		out = append(out, info)
	}
	return out
}

// PluginInvoke routes one request to a plugin's sidecar, starting it if
// needed. Only enabled, supported plugins are reachable; the enabled check is
// authoritative here, not in the frontend.
func (a *App) PluginInvoke(pluginID, method, payloadJSON string) (string, error) {
	inst, ok := a.installedPlugin(pluginID)
	if !ok {
		return "", fmt.Errorf("plugin %s is not installed", pluginID)
	}
	if inst.LoadError != "" {
		return "", fmt.Errorf("plugin %s failed to load: %s", pluginID, inst.LoadError)
	}
	if inst.Manifest.APIVersion != plugins.APIVersion {
		return "", fmt.Errorf("plugin %s targets host API v%d; this build provides v%d",
			pluginID, inst.Manifest.APIVersion, plugins.APIVersion)
	}
	if !a.pluginEnabled(pluginID) {
		return "", fmt.Errorf("plugin %s is disabled", pluginID)
	}
	var params json.RawMessage
	if strings.TrimSpace(payloadJSON) != "" {
		if !json.Valid([]byte(payloadJSON)) {
			return "", fmt.Errorf("plugin %s: payload for %q is not valid JSON", pluginID, method)
		}
		params = json.RawMessage(payloadJSON)
	}
	ctx, cancel := context.WithTimeout(context.Background(), pluginInvokeTimeout)
	defer cancel()
	res, err := a.plugins.sup.Invoke(ctx, inst, method, params)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// PluginSetEnabled flips a plugin's per-user enabled flag (persisted in
// settings.json) and stops its sidecar when disabling. Re-enabling clears any
// crash bench.
func (a *App) PluginSetEnabled(pluginID string, enabled bool) error {
	if _, ok := a.installedPlugin(pluginID); !ok {
		return fmt.Errorf("plugin %s is not installed", pluginID)
	}
	s := a.getSettings()
	if s.PluginsEnabled == nil {
		s.PluginsEnabled = map[string]bool{}
	}
	s.PluginsEnabled[pluginID] = enabled
	if err := a.SaveSettings(s); err != nil {
		return err
	}
	if enabled {
		a.plugins.sup.ResetStrikes(pluginID)
	} else {
		a.plugins.sup.Stop(pluginID)
	}
	return nil
}

func (a *App) pluginEnabled(pluginID string) bool {
	return a.getSettings().PluginsEnabled[pluginID]
}

// shutdownPlugins stops every sidecar; called from the Wails OnShutdown hook.
func (a *App) shutdownPlugins(_ context.Context) {
	if a.plugins != nil && a.plugins.sup != nil {
		a.plugins.sup.StopAll()
	}
}

// ── Frontend bundle serving ──────────────────────────────────────────────────

// pluginAssetMiddleware intercepts /plugins/* ahead of the rest of the asset
// chain. It must be assetserver Middleware, not the Handler fallback: in
// `wails dev` the frontend dev server answers unknown paths with its SPA
// index.html fallback, so a fallback Handler never sees plugin requests.
func (a *App) pluginAssetMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		serve := a.pluginAssets()
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/plugins/") {
				serve.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// pluginAssets serves plugin frontend files at /plugins/<id>/<path> so the
// webview can dynamically import entry modules same-origin. Only files under
// an installed plugin's directory are reachable, sidecar binaries are never
// served, and there are no directory listings.
func (a *App) pluginAssets() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rest, ok := strings.CutPrefix(r.URL.Path, "/plugins/")
		if !ok {
			http.NotFound(w, r)
			return
		}
		id, rel, ok := strings.Cut(rest, "/")
		if !ok || rel == "" {
			http.NotFound(w, r)
			return
		}
		inst, found := a.installedPlugin(id)
		if !found || inst.LoadError != "" {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(rel, "bin/") {
			http.NotFound(w, r)
			return
		}
		clean := filepath.Clean(filepath.FromSlash(rel))
		if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(inst.Dir, clean)
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, full)
	})
}
