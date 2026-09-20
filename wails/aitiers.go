package main

import (
	"strings"
	"sync"
	"time"

	"draftline/internal/ai/providers"
)

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

// A ladder is preferences, not pins. Each tier is tried against what the
// endpoint actually advertises, so a retired id costs nothing: it simply is
// not in the list that comes back, and the next entry is used instead.
type modelLadder struct {
	name      string
	baseURL   string
	anthropic bool
	tiers     map[aiTier][]string
}

var claudeLadder = modelLadder{
	name:      "claude",
	baseURL:   "https://api.anthropic.com/v1",
	anthropic: true,
	tiers: map[aiTier][]string{
		tierLite:   {"claude-haiku-4-5-20251001"},
		tierMedium: {"claude-sonnet-5", "claude-sonnet-4-6"},
		tierHigh:   {"claude-opus-5", "claude-opus-4-6"},
	},
}

var openAILadder = modelLadder{
	name:    "openai",
	baseURL: "https://api.openai.com/v1",
	tiers: map[aiTier][]string{
		tierLite:   {"gpt-5.6-luna", "gpt-5.5", "gpt-4o-mini"},
		tierMedium: {"gpt-5.6-terra", "gpt-5.6-sol", "gpt-5.5", "gpt-4o"},
		tierHigh:   {"gpt-6-astra", "gpt-5.6-sol", "gpt-5.5"},
	},
}

// resolveTierModel picks the best available model for a tier. A model the
// writer set by hand wins above the lite tier, where the point is to keep a
// bulk pass cheap.
func (a *App) resolveTierModel(l modelLadder, apiKey string, profile aiRequestProfile) string {
	prefs := l.tiers[profile.tier]
	if len(prefs) == 0 {
		return ""
	}
	pick := prefs[0]
	if available := a.availableModels(l, apiKey); len(available) > 0 {
		for _, m := range prefs {
			if available[m] {
				pick = m
				break
			}
		}
	}
	if profile.tier == tierLite {
		return pick
	}
	return a.resolveAIModel(pick)
}

// claudeCodeModel prefers a model the writer named, falling back to the tier
// alias so nothing here is pinned to a version.
func (a *App) claudeCodeModel(profile aiRequestProfile) string {
	if m := strings.TrimSpace(a.getSettings().AIModel); m != "" && strings.HasPrefix(strings.ToLower(m), "claude") {
		return m
	}
	return claudeCodeTierModel(profile.tier)
}

// ── model catalogue cache ────────────────────────────────────────────────────

const catalogueTTL = time.Hour

type catalogueEntry struct {
	models  map[string]bool
	fetched time.Time
}

var (
	catalogueMu sync.Mutex
	catalogue   = map[string]catalogueEntry{}
)

// availableModels lists what an endpoint serves, cached so the listing costs
// one request per provider per hour rather than one per rewrite. A failed
// listing is cached too, so an offline machine does not retry on every edit.
func (a *App) availableModels(l modelLadder, apiKey string) map[string]bool {
	catalogueMu.Lock()
	defer catalogueMu.Unlock()
	if e, ok := catalogue[l.name]; ok && time.Since(e.fetched) < catalogueTTL {
		return e.models
	}
	models := providers.ListModels(l.baseURL, apiKey, l.anthropic)
	catalogue[l.name] = catalogueEntry{models: models, fetched: time.Now()}
	return models
}

// seedCatalogue and clearCatalogue let tests drive model choice without a
// network call.
func seedCatalogue(name string, models map[string]bool) {
	catalogueMu.Lock()
	defer catalogueMu.Unlock()
	catalogue[name] = catalogueEntry{models: models, fetched: time.Now()}
}

func clearCatalogue() {
	catalogueMu.Lock()
	defer catalogueMu.Unlock()
	catalogue = map[string]catalogueEntry{}
}
