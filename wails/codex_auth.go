package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"draftline/internal/platform"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const codexStatusTimeout = 15 * time.Second

type codexProbeRunner func(ctx context.Context, path string, args ...string) ([]byte, error)

// checkCodexCLI asks the CLI itself whether it is authenticated. Credential
// location is deliberately opaque to Draftline: Codex may use auth.json on
// some systems and the OS credential store (including macOS Keychain) on
// others.
func (a *App) checkCodexCLI() types.ClaudeCodeStatus {
	npmAvailable := resolveNpmBin() != ""
	codexPath := resolveCodexBin()
	if codexPath == "" {
		return types.ClaudeCodeStatus{NpmAvailable: npmAvailable}
	}

	ctx, cancel := context.WithTimeout(context.Background(), codexStatusTimeout)
	defer cancel()
	version, authenticated := probeCodexCLI(ctx, codexPath, runCodexHostCommand)
	return types.ClaudeCodeStatus{
		Installed:     true,
		Authenticated: authenticated,
		NpmAvailable:  npmAvailable,
		Version:       version,
	}
}

func probeCodexCLI(ctx context.Context, path string, run codexProbeRunner) (string, bool) {
	versionOut, versionErr := run(ctx, path, "--version")
	version := ""
	if versionErr == nil {
		version = strings.TrimSpace(string(versionOut))
	}

	// Exit status is the public CLI contract: zero means an active login. Do
	// not parse human-readable output, which may change or be localized.
	_, authErr := run(ctx, path, "login", "status")
	return version, authErr == nil
}

func runCodexHostCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := codexExec(ctx, path, args...)
	prepareCodexHostCommand(cmd)
	platform.HideWindow(cmd)
	return cmd.CombinedOutput()
}

// prepareCodexHostCommand makes the managed npm CLI reliable when Draftline
// is launched from Finder/Explorer, where GUI applications often receive a
// smaller PATH than an interactive shell. HOME remains the user's real home
// so Codex can locate file-backed credentials where applicable; on macOS the
// CLI remains free to use Keychain instead.
func prepareCodexHostCommand(cmd *exec.Cmd) {
	home, _ := os.UserHomeDir()
	nodePath := nodeInstallBinDir()
	pathValue := nodePath
	if inherited := os.Getenv("PATH"); inherited != "" {
		pathValue += string(os.PathListSeparator) + inherited
	}

	overrides := map[string]string{"PATH": pathValue}
	if home != "" {
		overrides["HOME"] = home
		overrides["USERPROFILE"] = home
	}
	cmd.Env = replaceCommandEnv(os.Environ(), overrides)
}

func replaceCommandEnv(base []string, overrides map[string]string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if found {
			if _, replaced := overrides[strings.ToUpper(key)]; replaced {
				continue
			}
		}
		result = append(result, entry)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

// openCodexAuth runs "codex login" as a background process. The CLI opens
// the user's browser and waits for the OAuth callback. When it exits, the UI
// re-runs "codex login status" rather than guessing where credentials landed.
func (a *App) openCodexAuth() {
	codexPath := resolveCodexBin()
	if codexPath == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := codexExec(ctx, codexPath, "login")
		prepareCodexHostCommand(cmd)
		platform.HideWindow(cmd)
		if err := cmd.Start(); err == nil {
			_ = cmd.Wait()
		}
		runtime.EventsEmit(a.ctx, "codex:auth_complete", nil)
	}()
}
