package providers

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// This file holds the PURE helpers used by the CLI drivers (Claude Code and
// Codex subprocess execution) that remain in package main: argv construction,
// failure-message shaping, model selection, and pipe draining. Nothing here
// executes a process.

// CodexExecArgs builds the constant argv for a non-interactive codex run.
// The prompt is NEVER an argv element — it is piped via stdin ("-"), the same
// injection-safe pattern as the Claude CLI. lastMessageFile receives the final
// assistant message (--output-last-message) so the result needn't be parsed
// out of the progress stream.
func CodexExecArgs(model, lastMessageFile string, lightweight bool) []string {
	args := []string{
		"exec",
		"--skip-git-repo-check", // temp workdir is not a git repo
		"--ignore-user-config",
		"--ignore-rules",
		"--strict-config",
		"--sandbox", "read-only",
		"--ask-for-approval", "never",
		"--config", `shell_environment_policy.inherit="none"`,
		"--ephemeral",
		"--color", "never",
		"--json",
		"--output-last-message", lastMessageFile,
	}
	// Draftline needs text transformation, not an agent. Disable every stable
	// capability-bearing feature exposed by the pinned Codex CLI so manuscript
	// content cannot induce shell, file, browser, app, or sub-agent activity.
	for _, feature := range []string{
		"shell_tool", "unified_exec", "view_image", "apps", "browser_use",
		"computer_use", "image_generation", "multi_agent", "skill_search", "hooks",
	} {
		args = append(args, "--disable", feature)
	}
	if lightweight {
		args = append(args, "--config", `model_reasoning_effort="low"`)
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args, "-")
}

// ClaudeCodeExecArgs builds the argv for a non-interactive Claude Code run
// with every tool and capability disabled.
func ClaudeCodeExecArgs(model string, lightweight bool) []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", "1",
		"--model", model,
		"--safe-mode",
		"--disable-slash-commands",
		"--strict-mcp-config",
		"--mcp-config", `{}`,
		"--tools", "",
		"--permission-mode", "dontAsk",
		"--no-session-persistence",
	}
	if lightweight {
		args = append(args, "--effort", "low")
	}
	return args
}

// ClaudeFailureMessage maps CLI stderr to a user-facing error without ever
// echoing prompt/manuscript-derived content.
func ClaudeFailureMessage(stderr string, runErr error) string {
	lower := strings.ToLower(stderr)
	switch {
	case strings.Contains(lower, "401"), strings.Contains(lower, "403"), strings.Contains(lower, "auth"):
		return "Claude authentication failed. Sign in again from Settings › AI Studio."
	case strings.Contains(lower, "rate limit"), strings.Contains(lower, "429"):
		return "Claude rate limit reached. Wait a moment and try again."
	case strings.Contains(lower, "model") && (strings.Contains(lower, "not found") || strings.Contains(lower, "not supported")):
		return "Claude rejected the selected model. Check the model setting and try again."
	case runErr != nil:
		return "Claude request failed (" + runErr.Error() + "). Check your Claude sign-in and connection."
	default:
		return "Claude request failed. Check your Claude sign-in and connection."
	}
}

// CodexModelOverride prevents the shared legacy ai_model setting from leaking
// a provider-specific model into Codex when the AI Studio quick-switcher is
// used. Unknown values are left alone so future OpenAI model IDs still work.
func CodexModelOverride(model string) string {
	trimmed := strings.TrimSpace(model)
	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"claude", "gemini", "grok", "llama", "mistral"} {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	return trimmed
}

// ResolveCodexLightweightModel uses the CLI's own cached availability list so
// fast editing never hard-fails on installations that predate a model name.
// With no known lightweight model, an empty override safely uses the CLI's
// account default (still at low reasoning effort).
func ResolveCodexLightweightModel(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	if err != nil {
		return ""
	}
	return SelectCodexLightweightModel(data)
}

// SelectCodexLightweightModel picks the preferred fast-tier model advertised
// by the Codex CLI's models cache, or "" when none is available.
func SelectCodexLightweightModel(data []byte) string {
	var cache struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &cache) != nil {
		return ""
	}
	available := make(map[string]bool, len(cache.Models))
	for _, model := range cache.Models {
		available[model.Slug] = true
	}
	for _, preferred := range []string{"gpt-5.6-luna", "gpt-5.4-mini", "gpt-reserve"} {
		if available[preferred] {
			return preferred
		}
	}
	return ""
}

// CodexFailureMessage extracts an actionable error from Codex CLI stderr
// without ever echoing prompt/manuscript-derived content.
func CodexFailureMessage(stderr string, runErr error) string {
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// API failures commonly end in a JSON object containing detail/message.
		if start := strings.Index(line, "{"); start >= 0 {
			var payload map[string]any
			if json.Unmarshal([]byte(line[start:]), &payload) == nil {
				for _, key := range []string{"detail", "message", "error"} {
					if message, ok := payload[key].(string); ok && strings.TrimSpace(message) != "" {
						return "Codex request failed: " + TruncateCodexError(message)
					}
				}
			}
		}

		lower := strings.ToLower(line)
		actionable := strings.HasPrefix(lower, "error:") || strings.HasPrefix(lower, "error ") ||
			strings.Contains(lower, "unexpected status") || strings.Contains(lower, "not supported") ||
			strings.Contains(lower, "invalid model") || strings.Contains(lower, "authentication failed") ||
			strings.Contains(lower, "rate limit")
		if !actionable {
			continue
		}
		if idx := strings.Index(lower, "error:"); idx >= 0 {
			line = strings.TrimSpace(line[idx+len("error:"):])
		}
		return "Codex request failed: " + TruncateCodexError(line)
	}

	if runErr != nil {
		return "Codex request failed (" + runErr.Error() + "). Check your Codex model setting and ChatGPT sign-in."
	}
	return "Codex request failed. Check your Codex model setting and ChatGPT sign-in."
}

// TruncateCodexError caps an error message at a screenshot-friendly length.
func TruncateCodexError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 500 {
		return message[:500] + "…"
	}
	return message
}

// maxDrainLine caps how many bytes of a single output line are retained in
// memory for logging/preview. Longer lines are truncated to this size but the
// remainder is still consumed off the pipe.
const maxDrainLine = 2 * 1024 * 1024

// DrainLines reads r to EOF, invoking onLine once per newline-delimited line
// (with the trailing CR/LF stripped, matching bufio.Scanner.Text semantics).
//
// Unlike bufio.Scanner, it never stops on an over-long line: the retained
// portion of each line is capped at maxDrainLine bytes and the rest of that
// line is consumed and discarded. This guarantees the reader is drained for
// the full lifetime of the pipe so the child process can never block writing
// to a full stdout/stderr pipe (which would otherwise deadlock cmd.Wait()).
func DrainLines(r io.Reader, onLine func(string)) {
	br := bufio.NewReaderSize(r, 64*1024)
	var b strings.Builder
	pending := false // some bytes of the current line have been seen
	for {
		chunk, err := br.ReadString('\n')
		if len(chunk) > 0 {
			pending = true
			// Strip a single trailing LF (and preceding CR) to match Scanner.
			end := len(chunk)
			hasNL := chunk[end-1] == '\n'
			if hasNL {
				end--
				if end > 0 && chunk[end-1] == '\r' {
					end--
				}
			}
			if remaining := maxDrainLine - b.Len(); remaining > 0 {
				if end > remaining {
					b.WriteString(chunk[:remaining])
				} else {
					b.WriteString(chunk[:end])
				}
			}
			if hasNL {
				onLine(b.String())
				b.Reset()
				pending = false
			}
		}
		if err != nil {
			// EOF or read error: flush any trailing line without a newline.
			if pending {
				onLine(b.String())
			}
			return
		}
	}
}
