package main

import (
	"strings"

	"draftline/internal/ai/providers"
	"draftline/internal/types"
)

// dispatchAI routes requests that do not have a task-specific override through
// the configured default provider.
func (a *App) dispatchAI(system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	return a.dispatchAIWithMode(system, userMsg, profile, "")
}

// dispatchAIWithMode routes one request through an explicit task provider.
// Empty or invalid modes inherit the user's default. There is deliberately no
// fallback chain: a task assigned to Codex either uses Codex or reports that
// Codex needs setup instead of silently sending manuscript text elsewhere.
func (a *App) dispatchAIWithMode(system, userMsg string, profile aiRequestProfile, providerMode string) types.AIRewriteResult {
	ctx, release, ok := a.acquireAI()
	if !ok {
		return types.AIRewriteResult{Error: "an AI request is already in progress — wait for it to finish or cancel it"}
	}
	defer release()

	s := a.getSettings()
	providerMode = normalizeAIProviderMode(providerMode, s.AIMode)
	switch providerMode {
	case "claudecode":
		return a.callClaudeCode(ctx, system, userMsg, profile)
	case "codex":
		return a.callCodexCLI(ctx, system, userMsg, profile)
	case "local":
		if s.AILocalEndpoint == "" || s.AILocalModel == "" {
			return types.AIRewriteResult{Error: "no local model configured — set the endpoint and model in App Settings"}
		}
		return providers.Local(providers.Request{
			Ctx: ctx, System: system, UserMsg: userMsg, Settings: s, Emit: a.aiEmit,
		})
	default: // "api"
		if s.AIProvider == "" || a.getAPIKey() == "" {
			return types.AIRewriteResult{Error: "no API provider configured — add your key in App Settings"}
		}
		req := providers.Request{
			Ctx: ctx, System: system, UserMsg: userMsg,
			APIKey: a.getAPIKey(), Settings: s, Emit: a.aiEmit,
		}
		switch s.AIProvider {
		case "claude":
			req.Model = a.resolveRequestModel("claude-sonnet-4-6", "claude-haiku-4-5-20251001", profile)
			return providers.Claude(req)
		case "openai":
			req.Model = a.resolveRequestModel("gpt-4o", "gpt-4o-mini", profile)
			return providers.OpenAI(req)
		case "gemini":
			req.Model = a.resolveRequestModel("gemini-1.5-pro", "gemini-1.5-flash", profile)
			return providers.Gemini(req)
		case "grok":
			req.Model = a.resolveRequestModel("grok-2", "", profile)
			return providers.Grok(req)
		default:
			return types.AIRewriteResult{Error: "unknown provider: " + s.AIProvider}
		}
	}
}

func normalizeAIProviderMode(requested, fallback string) string {
	switch requested {
	case "claudecode", "codex", "api", "local":
		return requested
	}
	switch fallback {
	case "claudecode", "codex", "api", "local":
		return fallback
	default:
		return "claudecode"
	}
}

// modelMatchesProvider prevents the legacy shared model override from leaking
// across task routes. Each non-default route safely uses its provider default
// when the configured model clearly belongs to another provider.
func modelMatchesProvider(model, defaultModel string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	defaultModel = strings.ToLower(defaultModel)
	switch {
	case strings.HasPrefix(defaultModel, "claude"):
		return strings.HasPrefix(model, "claude")
	case strings.HasPrefix(defaultModel, "gemini"):
		return strings.HasPrefix(model, "gemini")
	case strings.HasPrefix(defaultModel, "grok"):
		return strings.HasPrefix(model, "grok")
	default: // OpenAI-compatible API defaults.
		return strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4")
	}
}
