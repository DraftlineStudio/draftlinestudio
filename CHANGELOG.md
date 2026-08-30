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
