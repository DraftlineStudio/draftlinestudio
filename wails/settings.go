package main

// Settings persistence and the API key.
//
// The key lives in the OS keyring. On a machine without one it falls back to
// settings.json, which is why that file and its directory are 0700 and why the
// key is stripped from everything that crosses the Wails bridge: the frontend
// is told whether a key exists, never what it is.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"draftline/internal/fsutil"
	"draftline/internal/logging"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
)

// ── Settings ─────────────────────────────────────────────────────────────────

// configRoot resolves the per-user configuration directory. It is a variable
// rather than a direct call so tests can point it somewhere harmless: the
// functions below write the writer's real settings file, and a test that
// reaches the real one destroys their setup. See TestMain.
var configRoot = os.UserConfigDir

// configDir is the per-user directory holding settings and the recents list.
// It is 0700 because settings.json carries a plaintext API key on machines
// with no keyring; an existing looser directory from an older build is
// tightened best-effort.
func configDir() string {
	base, err := configRoot()
	if err != nil {
		if base, err = os.UserHomeDir(); err != nil {
			return ""
		}
	}
	dir := filepath.Join(base, "draftline")
	_ = os.MkdirAll(dir, 0700)
	_ = os.Chmod(dir, 0700)
	return dir
}

func (a *App) settingsPath() string {
	return filepath.Join(configDir(), "settings.json")
}

// loadSettingsFromDisk reads settings.json verbatim (including a legacy
// plaintext API key, which only startup's migration may see).
// defaultSaveDir is Documents/Draftline. A writer's Documents folder is
// usually already crowded, and projects bring lock files and exports with
// them; a folder of their own is also somewhere a sync client can be pointed
// at without capturing everything else.
func defaultSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	parent := filepath.Join(home, "Documents")
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		parent = home
	}
	return filepath.Join(parent, "Draftline")
}

// bookDialogDir is the folder the open and save dialogs start in, created if
// it is missing so the dialog does not silently fall back elsewhere.
func (a *App) bookDialogDir() string {
	dir := a.getSettings().DefaultSaveDir
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

func (a *App) loadSettingsFromDisk() types.AppSettings {
	defaults := types.AppSettings{
		DefaultSaveDir:          defaultSaveDir(),
		AIEnabled:               false,
		AIMode:                  "claudecode",
		AITaskRoutes:            map[string]string{},
		DarkMode:                true,
		ThemeMode:               "dark",
		AutoThemeUseManual:      true,
		AutoThemeDawn:           "06:30",
		AutoThemeDusk:           "19:00",
		ActivityAutoSaveEnabled: true,
		CustomDictionary:        []string{},
		SpellCheckEnabled:       true,
		GrammarCheckEnabled:     true,
		CastEnabled:             true,
		StoryBibleEnabled:       true,
		AnalysisEnabled:         true,
		ReadAloudVoice:          "af_heart",
		ReadAloudSpeed:          1.1,
		ReadAloudDevice:         "native",
		ReadAloudThreads:        "auto",
		ReadAloudVolume:         1,
		ReadAloudGlow:           true,
		AnalysisCPUProfile:      "adaptive",
		CharactersLaneView:      "grid",
		// Minimum sidebar width by default so unmaximised windows keep
		// editor room; a user-dragged width overrides on save.
		SidebarPanelWidth: 280,
		// Open on the Writing Dashboard by default; "" means closed.
		SidebarActiveSection: "dashboard",
		UpdateCheckEnabled:   true,
	}
	data, err := os.ReadFile(a.settingsPath())
	if err != nil {
		return defaults
	}
	// Decode over defaults so settings written by older versions inherit newly
	// introduced feature flags instead of silently disabling them.
	s := defaults
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults
	}
	// Settings written before this field had a default carry an empty string,
	// which decodes over the default rather than leaving it alone.
	if strings.TrimSpace(s.DefaultSaveDir) == "" {
		s.DefaultSaveDir = defaults.DefaultSaveDir
	}
	return s
}

// LoadSettings returns the settings for the frontend. The API key itself
// never crosses the Wails bridge — only the has_api_key flag does.
func (a *App) LoadSettings() types.AppSettings {
	s := a.loadSettingsFromDisk()
	s.AIAPIKey = ""
	// Provider keys are stored outside the keyring on machines without one.
	// They must not cross the bridge any more than the main key does.
	for i := range s.AIProviders {
		s.AIProviders[i].APIKey = ""
	}
	s.HasAPIKey = a.HasAPIKey()
	return s
}

func (a *App) writeSettingsFile(settings types.AppSettings) error {
	settings.HasAPIKey = false // derived at load time, not persisted
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(a.settingsPath(), data, 0600)
}

func (a *App) SaveSettings(settings types.AppSettings) error {
	// Never trust a key from the frontend; keys arrive via SetAPIKey only.
	settings.AIAPIKey = ""
	// Providers have their own calls. A settings round-trip must not revert
	// them, and cannot: the frontend's copy has the keys stripped.
	settings.AIProviders = a.getSettings().AIProviders
	a.setSettings(settings)
	logging.SetEnabled(settings.AIDebugLogging)

	onDisk := settings
	if legacy := a.getLegacyAPIKey(); legacy != "" {
		// Keyring unavailable on this machine: keep the stored key so a
		// settings round-trip doesn't wipe it.
		onDisk.AIAPIKey = legacy
	}
	return a.writeSettingsFile(onDisk)
}

// ── API key (OS keyring) ─────────────────────────────────────────────────────

const (
	keyringService = "draftline"
	keyringAccount = "ai_api_key"
)

// SetAPIKey stores the key in the OS keyring. If no keyring is available it
// falls back to settings.json (user-only permissions) rather than losing it.
func (a *App) SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key is empty")
	}
	if err := keyring.Set(keyringService, keyringAccount, key); err != nil {
		log.Printf("keyring unavailable (%v); storing key in settings.json", err)
		a.setLegacyAPIKey(key)
		a.setCachedKey(key)
		onDisk := a.getSettings()
		onDisk.AIAPIKey = key
		return a.writeSettingsFile(onDisk)
	}
	a.setLegacyAPIKey("")
	a.setCachedKey(key)
	// Make sure no plaintext copy lingers on disk.
	onDisk := a.getSettings()
	onDisk.AIAPIKey = ""
	return a.writeSettingsFile(onDisk)
}

// HasAPIKey reports whether a key is stored, without revealing it.
func (a *App) HasAPIKey() bool {
	return a.getAPIKey() != ""
}

// ClearAPIKey removes the stored key from the keyring and settings.json.
// Local copies (legacy plaintext, cache) are cleared even if the keyring
// operation fails — on keyring-less machines settings.json is the only place
// the key lives, and clearing must not be blocked by an "unavailable" error.
func (a *App) ClearAPIKey() error {
	delErr := keyring.Delete(keyringService, keyringAccount)
	if delErr != nil && errors.Is(delErr, keyring.ErrNotFound) {
		delErr = nil
	}
	a.setLegacyAPIKey("")
	a.setCachedKey("")
	onDisk := a.getSettings()
	onDisk.AIAPIKey = ""
	if err := a.writeSettingsFile(onDisk); err != nil {
		return err
	}
	return delErr
}

func (a *App) setCachedKey(key string) {
	a.apiKeyMu.Lock()
	a.apiKeyCache = key
	a.apiKeyLoaded = true
	a.apiKeyMu.Unlock()
}

// getAPIKey returns the stored key (keyring first, then legacy fallback),
// caching the keyring read.
func (a *App) getAPIKey() string {
	a.apiKeyMu.Lock()
	defer a.apiKeyMu.Unlock()
	if a.apiKeyLoaded {
		return a.apiKeyCache
	}
	v, err := keyring.Get(keyringService, keyringAccount)
	switch {
	case err == nil:
		a.apiKeyCache = v
	case errors.Is(err, keyring.ErrNotFound):
		a.apiKeyCache = a.legacyAPIKey
	default:
		// Transient keyring failure: fall back without caching so the next
		// call retries the keyring.
		return a.legacyAPIKey
	}
	a.apiKeyLoaded = true
	return a.apiKeyCache
}

func (a *App) BrowseForDirectory() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Default Save Location",
	})
	if err != nil {
		return ""
	}
	return path
}
