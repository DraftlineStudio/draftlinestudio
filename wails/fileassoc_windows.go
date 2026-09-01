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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const draftlineProgID = "Draftline.Project"

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

	type entry struct {
		path  string
		name  string // "" for the default value
		value string
	}
	entries := []entry{
		// Owned document type for .draftline.
		{`Software\Classes\` + draftlineProgID, "", "Draftline Project"},
		{`Software\Classes\` + draftlineProgID + `\DefaultIcon`, "", fmt.Sprintf(`"%s",0`, exe)},
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

	for _, e := range entries {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, e.path, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s: %w", e.path, err)
		}
		err = key.SetStringValue(e.name, e.value)
		key.Close()
		if err != nil {
			return fmt.Errorf("set %s[%s]: %w", e.path, e.name, err)
		}
	}
	return nil
}
