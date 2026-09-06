# AI Studio Sidebar — Design Gaps & Deferred Work

The AI Studio sidebar (0.16.02418–02419) implements the "AI Studio Sidebar" design. This documents what the design shows but was deliberately not built, for separate work.

## 1. Pass Strength (Light / Standard / Deep) — deferred

The design shows a segmented "Pass Strength" control for Line Edit and Copy Edit with per-mode hints ("Light touches rhythm only where it stumbles" / "Deep restructures sentences aggressively"). The backend has one intensity per mode, so the control is not rendered.

**Implementation sketch when wanted:** add a `strength` param to `RewriteText` (or fold into the style-options JSON), and in `internal/ai/prompt.go` `BuildSystemPrompt` append calibrated guidance per (mode, strength) — e.g. copy_edit Light = "fix only outright errors", Deep = "also enforce consistent house style". Persist the choice in `AppSettings`.

## 2. Single API-key slot limits the provider menu

The design lists every API provider as its own switchable row. The app stores **one** API key (OS keyring, single slot) for whichever provider is currently selected — so the quick-switcher shows one API row (the configured provider) rather than four. Per-provider ready states would be misleading today.

**Fix when wanted:** per-provider keyring entries (`draftline/ai_api_key_<provider>`), migration from the single slot, `HasAPIKey(provider)` binding, and then one menu row per provider with real ready states.

## 3. Task routing is implemented; per-provider model memory is not

Line Edit, Copy Edit, Expand, Smooth, and Custom Prompt each support an explicit provider override and otherwise inherit the default provider. The sidebar menu edits the active task's route; Settings exposes all routes together. Routing is strict and never falls through to another provider.

The app still has one shared optional `ai_model` override (plus the independent local-model field), so it does not remember a different model for every provider. The backend rejects a clearly incompatible shared override and uses that provider's safe default instead. True per-provider model memory still needs separate persisted model fields.

## 4. Kept intentionally different from the design

- The design's own 38px header duplicates the tools panel's `slide-panel-header`; the app keeps the panel header and starts the sidebar at the provider bar.
- The design's glyph rail and status bar are existing app chrome, out of scope.
- The `@ai` in-text tip is real — custom mode has parsed `@ai <instruction>` from the manuscript since early builds.
