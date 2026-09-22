package main

// The Claude Code driver.
//
// Two ways in, chosen by what the CLI left behind. A credentials file holding
// a direct API key means Draftline can call the Anthropic API itself and
// stream the reply; OAuth credentials mean only the CLI can, so the request
// goes out as a subprocess. The first is faster and does not depend on a
// subprocess staying alive, which is why it is tried first.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"draftline/internal/ai/providers"
	"draftline/internal/logging"
	"draftline/internal/platform"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// callClaudeCode handles the "claudecode" AI mode. If the credentials file
// contains a direct API key it calls the Anthropic API with streaming directly —
// faster and more reliable than the CLI subprocess. Falls back to the CLI when
// only OAuth credentials are present.
func (a *App) callClaudeCode(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	if apiKey := readClaudeAPIKey(); apiKey != "" {
		model := a.resolveTierModel(claudeLadder, apiKey, profile)
		runtime.EventsEmit(a.ctx, "ai:log", "Using the "+profile.tier.label()+" model: "+model)
		runtime.EventsEmit(a.ctx, "ai:log", "Connecting to Anthropic API…")
		result, err := providers.StreamAnthropic(providers.Request{
			Ctx: ctx, System: system, UserMsg: userMsg,
			Model: model, APIKey: apiKey, Emit: a.aiEmit,
		})
		if err != nil {
			return types.AIRewriteResult{Error: err.Error()}
		}
		return types.AIRewriteResult{Result: result}
	}
	return a.callClaudeCodeCLI(ctx, system, userMsg, profile)
}

// callClaudeCodeCLI invokes the Claude Code CLI as a subprocess.
// Used as a fallback when only OAuth credentials are available (no raw API key).
func (a *App) callClaudeCodeCLI(ctx context.Context, system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	path := resolveClaudeBin()
	if path == "" {
		return types.AIRewriteResult{Error: "Claude Code is not installed — open Settings › AI Studio to set it up"}
	}
	model := a.claudeCodeModel(profile)
	runtime.EventsEmit(a.ctx, "ai:log", "Using the "+profile.tier.label()+" model: "+model)

	// Create an isolated home directory: real credentials so auth works, but no
	// MCP server config. MCP servers are started between init and the first API
	// call — if any hang, the subprocess hangs silently for the full timeout.
	tempHome, err := os.MkdirTemp("", "draftline-claude-*")
	if err != nil {
		return types.AIRewriteResult{Error: "cannot create temp dir: " + err.Error()}
	}
	defer func() { _ = os.RemoveAll(tempHome) }()

	realHome, _ := os.UserHomeDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	if data, e2 := os.ReadFile(filepath.Join(realHome, ".claude", ".credentials.json")); e2 == nil {
		_ = os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), data, 0600)
	}
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"mcpServers":{}}`), 0644)

	// The prompt is piped via stdin rather than passed as an argv element: in
	// print mode (-p) the CLI reads piped stdin as the prompt. This keeps
	// manuscript-derived content out of every exec path (notably the cmd.exe
	// .cmd-shim fallback, where argv metacharacters would be interpreted) and
	// sidesteps the ~32K Windows command-line length limit.
	fullPrompt := system + "\n\n" + userMsg
	claudeArgs := providers.ClaudeCodeExecArgs(model, profile.tier == tierLite)
	cmd := claudeExec(ctx, path, claudeArgs...)
	cmd.Stdin = strings.NewReader(fullPrompt)
	prepareClaudeRequestCommand(cmd, tempHome)

	stdoutPipe, stdoutPipeErr := cmd.StdoutPipe()
	stderrPipe, stderrPipeErr := cmd.StderrPipe()

	runtime.EventsEmit(a.ctx, "ai:log", "Starting: "+filepath.Base(cmd.Path))
	if err := cmd.Start(); err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	runtime.EventsEmit(a.ctx, "ai:log", "Process started…")

	var wg sync.WaitGroup
	var resultText string
	var stderrBuf strings.Builder

	if stdoutPipeErr == nil {
		wg.Go(func() {
			streamed := false
			providers.DrainLines(stdoutPipe, func(line string) {
				// Raw stream-json lines contain prompt/manuscript-derived
				// content. Keep them only in the opt-in debug log; never emit
				// them to the ai:log runtime event (which can appear in
				// screenshots). Parity with the Codex path.
				logging.AIContent("CLAUDE_STREAM", line)
				var obj map[string]any
				if json.Unmarshal([]byte(line), &obj) == nil {
					switch obj["type"] {
					case "assistant", "content_block_delta", "message_delta":
						if !streamed {
							streamed = true
							runtime.EventsEmit(a.ctx, "ai:log", "Streaming…")
						}
					case "result":
						if r, ok := obj["result"].(string); ok {
							resultText = r
						}
						if isErr, _ := obj["is_error"].(bool); isErr {
							if msg, ok := obj["result"].(string); ok {
								stderrBuf.WriteString(msg)
							}
							runtime.EventsEmit(a.ctx, "ai:log", "Error")
						} else {
							runtime.EventsEmit(a.ctx, "ai:log", "Done")
						}
					}
				}
			})
		})
	}
	if stderrPipeErr == nil {
		wg.Go(func() {
			providers.DrainLines(stderrPipe, func(line string) {
				stderrBuf.WriteString(line + "\n")
				// stderr may echo prompt/manuscript-derived content; keep it in
				// the opt-in debug log only, not the screenshot-visible ai:log.
				logging.AIContent("CLAUDE_STDERR", line)
			})
		})
	}

	runErr := cmd.Wait()
	wg.Wait()

	if runErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return types.AIRewriteResult{Error: "timed out — try a shorter chapter or check your Claude Code connection"}
		}
		return types.AIRewriteResult{Error: providers.ClaudeFailureMessage(stderrBuf.String(), runErr)}
	}
	if strings.TrimSpace(resultText) == "" {
		return types.AIRewriteResult{Error: "Claude produced no output — check that you are signed in (Settings › AI Studio)"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(resultText)}
}

// CheckClaudeCode checks whether the claude CLI is installed and authenticated.
// It checks both the system PATH and Draftline's bundled install location.
func (a *App) CheckClaudeCode() types.ClaudeCodeStatus {
	npmAvailable := resolveNpmBin() != ""

	claudePath := resolveClaudeBin()
	if claudePath == "" {
		return types.ClaudeCodeStatus{NpmAvailable: npmAvailable}
	}
	out, err := claudeExec(context.Background(), claudePath, "--version").Output()
	version := ""
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	home, _ := os.UserHomeDir()
	credPaths := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".config", "claude", ".credentials.json"),
	}
	authenticated := false
	for _, p := range credPaths {
		if _, err := os.Stat(p); err == nil {
			authenticated = true
			break
		}
	}
	return types.ClaudeCodeStatus{Installed: true, Authenticated: authenticated, NpmAvailable: npmAvailable, Version: version}
}

// OpenClaudeAuth runs "claude auth login" as a hidden background process.
// It opens the user's browser to complete the OAuth flow. When the process
// exits (auth complete), it emits a "claude:auth_complete" event to the frontend.
func (a *App) OpenClaudeAuth() {
	claudePath := resolveClaudeBin()
	if claudePath == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := claudeExec(ctx, claudePath, "auth", "login")
		platform.HideWindow(cmd)
		_ = cmd.Start()
		_ = cmd.Wait()
		runtime.EventsEmit(a.ctx, "claude:auth_complete", nil)
	}()
}

// readClaudeAPIKey reads the Anthropic API key stored by the Claude Code CLI.
// Returns empty string if not found or if using OAuth (no direct API key).
func readClaudeAPIKey() string {
	home, _ := os.UserHomeDir()
	credPaths := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".config", "claude", ".credentials.json"),
	}
	for _, p := range credPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var creds struct {
			APIKey string `json:"apiKey"`
		}
		if json.Unmarshal(data, &creds) == nil && creds.APIKey != "" {
			return creds.APIKey
		}
	}
	return ""
}
