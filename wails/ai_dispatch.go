package main

import (
	"strings"

	"draftline/internal/ai/providers"
	"draftline/internal/types"
)

// dispatchAI routes a request through the configured default provider.
func (a *App) dispatchAI(system, userMsg string, profile aiRequestProfile) types.AIRewriteResult {
	return a.dispatchAIWithMode(system, userMsg, profile, "")
}

// dispatchAIWithMode routes one request through an explicit task provider.
// There is deliberately no fallback chain: a task assigned to Codex either uses
// Codex or reports that Codex needs setup, rather than silently sending
// manuscript text somewhere else.
func (a *App) dispatchAIWithMode(system, userMsg string, profile aiRequestProfile, providerMode string) types.AIRewriteResult {
	ctx, release, ok := a.acquireAI()
	if !ok {
		return types.AIRewriteResult{Error: "an AI request is already in progress — wait for it to finish or cancel it"}
	}
	defer release()

	s := a.getSettings()
	providerMode = normalizeAIProviderMode(providerMode, s.AIMode, s.AIProviders)
	switch providerMode {
	case "claudecode":
		return a.callClaudeCode(ctx, system, userMsg, profile)
	case "codex":
		return a.callCodexCLI(ctx, system, userMsg, profile)
	case "api":
		if s.AIProvider == "" || a.getAPIKey() == "" {
			return types.AIRewriteResult{Error: "no API provider configured — add your key in App Settings"}
		}
		req := providers.Request{
			Ctx: ctx, System: system, UserMsg: userMsg,
			APIKey: a.getAPIKey(), Settings: s, Emit: a.aiEmit,
		}
		switch s.AIProvider {
		case "claude":
			req.Model = a.resolveTierModel(claudeAPITierModels, profile)
			return providers.Claude(req)
		case "openai":
			req.Model = a.resolveTierModel(openAITierModels, profile)
			return providers.OpenAI(req)
		default:
			return types.AIRewriteResult{Error: "unknown provider: " + s.AIProvider}
		}
	default:
		p, found := findAIProvider(s.AIProviders, providerMode)
		if !found {
			return types.AIRewriteResult{Error: "that AI provider is no longer configured — check App Settings"}
		}
		if strings.TrimSpace(p.BaseURL) == "" || strings.TrimSpace(p.Model) == "" {
			return types.AIRewriteResult{Error: providerName(p) + " needs an endpoint and a model — set them in App Settings"}
		}
		if p.Kind == "cloud" && a.providerKey(p) == "" {
			return types.AIRewriteResult{Error: providerName(p) + " needs an API key — add it in App Settings"}
		}
		return providers.Custom(providers.Request{
			Ctx: ctx, System: system, UserMsg: userMsg,
			APIKey: a.providerKey(p), Settings: s, Emit: a.aiEmit,
		}, p)
	}
}

// normalizeAIProviderMode resolves a requested mode, falling back to the
// default and then to Claude Code when neither names anything configured.
func normalizeAIProviderMode(requested, fallback string, list []types.AIProvider) string {
	if m := validAIMode(requested, list); m != "" {
		return m
	}
	if m := validAIMode(fallback, list); m != "" {
		return m
	}
	return "claudecode"
}

func validAIMode(mode string, list []types.AIProvider) string {
	switch mode {
	case "claudecode", "codex", "api":
		return mode
	}
	if _, ok := findAIProvider(list, mode); ok {
		return mode
	}
	return ""
}

func findAIProvider(list []types.AIProvider, id string) (types.AIProvider, bool) {
	if strings.TrimSpace(id) == "" {
		return types.AIProvider{}, false
	}
	for _, p := range list {
		if p.ID == id {
			return p, true
		}
	}
	return types.AIProvider{}, false
}

func providerName(p types.AIProvider) string {
	if n := strings.TrimSpace(p.Nickname); n != "" {
		return n
	}
	return "That provider"
}

// modelMatchesProvider keeps the shared model override from leaking across
// task routes; a mismatched model falls back to the route's own default.
func modelMatchesProvider(model, defaultModel string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	defaultModel = strings.ToLower(defaultModel)
	switch {
	case strings.HasPrefix(defaultModel, "claude"):
		return strings.HasPrefix(model, "claude")
	default: // OpenAI-compatible API defaults.
		return strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4")
	}
}
