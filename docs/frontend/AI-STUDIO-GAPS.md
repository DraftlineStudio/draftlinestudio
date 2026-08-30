# AI Studio Sidebar — Design Gaps & Deferred Work

The AI Studio sidebar (0.16.02418–02419) implements the "AI Studio Sidebar" design from `reference-assets/AI sidebar redesign overhaul/`. This documents what the design shows but was deliberately not built, for separate work.

## 1. Pass Strength (Light / Standard / Deep) — deferred by user decision

The design shows a segmented "Pass Strength" control for Line Edit and Copy Edit with per-mode hints ("Light touches rhythm only where it stumbles" / "Deep restructures sentences aggressively"). The backend has one intensity per mode, so the control is not rendered.

**Implementation sketch when wanted:** add a `strength` param to `RewriteText` (or fold into the style-options JSON), and in `internal/ai/prompt.go` `BuildSystemPrompt` append calibrated guidance per (mode, strength) — e.g. copy_edit Light = "fix only outright errors", Deep = "also enforce consistent house style". Persist the choice in `AppSettings`. The design's hint copy is in `AI Studio Sidebar.dc.html` (`HINTS` map).

## 2. Single API-key slot limits the provider menu

The design lists every API provider as its own switchable row. The app stores **one** API key (OS keyring, single slot) for whichever provider is currently selected — so the quick-switcher shows one API row (the configured provider) rather than four. Per-provider ready states would be misleading today.

**Fix when wanted:** per-provider keyring entries (`draftline/ai_api_key_<provider>`), migration from the single slot, `HasAPIKey(provider)` binding, and then one menu row per provider with real ready states.

## 3. Model names in the menu are display-only

The design shows a model name per provider row (e.g. "Sonnet 4.6", "GPT-5.2"). The app has a single shared `ai_model` setting (plus CLI defaults when blank), so the menu displays a model only for the active route and an account label for inactive routes — it is not a per-provider model picker. The quick-switcher clears incompatible provider-specific values, and the backend independently prevents non-Codex model IDs from reaching the Codex CLI. A per-route model memory would still need per-route settings fields.

## 4. Kept intentionally different from the design

- The design's own 38px header duplicates the tools panel's `slide-panel-header`; the app keeps the panel header and starts the sidebar at the provider bar.
- The design's glyph rail and status bar are existing app chrome, out of scope.
- The `@ai` in-text tip is real — custom mode has parsed `@ai <instruction>` from the manuscript since early builds.
