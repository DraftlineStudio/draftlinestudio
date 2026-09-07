package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"draftline/internal/platform"
)

// prepareClaudeRequestCommand confines Claude's config discovery to the
// temporary request home and prevents its console-subsystem executable from
// opening a terminal over the Wails application on Windows.
func prepareClaudeRequestCommand(cmd *exec.Cmd, tempHome string) {
	cmd.Dir = tempHome

	nodeDir := nodeInstallBinDir()
	baseEnv := os.Environ()
	filteredEnv := make([]string, 0, len(baseEnv)+5)
	for _, entry := range baseEnv {
		key, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(key) {
		case "PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH":
			continue
		}
		filteredEnv = append(filteredEnv, entry)
	}

	volume := filepath.VolumeName(tempHome)
	cmd.Env = append(filteredEnv,
		"PATH="+nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+tempHome,
		"USERPROFILE="+tempHome,
		"HOMEDRIVE="+volume,
		"HOMEPATH="+strings.TrimPrefix(tempHome, volume),
	)
	platform.HideWindow(cmd)
}
