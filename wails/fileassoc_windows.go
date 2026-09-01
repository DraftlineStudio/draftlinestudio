//go:build windows

package main

// Best-effort, per-user file associations. Written under HKCU\Software\Classes
// (no admin rights needed, works for the portable exe):
//   - .draftline is claimed as Draftline's own document type.
//   - .epub/.docx only gain an "Open with → Draftline" entry — the user's
//     default ebook reader / Word association is never touched.
// Everything is idempotent and re-run on every launch so a moved exe heals
// its own registration. The dev binary (draftline-dev.exe) never registers,
// so associations can't end up pointing at a wails-dev build.

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const draftlineProgID = "Draftline.Project"

// Document icon: the dl-icon mark on the brand-blue rounded square
// (generated from reference-assets/imgs/dl-icon.png; PNG-compressed
// multi-size ICO). Extracted to %LOCALAPPDATA%\Draftline at registration
// because DefaultIcon needs a stable on-disk path.
//
//go:embed build/windows/draftline-doc.ico
var docIconData []byte

// ensureDocIcon writes the document icon beside the user's app data and
// returns its path, or "" to fall back to the exe's own icon.
func ensureDocIcon() string {
	base, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, "Draftline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	path := filepath.Join(dir, "draftline-doc.ico")
	if existing, err := os.ReadFile(path); err != nil || !bytes.Equal(existing, docIconData) {
		if err := os.WriteFile(path, docIconData, 0o644); err != nil {
			return ""
		}
	}
	return path
}

func registerFileAssociations() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	base := strings.ToLower(filepath.Base(exe))
	if base != "draftline.exe" {
		return nil // dev or renamed binary: don't claim associations
	}
	command := fmt.Sprintf(`"%s" "%%1"`, exe)
	docIcon := ensureDocIcon()
	if docIcon == "" {
		docIcon = fmt.Sprintf(`"%s",0`, exe) // fall back to the app icon
	}

	type entry struct {
		path  string
		name  string // "" for the default value
		value string
	}
	entries := []entry{
		// Bump this value whenever the registration schema changes: it forces
		// one changed=true pass (and thus one shell refresh) for users whose
		// entries are otherwise already correct.
		{`Software\Classes\` + draftlineProgID, "RegistrationVersion", "3"},
		// Owned document type for .draftline.
		{`Software\Classes\` + draftlineProgID, "", "Draftline Project"},
		{`Software\Classes\` + draftlineProgID + `\DefaultIcon`, "", docIcon},
		{`Software\Classes\` + draftlineProgID + `\shell\open\command`, "", command},
		{`Software\Classes\.draftline`, "", draftlineProgID},
		{`Software\Classes\.draftline\OpenWithProgids`, draftlineProgID, ""},
		// "Open with" entry (non-default) for the importable formats.
		{`Software\Classes\Applications\draftline.exe`, "FriendlyAppName", "Draftline"},
		{`Software\Classes\Applications\draftline.exe\shell\open\command`, "", command},
		{`Software\Classes\Applications\draftline.exe\SupportedTypes`, ".draftline", ""},
		{`Software\Classes\Applications\draftline.exe\SupportedTypes`, ".epub", ""},
		{`Software\Classes\Applications\draftline.exe\SupportedTypes`, ".docx", ""},
	}

	changed := false
	for _, e := range entries {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, e.path, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s: %w", e.path, err)
		}
		// Write only on change so we know when the shell needs a refresh —
		// and so quiet launches stay quiet.
		if existing, _, err := key.GetStringValue(e.name); err != nil || existing != e.value {
			if err := key.SetStringValue(e.name, e.value); err != nil {
				key.Close()
				return fmt.Errorf("set %s[%s]: %w", e.path, e.name, err)
			}
			changed = true
		}
		key.Close()
	}

	if changed {
		// Without this, Explorer keeps showing the cached blank-page icon for
		// .draftline files until the icon cache happens to rebuild.
		notifyShellAssocChanged()
	}
	return nil
}

var procSHChangeNotify = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")

const (
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0x0000
)

func notifyShellAssocChanged() {
	_, _, _ = procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
}
