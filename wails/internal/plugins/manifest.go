// Package plugins implements the Draftline plugin platform host: manifest
// parsing and validation, plugin discovery across the machine-wide and
// per-user roots, and supervision of plugin sidecar processes (JSON-RPC 2.0
// over stdio). Plugin code never runs in this process: frontend bundles are
// loaded by the webview, and backend logic lives in sidecar executables.
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SchemaVersion is the manifest schema this host reads.
const SchemaVersion = 1

// APIVersion is the host API version this build exposes to plugins. A plugin
// declaring a different api_version is listed but never activated.
const APIVersion = 1

// Manifest is a plugin's manifest.json. Every capability a plugin has is
// declared here; the host renders contribution slots from this file without
// executing any plugin code.
type Manifest struct {
	Schema      int      `json:"schema"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Publisher   string   `json:"publisher"`
	APIVersion  int      `json:"api_version"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Activation  []string `json:"activation,omitempty"`
	// Frontend names the plugin's JS entry module, relative to the plugin
	// directory (conventionally "frontend/index.js").
	Frontend *FrontendSpec `json:"frontend,omitempty"`
	// Sidecar maps "<goos>-<goarch>" to the executable path relative to the
	// plugin directory (e.g. "windows-amd64": "bin/plugin.exe").
	Sidecar     map[string]string `json:"sidecar,omitempty"`
	Contributes Contributions     `json:"contributes,omitempty"`
}

type FrontendSpec struct {
	Entry string `json:"entry"`
}

// Contributions are the declarative UI slots a plugin fills. The host shows
// these (settings nav items, dock bars, shortcuts) before the plugin's code
// is loaded; activation happens when a slot is first exercised.
type Contributions struct {
	EditorDockBars   []DockBarContribution         `json:"editorDockBars,omitempty"`
	SettingsSections []SettingsSectionContribution `json:"settingsSections,omitempty"`
	Commands         []CommandContribution         `json:"commands,omitempty"`
}

type DockBarContribution struct {
	ID string `json:"id"`
}

type SettingsSectionContribution struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type CommandContribution struct {
	ID string `json:"id"`
	// Shortcut uses "Mod+Shift+X" notation; Mod is Ctrl on Windows/Linux and
	// Cmd on macOS. Optional — commands may be invocation-only.
	Shortcut string `json:"shortcut,omitempty"`
}

// KnownPermissions is the closed set of permission strings a manifest may
// declare. Validation fails on anything else so new capabilities are a
// deliberate host change, never a typo.
var KnownPermissions = map[string]string{
	"editor.read":     "read the manuscript text and selection",
	"editor.decorate": "highlight ranges in the editor",
	"propose-edits":   "propose changes through the review panel",
	"book.plugin-data": "store plugin data inside the book file " +
		"(namespaced to the plugin)",
	"settings":               "keep per-user plugin settings",
	"secrets":                "store secrets in the OS keyring (namespaced)",
	"network":                "network access from the plugin's own process",
	"network:model-download": "download pinned, hash-verified artifacts",
}

// KnownActivationEvents is the closed set of activation triggers.
var KnownActivationEvents = map[string]bool{
	"onStartupIfEnabled": true,
	"onCommand":          true,
	"onSettingsOpen":     true,
	"onBookOpen":         true,
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*(\.[a-z0-9][a-z0-9-]*)+$`)

// ParseManifest reads and validates a plugin directory's manifest.json.
func ParseManifest(dir string) (Manifest, error) {
	var m Manifest
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return m, err
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return m, fmt.Errorf("manifest.json: %w", err)
	}
	if err := m.Validate(); err != nil {
		return m, err
	}
	return m, nil
}

// Validate enforces the manifest contract. It never touches the filesystem;
// path fields are checked for shape only (relative, forward slashes, no
// traversal) so a manifest can be validated in isolation.
func (m Manifest) Validate() error {
	if m.Schema != SchemaVersion {
		return fmt.Errorf("unsupported manifest schema %d (host reads %d)", m.Schema, SchemaVersion)
	}
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("invalid plugin id %q (want publisher.name in lowercase kebab)", m.ID)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("plugin %s: name is required", m.ID)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("plugin %s: version is required", m.ID)
	}
	if strings.TrimSpace(m.Publisher) == "" {
		return fmt.Errorf("plugin %s: publisher is required", m.ID)
	}
	// api_version is validated for presence here; whether the host supports
	// it is an activation-time decision so listing still works.
	if m.APIVersion < 1 {
		return fmt.Errorf("plugin %s: api_version is required", m.ID)
	}
	for _, p := range m.Permissions {
		if _, ok := KnownPermissions[p]; !ok {
			return fmt.Errorf("plugin %s: unknown permission %q", m.ID, p)
		}
	}
	for _, a := range m.Activation {
		if !KnownActivationEvents[a] {
			return fmt.Errorf("plugin %s: unknown activation event %q", m.ID, a)
		}
	}
	if m.Frontend != nil {
		if err := validateRelPath(m.Frontend.Entry); err != nil {
			return fmt.Errorf("plugin %s: frontend.entry: %w", m.ID, err)
		}
	}
	for platform, bin := range m.Sidecar {
		if err := validateRelPath(bin); err != nil {
			return fmt.Errorf("plugin %s: sidecar[%s]: %w", m.ID, platform, err)
		}
	}
	if m.Frontend == nil && len(m.Sidecar) == 0 {
		return fmt.Errorf("plugin %s: declares neither a frontend entry nor a sidecar", m.ID)
	}
	for _, c := range m.Contributes.Commands {
		if strings.TrimSpace(c.ID) == "" {
			return fmt.Errorf("plugin %s: command without an id", m.ID)
		}
	}
	for _, s := range m.Contributes.SettingsSections {
		if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.Title) == "" {
			return fmt.Errorf("plugin %s: settings section needs id and title", m.ID)
		}
	}
	for _, d := range m.Contributes.EditorDockBars {
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("plugin %s: dock bar without an id", m.ID)
		}
	}
	return nil
}

// validateRelPath accepts clean, forward-slash relative paths that stay
// inside the plugin directory.
func validateRelPath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path %q must use forward slashes", p)
	}
	if strings.HasPrefix(p, "/") || filepath.IsAbs(p) || (len(p) > 1 && p[1] == ':') {
		return fmt.Errorf("path %q must be relative", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "." || seg == "" {
			return fmt.Errorf("path %q must be clean (no . or .. segments)", p)
		}
	}
	return nil
}
