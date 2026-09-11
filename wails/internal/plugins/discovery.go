package plugins

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// Installed is one discovered plugin directory. A directory whose manifest
// fails to parse is still reported (with LoadError set) so the UI can show a
// broken install instead of silently hiding it.
type Installed struct {
	Dir       string
	Manifest  Manifest
	LoadError string
	// Root records which root the plugin was found under ("dev", "shared",
	// "user") — informational, surfaced in the plugins UI.
	Root string
}

// SharedRoot returns the machine-wide Draftline data directory: one copy of
// every plugin, model, and tool per machine regardless of how many OS users
// run the app. The installer creates it (elevated) with a users-writable
// ACL on Windows; everything loaded from it is verified before use because
// it is shared-writable space.
func SharedRoot() string {
	switch runtime.GOOS {
	case "windows":
		if pd := os.Getenv("ProgramData"); pd != "" {
			return filepath.Join(pd, "Draftline")
		}
		return ""
	case "darwin":
		return "/Library/Application Support/Draftline"
	default:
		return "/var/lib/draftline"
	}
}

// UserRoot returns the per-user fallback root, used when the shared root is
// not present or not writable (portable installs, locked-down machines).
func UserRoot() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "draftline")
}

// Root describes one plugin search root.
type Root struct {
	Name string // "dev" | "shared" | "user"
	Dir  string // the plugins directory under that root
}

// Roots returns the plugin search roots in precedence order. devDir, when
// non-empty (--plugin-dev / DRAFTLINE_PLUGIN_DEV), is searched first so a
// plugin under development shadows an installed copy of the same id.
func Roots(devDir string) []Root {
	var roots []Root
	if devDir != "" {
		roots = append(roots, Root{Name: "dev", Dir: devDir})
	}
	if shared := SharedRoot(); shared != "" {
		roots = append(roots, Root{Name: "shared", Dir: filepath.Join(shared, "plugins")})
	}
	if user := UserRoot(); user != "" {
		roots = append(roots, Root{Name: "user", Dir: filepath.Join(user, "plugins")})
	}
	return roots
}

// Discover scans the given roots for plugin directories. The first root that
// provides a plugin id wins; later occurrences of the same id are ignored.
// Roots that don't exist are skipped silently. Results are sorted by id for
// stable listings.
func Discover(roots []Root) []Installed {
	seen := map[string]bool{}
	var out []Installed
	for _, root := range roots {
		entries, err := os.ReadDir(root.Dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root.Dir, e.Name())
			inst := Installed{Dir: dir, Root: root.Name}
			m, err := ParseManifest(dir)
			if err != nil {
				// A directory without a manifest.json at all is not a plugin
				// (e.g. the shared root's .staging dir); skip it quietly.
				if os.IsNotExist(err) {
					continue
				}
				inst.LoadError = err.Error()
				// Identify a broken plugin by its directory name so the UI
				// can point at it.
				inst.Manifest.ID = e.Name()
			} else {
				inst.Manifest = m
			}
			if inst.Manifest.ID == "" || seen[inst.Manifest.ID] {
				continue
			}
			seen[inst.Manifest.ID] = true
			out = append(out, inst)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.ID < out[j].Manifest.ID })
	return out
}

// PlatformKey is the "<goos>-<goarch>" key used in Manifest.Sidecar.
func PlatformKey() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// SidecarPath resolves the plugin's sidecar executable for this platform.
// Returns "" when the plugin has no sidecar for the running platform.
func (i Installed) SidecarPath() string {
	rel, ok := i.Manifest.Sidecar[PlatformKey()]
	if !ok {
		return ""
	}
	return filepath.Join(i.Dir, filepath.FromSlash(rel))
}

// FrontendEntryPath resolves the plugin's frontend entry module on disk.
// Returns "" when the plugin has no frontend.
func (i Installed) FrontendEntryPath() string {
	if i.Manifest.Frontend == nil {
		return ""
	}
	return filepath.Join(i.Dir, filepath.FromSlash(i.Manifest.Frontend.Entry))
}
