package main

import "strings"

// A tier sizes the model to the work rather than naming a version. Providers
// retire model IDs constantly; a tier outlives them.
type aiTier string

const (
	tierLite   aiTier = "lite"
	tierMedium aiTier = "medium"
	tierHigh   aiTier = "high"
)

type aiRequestProfile struct {
	tier aiTier
}

// standardAIRequest covers free-form prompts, where the ask is unknown.
var standardAIRequest = aiRequestProfile{tier: tierHigh}

// rewriteRequestProfile: a copy pass is mechanical, a line edit is prose
// judgement, expanding is writing.
func rewriteRequestProfile(mode string) aiRequestProfile {
	switch mode {
	case "copy_edit":
		return aiRequestProfile{tier: tierLite}
	case "expand":
		return aiRequestProfile{tier: tierHigh}
	default: // line_edit, smooth
		return aiRequestProfile{tier: tierMedium}
	}
}

func (t aiTier) label() string {
	switch t {
	case tierLite:
		return "fast"
	case tierHigh:
		return "most capable"
	default:
		return "standard"
	}
}

// claudeCodeTierModel uses the CLI's own tier aliases, which it resolves to
// whatever it currently ships. A retired snapshot cannot strand an install.
func claudeCodeTierModel(t aiTier) string {
	switch t {
	case tierLite:
		return "haiku"
	case tierHigh:
		return "opus"
	default:
		return "sonnet"
	}
}

// The direct APIs take no tier aliases, so these are the one place a version
// is named. Everything else resolves a tier.
var claudeAPITierModels = map[aiTier]string{
	tierLite:   "claude-haiku-4-5-20251001",
	tierMedium: "claude-sonnet-4-6",
	tierHigh:   "claude-opus-4-6",
}

var openAITierModels = map[aiTier]string{
	tierLite:   "gpt-4o-mini",
	tierMedium: "gpt-4o",
	tierHigh:   "o3",
}

// resolveTierModel picks the model for a tier. A model the writer set by hand
// wins above the lite tier, where the point is to keep a bulk pass cheap.
func (a *App) resolveTierModel(models map[aiTier]string, profile aiRequestProfile) string {
	def := models[profile.tier]
	if profile.tier == tierLite {
		return def
	}
	return a.resolveAIModel(def)
}

// claudeCodeModel prefers a model the writer named, falling back to the tier
// alias so nothing here is pinned to a version.
func (a *App) claudeCodeModel(profile aiRequestProfile) string {
	if m := strings.TrimSpace(a.getSettings().AIModel); m != "" && strings.HasPrefix(strings.ToLower(m), "claude") {
		return m
	}
	return claudeCodeTierModel(profile.tier)
}
