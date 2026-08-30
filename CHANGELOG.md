# Changelog

All notable changes to Draftline are listed here, newest first.

## [0.14.02349] - 2026-08-30

### Changed
- Spelling suggestions are about ten times faster.

## [0.14.02348] - 2026-08-30

### Fixed
- Custom dictionary entries now cover possessive forms automatically.

## [0.14.02347] - 2026-08-30

### Added
- Added offline grammar and style checking.
- Added spelling suggestions and a custom dictionary.
- Added a Plugins page in Settings for turning features on and off.

### Changed
- Spelling suggestions now run in the background.
- Claude Code work now runs in a separate process.

### Fixed
- Fixed Claude Code not being found on Windows.
- Fixed spell check lag.
- Fixed crashes from excessive re-renders during editing.
- Fixed suggestions appearing for the wrong word.
- Fixed sidebar width and feature settings not saving.

### Removed
- Removed trial and paid edition branding.

## [0.12.02330] - 2026-07-08

### Added
- Added project documentation.

### Changed
- Split the tools panel, book store, and settings dialog into smaller modules.

### Fixed
- Fixed code quality warnings.
- Updated Go and npm dependencies for security fixes.

## [0.12.02325] - 2026-07-08

### Added
- Added a Beat Sheet for story structure.
- Added a Foreshadowing Ledger.
- Added a Knowledge Matrix for tracking secrets.
- Added character auto-detection and highlighting.

### Changed
- Moved PlotWalker tools to the Bible tab.

## [0.11.x] - Earlier releases

### Added
- AI rewriting with Claude Code, Anthropic, OpenAI, Gemini, Grok, and local models.
- Style Mixer for controlling AI output.
- Prose guide for style matching.
- EPUB and DOCX import and export.
- PDF export for ebooks and print.
- Custom title bar.
- Autosave.
- Writing goals and session statistics.
- AI content detection.
- Spell check.
- Light and dark themes with automatic switching.
- Chapter management with front and back matter.
- Story Bible for characters, plot notes, and timeline.

## Versioning

Draftline does not use Semantic Versioning. It uses linear build versioning, `MAJOR.MINOR.BUILD`.

- MAJOR and MINOR go up by hand when a milestone or major feature set lands.
- BUILD goes up by one for every change to the main branch and never resets.

If a package manager or strict SemVer parsing is a hard requirement for your workflow, this project may not be a fit for you.

If this versioning system has a formal name, I am unaware of it, feel free to raise an issue in Github and hit me with a "WeLl AcTuAlLy" if you know the name.
