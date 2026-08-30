# Changelog

All notable changes to Draftline will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),

## Versioning System

This project does NOT use Semantic Versioning (SemVer).

Instead, it uses a Linear Build Versioning system.
Format: MAJOR.MINOR.BUILD

* **MAJOR / MINOR:** Incremented manually when major feature sets or milestones are reached.
* **BUILD:** A monotonic counter that increments by 1 for every single change/commit to the main branch. One fix = one number.

If a package manager or strict SemVer parsing is a hard requirement for your workflow, this project may not be a fit for you.

If this versioning system has a formal name, I am unaware of it, feel free to raise an issue in Github and hit me with a "WeLl AcTuAlLy" if you know the name.


## [0.15.02375] - 2026-08-30

### Fixed
- Fixed the AI-detection "Uniform sentence structure" flag never firing: the condition `sentenceVariety < 30 && sentenceVariety > 70` was always false; it now correctly triggers on `sentenceVariety > 70` (high score = uniform, AI-like structure), matching the sibling flag thresholds

---

## [0.15.02374] - 2026-08-30

### Added
- Integrated the local pure-Go prose/v3 statistical NLP pipeline to distinguish people from places, facilities, and organizations without generative AI or network access
- Added real-prose regression coverage for Chicago street addresses, cross-chapter place/person name collisions, possessives, titled names, and invented science-fiction names

### Changed
- Combined NLP classifications with Draftline's deterministic honorific, possessive, alias, and address rules instead of treating either system as an infallible classifier
- Made non-person evidence and corroboration book-wide while allowing strong person evidence to resolve genuine name/location collisions
- Batched NLP work across a bounded concurrent worker pool so explicit full-book detection remains responsive on large manuscripts

### Fixed
- Prevented `Chicago`, `Hubbard Street`, `Ontario`, `Wells Ave`, `Lower Wacker Drive`, and their component tokens from being promoted into the cast while correctly retaining `Detective Daniel Hanlon`, `Hanlon's`, and `Mara Ionescu`

---

## [0.15.02373] - 2026-08-30

### Changed
- Annotated the Sol 5.6 security audit (`docs/ai-audit/gpt-sol.md`) with a per-finding status scoreboard: findings 3–6 fixed by builds 02351–02366, findings 1–2 (frontend data-loss flows) remain the open release blockers

---

## [0.15.02372] - 2026-08-30

### Added
- Added regression tests for backup rotation and restore, book save/open round-trips, atomic file writes, archive decompression limits, and Node archive extraction safety

---

## [0.15.02371] - 2026-08-30

### Changed
- Tidied Go module metadata; promoted go-keyring to a direct dependency

---

## [0.15.02370] - 2026-08-30

### Fixed
- A transient OS keyring read failure no longer permanently caches an empty API key until restart; the next AI request retries the keyring

---

## [0.15.02369] - 2026-08-30

### Fixed
- Clearing the API key now removes the plaintext fallback copy from settings.json even when the OS keyring is unavailable — previously the one machine that stored the key on disk was the one where clearing silently failed

---

## [0.15.02368] - 2026-08-30

### Changed
- AI Studio determines API configuration from the backend's has-key flag instead of reading a key value from frontend state

---

## [0.15.02367] - 2026-08-30

### Changed
- The settings dialog no longer displays or round-trips the stored API key: the field only accepts a replacement key (submitted through the dedicated keyring binding on save) and gains a Clear button

---

## [0.15.02366] - 2026-08-30

### Security
- Opening .draftline files and importing EPUB/DOCX now enforce archive limits (entry count, per-entry size, total size, compression ratio) with bounded decompression reads, preventing crafted archives from exhausting memory

---

## [0.15.02365] - 2026-08-30

### Fixed
- Backup restores write the manuscript atomically, and a failed pre-restore safety backup is logged instead of silently ignored

---

## [0.15.02364] - 2026-08-30

### Fixed
- Backup copies and metadata are written atomically, and backup-rotation failures are now surfaced in the log instead of swallowed

---

## [0.15.02363] - 2026-08-30

### Fixed
- Book saves are crash-safe: the archive is written to a same-directory temporary file and atomically renamed over the manuscript, and the produced archive is validated (manifest readable) before it replaces the user's file — a disk-full or crash mid-save can no longer truncate the manuscript

---

## [0.15.02362] - 2026-08-30

### Security
- The Claude Code CLI install is pinned to an explicit version instead of latest, keeping the setup supply chain auditable

---

## [0.15.02361] - 2026-08-30

### Security
- Node.js archive extraction caps entry count and per-entry/total decompressed sizes, counted on actual bytes written rather than declared headers

---

## [0.15.02360] - 2026-08-30

### Security
- Archive-supplied file modes are masked during extraction, stripping setuid/setgid/sticky and non-permission bits

---

## [0.15.02359] - 2026-08-30

### Security
- Tar symlinks in the Node.js archive must resolve inside the destination directory and hardlink entries are rejected outright

---

## [0.15.02358] - 2026-08-30

### Security
- Node.js archive entry paths are confined to the destination directory, rejecting absolute, drive-prefixed, and traversal paths (zip-slip)

---

## [0.15.02357] - 2026-08-30

### Security
- The Node.js bootstrap download is verified against pinned per-platform SHA-256 checksums (matching the official nodejs.org SHASUMS256.txt) with a download size cap; unlisted platforms are refused rather than installed unverified

---

## [0.15.02356] - 2026-08-30

### Security
- AI debug log files are created user-only (0600) in a user-only (0700) directory

---

## [0.15.02355] - 2026-08-30

### Security
- AI debug logging — which can include manuscript text and prompts — is now explicitly opt-in via a new setting with a toggle in AI Studio settings, and is off by default

---

## [0.15.02354] - 2026-08-30

### Security
- settings.json is written atomically with user-only (0600) permissions

---

## [0.15.02353] - 2026-08-30

### Fixed
- On machines without an OS keyring, the API key is retained in user-only settings.json instead of being lost, and saving settings no longer wipes the stored key

---

## [0.15.02352] - 2026-08-30

### Security
- The API key never crosses the frontend/backend bridge in either direction: settings expose only a has-key flag, saved settings discard any key field, and new SetAPIKey / HasAPIKey / ClearAPIKey bindings handle the key directly

---

## [0.15.02351] - 2026-08-30

### Security
- AI provider API keys are stored in the OS keyring (Windows Credential Manager / macOS Keychain / Secret Service) instead of plaintext settings.json, with a one-time migration that strips the plaintext copy only after the keyring accepts the key

---

## [0.15.02350] - 2026-08-30

### Added
- Added a JetBrains-inspired Find/Replace bar scoped to the current chapter, opened with `Ctrl+F` or `Ctrl+H`
- Added live match highlighting, current-match navigation, match counts, case-sensitive search, and whole-word search
- Added formatting-safe Replace and single-transaction Replace All operations that can be undone in one step
- Added roadmap designs for character refactoring and AI-assisted story-detail/event refactoring with evidence-based human review

### Changed
- Advanced the feature line to 0.15 for chapter editing and future manuscript-refactoring work
- Defined fixture-based accuracy gates and a stable-ID entity-resolution foundation to replace the current prototype character-detection heuristics before refactoring ships

---

## [0.14.02349] - 2026-08-30

### Changed
- Replaced typo-js's exhaustive edit-distance-two suggestion generator with a prewarmed, length-indexed bounded Damerau-Levenshtein search
- Moved dictionary parsing and suggestion-index construction to application startup in the existing worker, eliminating first-use context-menu initialization

### Fixed
- Reduced representative uncached spelling-suggestion latency from 1.3–3.3 seconds to roughly 130–185 milliseconds while preserving five ranked corrections

---

## [0.14.02348] - 2026-08-30

### Fixed
- Normalized possessive spellings to their lexical root so adding `FleetCom's` or `FleetCom’s` stores `FleetCom`, and all possessive uses are then recognized automatically
- Migrated existing custom-dictionary possessive entries to deduplicated root words

---

## [0.14.02347] - 2026-08-30

### Added
- Offline grammar checking with distinct grammar and style diagnostics, contextual explanations, and one-click corrections
- Persistent spelling diagnostics with asynchronous suggestions and a personal custom dictionary
- A searchable, JetBrains-inspired Plugins settings page for enabling or disabling bundled features
- Feature controls for Spelling, Grammar Check, Cast, Story Bible, Plot Walker, and AI Studio
- A registry-based feature model designed to accommodate community plugins in a future release

### Changed
- Moved expensive spelling suggestions into a dedicated web worker, keeping editor input and context menus responsive
- Routed Claude Code AI work through a dedicated subprocess flow so long-running generation does not block the main application process
- Improved Claude Code discovery on Windows by launching the native executable or Node entry point directly instead of relying on fragile command-shim invocation
- Preserved existing user settings while adding backwards-compatible defaults for newly introduced feature flags
- Normalized backend and frontend version reporting to `0.14.02347`

### Fixed
- Fixed Claude Code integration failures that surfaced as `not recognized as an internal or external command` in the AI sidebar
- Fixed persistent spell-check lag caused by synchronous suggestion generation on the editor thread
- Fixed reactive editor-store subscriptions that caused excessive rerenders, instability, and intermittent crashes during editing and AI review
- Fixed spelling markers disappearing when the custom context menu took focus
- Fixed misspelling suggestions being calculated for the keyboard cursor instead of the word that was right-clicked
- Fixed sidebar width and feature settings being discarded when settings were persisted

### Removed
- Removed legacy trial, Indie, and paid-edition branding from the welcome screen for the open-source release

---

## [0.2.02330]

### Added
- Comprehensive documentation structure in `/docs`
- This changelog for tracking changes going forward

### Changed
- Reorganized documentation into topic-based folders
- Moved legacy docs to `/docs/deprecated`
- **Major refactor**: `ToolsPanel.tsx` split from 2,131 lines to 121 lines (94% reduction)
  - Extracted `tools/constants.ts` - shared configuration
  - Extracted `tools/types.ts` - shared type definitions
  - Extracted `tools/GlyphIcon.tsx` - icon component
  - Extracted `tools/Dashboard/` - word counts, goals, session stats
  - Extracted `tools/AIStudio/` - AI modes, style mixer, streaming
  - Extracted `tools/StoryBible/` - characters, plot notes, timeline
  - Extracted `tools/PlotWalker/` - beats, foreshadowing, knowledge matrix
- **Store refactor**: `bookStore.ts` split into domain-specific stores
  - Extracted `editorStore.ts` - editor ref, selection, inline prompt, diff/review
  - Extracted `storyBibleStore.ts` - characters CRUD, merging, highlighting
  - Extracted `plotStore.ts` - beats, foreshadowing, knowledge matrix
  - `bookStore.ts` now delegates to specialized stores (947 → 713 lines)
- **Dialog refactor**: `AppSettingsDialog.tsx` split from 683 lines to modular components
  - Extracted `settings/constants.ts` - model options, fonts, trim sizes
  - Extracted `settings/types.ts` - TypeScript interfaces
  - Extracted `settings/ApplicationSection.tsx` - identity, theme, save location
  - Extracted `settings/AIStudioSection.tsx` - Claude Code, API keys, local model
  - Extracted `settings/BookSection.tsx` - editor display, export settings

### Fixed
- All Qodana code quality issues resolved (0 remaining)
- Fixed defer-in-loop warnings in import.go
- Fixed unhandled error returns throughout codebase
- Updated Go dependencies to resolve security CVEs:
  - golang.org/x/crypto v0.53.0
  - golang.org/x/net v0.56.0
  - golang.org/x/sys v0.46.0
  - golang.org/x/text v0.38.0
- Updated Vite to resolve npm audit vulnerabilities

### Removed
- Unused constants: `AppVersionMajor`, `AppVersionMinor`, `AppVersionBuild`

---

## [0.12.02325] - 2026-07-08

### Added
- Beat Sheet system for Save the Cat! story structure tracking
- Foreshadowing Ledger for plant/reinforce/payoff tracking
- Knowledge Matrix for tracking who knows what secrets
- Character auto-detection with dialogue pattern recognition
- Attribute extraction (eye color, hair color, age)
- Character highlighting in editor

### Changed
- PlotWalker tools moved to Bible tab in sidebar
- Improved character mention counting

### Fixed
- Various UI polish and bug fixes

---

## [0.11.x] - Previous Releases

### Features Added Over Time
- AI rewriting with multiple providers (Claude Code, Anthropic, OpenAI, Gemini, Grok, Local)
- Style Mixer system (8 dimensions for controlling AI output)
- Prose guide feature for style matching
- EPUB import and export
- DOCX import and export
- PDF export (ebook and print-ready formats)
- Custom title bar with Windows integration
- Auto-save functionality
- Writing goals tracking (daily word count, manuscript progress)
- Session statistics
- AI content detection
- Spell checking
- Dark/light theme with auto-switching
- Chapter management (add, delete, reorder, types)
- Front matter and back matter support
- Story Bible (characters, plot notes, timeline)

---

## Version Format

`MAJOR.MINOR.BUILD`

- **MAJOR**: Significant milestones (1.x, 2.x, 3.x)
- **MINOR**: Feature chunks within major (x.1, x.2, x.3)
- **BUILD**: Always incrementing 5-digit build number (never resets)

Example progression: 0.8.02313 → 0.8.02314 (bug fix) → 0.9.02315 (new feature set)
