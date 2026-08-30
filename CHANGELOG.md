# Changelog

All notable changes to Draftline are listed here, newest first.

## [0.15.02380] - 2026-08-30

### Security
- Claude Code prompts are now sent through stdin instead of the command line.
- Hardened the Windows batch launcher against command injection.

## [0.15.02379] - 2026-08-30

### Fixed
- PDF, DOCX, and print exports now decode HTML entities.

## [0.15.02378] - 2026-08-30

### Security
- Auto theme no longer uses location unless you turn it on.
- Location lookups now use HTTPS.

## [0.15.02377] - 2026-08-30

### Security
- The Gemini API key is no longer sent in the URL.

## [0.15.02376] - 2026-08-30

### Security
- Updated nanoid and postcss to fix security advisories.

## [0.15.02375] - 2026-08-30

### Fixed
- Fixed the Uniform Sentence Structure flag in AI detection.

## [0.15.02374] - 2026-08-30

### Added
- Character detection now uses local NLP to tell people apart from places and organizations.

## [0.15.02370] - 2026-08-30

### Fixed
- A temporary keyring error no longer clears the API key until restart.

## [0.15.02369] - 2026-08-30

### Fixed
- Clearing the API key now removes the fallback copy from settings.

## [0.15.02367] - 2026-08-30

### Changed
- Settings no longer displays the stored API key and has a Clear button.

## [0.15.02366] - 2026-08-30

### Security
- Added size limits when opening project files and importing EPUB and DOCX.

## [0.15.02365] - 2026-08-30

### Fixed
- Backup restores are now written safely.

## [0.15.02364] - 2026-08-30

### Fixed
- Backups are now written safely.

## [0.15.02363] - 2026-08-30

### Fixed
- Saves are now crash-safe.

## [0.15.02362] - 2026-08-30

### Security
- Pinned the Claude Code install to a specific version.

## [0.15.02361] - 2026-08-30

### Security
- Limited file sizes when extracting the Node.js runtime.

## [0.15.02360] - 2026-08-30

### Security
- Stripped unsafe file permissions when extracting the Node.js runtime.

## [0.15.02359] - 2026-08-30

### Security
- Blocked unsafe links when extracting the Node.js runtime.

## [0.15.02358] - 2026-08-30

### Security
- Blocked path traversal when extracting the Node.js runtime.

## [0.15.02357] - 2026-08-30

### Security
- The Node.js download is now verified against pinned checksums.

## [0.15.02356] - 2026-08-30

### Security
- AI debug logs are now readable only by the current user.

## [0.15.02355] - 2026-08-30

### Security
- AI debug logging is now off by default.

## [0.15.02354] - 2026-08-30

### Security
- Settings are now saved safely and readable only by the current user.

## [0.15.02353] - 2026-08-30

### Fixed
- The API key is no longer lost on systems without a keyring.

## [0.15.02352] - 2026-08-30

### Security
- The API key no longer passes through the frontend.

## [0.15.02351] - 2026-08-30

### Security
- API keys are now stored in the system keyring.

## [0.15.02350] - 2026-08-30

### Added
- Added chapter Find and Replace with Ctrl+F and Ctrl+H.

### Changed
- Moved to version 0.15.

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
