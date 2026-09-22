package main

// The Codex CLI driver.
//
// Codex runs as a separate process rather than over HTTP, so there is no key
// to hold: the CLI owns the writer's ChatGPT session. That is the whole reason
// this path exists beside the API providers.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"draftline/internal/ai/providers"
	"draftline/internal/platform"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CheckCodexCLI checks whether the OpenAI Codex CLI is installed and
// authenticated (ChatGPT-account mode). Reuses the ClaudeCodeStatus shape.
func (a *App) CheckCodexCLI() types.ClaudeCodeStatus {
	return a.checkCodexCLI()
}

// OpenCodexAuth runs "codex login" as a hidden background process. It opens
// the user's browser for the ChatGPT OAuth flow — the in-app intercept for
// what would otherwise require running /login in a terminal. When the process
// exits it emits "codex:auth_complete".
func (a *App) OpenCodexAuth() {
	a.openCodexAuth()
}

// callCodexCLI handles the "codex" AI mode: prose rewrites through the OpenAI
// Codex CLI using the user's ChatGPT account.
func (a *App) callCodexCLI(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	return a.callCodexCLIAtPath(ctx, resolveCodexBin(), system, userMsg, profile)
}

// callCodexCLIAtPath keeps the managed executable dependency injectable for
// tests. Production always reaches it through callCodexCLI/resolveCodexBin.
func (a *App) callCodexCLIAtPath(ctx context.Context, path, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	if path == "" {
		return types.AIRewriteResult{Error: "Codex CLI is not installed — open Settings › AI Studio to set it up"}
	}

	// Isolated home: real auth.json so the ChatGPT session works, but none of
	// the user's config.toml (MCP servers etc. could hang the subprocess).
	tempHome, err := os.MkdirTemp("", "draftline-codex-*")
	if err != nil {
		return types.AIRewriteResult{Error: "cannot create temp dir: " + err.Error()}
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	realHome, _ := os.UserHomeDir()
	codexDir := filepath.Join(tempHome, ".codex")
	_ = os.MkdirAll(codexDir, 0755)
	for _, cachedFile := range []string{"auth.json", "models_cache.json"} {
		if data, e2 := os.ReadFile(filepath.Join(realHome, ".codex", cachedFile)); e2 == nil {
			_ = os.WriteFile(filepath.Join(codexDir, cachedFile), data, 0600)
		}
	}

	lastMsg := filepath.Join(tempHome, "last-message.txt")
	model := providers.CodexModelOverride(a.getSettings().AIModel)
	if profile.tier == tierLite {
		// Prefer a fast model this installation actually advertises; an empty
		// override falls through to the CLI's own default.
		model = providers.ResolveCodexLightweightModel(realHome)
		if model != "" {
			runtime.EventsEmit(a.ctx, "ai:log", "Fast edit model: "+model)
		}
	}
	cmd := codexExec(ctx, path, providers.CodexExecArgs(model, lastMsg, profile.tier == tierLite)...)
	cmd.Stdin = strings.NewReader(system + "\n\n" + userMsg)
	cmd.Dir = tempHome

	nodeDir := nodeInstallBinDir()
	baseEnv := os.Environ()
	filteredEnv := make([]string, 0, len(baseEnv)+5)
	for _, e := range baseEnv {
		key, _, _ := strings.Cut(e, "=")
		switch strings.ToUpper(key) {
		case "PATH", "HOME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "CODEX_HOME":
			continue
		}
		filteredEnv = append(filteredEnv, e)
	}
	vol := filepath.VolumeName(tempHome)
	cmd.Env = append(filteredEnv,
		"PATH="+nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+tempHome,
		"USERPROFILE="+tempHome,
		"HOMEDRIVE="+vol,
		"HOMEPATH="+strings.TrimPrefix(tempHome, vol),
		"CODEX_HOME="+codexDir,
	)
	platform.HideWindow(cmd)

	stdoutPipe, stdoutPipeErr := cmd.StdoutPipe()
	stderrPipe, stderrPipeErr := cmd.StderrPipe()

	runtime.EventsEmit(a.ctx, "ai:log", "Starting: "+filepath.Base(cmd.Path))
	if err := cmd.Start(); err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	runtime.EventsEmit(a.ctx, "ai:log", "Process started…")

	var wg sync.WaitGroup
	var stderrBuf strings.Builder
	for _, p := range []struct {
		pipe io.ReadCloser
		err  error
		errs bool
	}{{stdoutPipe, stdoutPipeErr, false}, {stderrPipe, stderrPipeErr, true}} {
		if p.err != nil {
			continue
		}
		wg.Go(func() {
			providers.DrainLines(p.pipe, func(line string) {
				if p.errs {
					stderrBuf.WriteString(line + "\n")
				}
			})
		})
	}

	runErr := cmd.Wait()
	wg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		return types.AIRewriteResult{Error: "cancelled"}
	}
	if runErr != nil {
		return types.AIRewriteResult{Error: providers.CodexFailureMessage(stderrBuf.String(), runErr)}
	}

	result, err := os.ReadFile(lastMsg)
	if err != nil || len(strings.TrimSpace(string(result))) == 0 {
		return types.AIRewriteResult{Error: "Codex produced no output — check that you are signed in (Settings › AI Studio)"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(string(result))}
}
