# Changelog

All notable changes to Draftline are listed here, newest first.

## [0.16.02482] - 2026-09-01

### Added
- Registered .storiverse files with their own icon.

## [0.16.02481] - 2026-09-01

### Changed
- Added a dedicated document icon for .draftline files.

## [0.16.02480] - 2026-09-01

### Fixed
- .draftline files now show the Draftline icon in Explorer right away.

## [0.16.02479] - 2026-09-01

### Fixed
- Opening a file while Draftline is already running now opens that file.

## [0.16.02478] - 2026-09-01

### Added
- Draftline now registers file associations for .draftline, .epub, and .docx on Windows.

## [0.16.02477] - 2026-09-01

### Added
- Files opened from the operating system now open or import in Draftline.

### Fixed
- Opening a recent project with unsaved changes now opens that project after Save or Discard.

## [0.16.02476] - 2026-09-01

### Added
- Added a Worth Reviewing queue to the Evidence Archive.
- Evidence can be annotated, pinned, and marked reviewed.

## [0.16.02475] - 2026-09-01

### Added
- Story Search now shows source-backed detail trails for characters, places, and objects.

## [0.16.02474] - 2026-09-01

### Fixed
- The Plot and Prose status chips now track their own analysis phases.

## [0.16.02473] - 2026-09-01

### Changed
- Simplified the status bar to Characters, Plot, and Prose chips.
- Added a Character Map button to the status bar.

### Removed
- Removed the AI detection meter and word count from the status bar.

## [0.16.02470] - 2026-09-01

### Added
- Added a local fact and event index built during analysis.
- Added an Evidence Index tab to Story Search.

### Changed
- Evidence analysis is about three times faster.

## [0.16.02469] - 2026-08-31

### Added
- Added Story Search, a manuscript search window that understands character aliases.

## [0.16.02463] - 2026-08-31

### Changed
- Removed readability grade estimates from Signals and moved readability formulas into Prose details.

## [0.16.02462] - 2026-08-31

### Fixed
- Imported chapters no longer take the book title as their name.
- Untitled front and back matter sections now use their section name.

## [0.16.02461] - 2026-08-31

### Added
- EPUB imports now sort dedications, prologues, epilogues, and similar sections into front and back matter.
- Imported copyright pages fill the Copyright section.
- The import preview now shows warnings and detected sections.

## [0.16.02460] - 2026-08-31

### Fixed
- EPUB imports no longer split chapters too aggressively.
- Chapter headings become chapter titles instead of duplicating in the text.

### Changed
- Added size limits for very large imports.

## [0.16.02459] - 2026-08-31

### Fixed
- Replaced the EPUB importer's HTML handling with a real parser.
- Poetry and preformatted text keep their line breaks on import.
- EPUBs in UTF-16, Latin-1, and Windows-1252 now import correctly.

### Changed
- Imports now report how many images were removed.

## [0.16.02458] - 2026-08-31

### Fixed
- A bad EPUB or DOCX file can no longer crash Draftline.
- A rendering error in the editor now shows a retry screen instead of a blank window.

### Changed
- EPUB imports skip covers, navigation documents, and very large files.

## [0.16.02457] - 2026-08-31

### Changed
- Redesigned Pacing with a tempo score, labeled bands, and chapter-to-chapter changes.

### Fixed
- Author's notes are no longer included in story analysis.

## [0.16.02456] - 2026-08-31

### Changed
- Redesigned the Writing Dashboard around manuscript, daily, and session progress.

### Added
- Added a 7-day writing streak.

## [0.16.02455] - 2026-08-31

### Added
- Added the AI Analysis panel with a chapter score, whole-book scan, and flagged passages.

## [0.16.02454] - 2026-08-31

### Added
- Added the Signals panel with manuscript stats and changes since the last run.

## [0.16.02453] - 2026-08-31

### Added
- Added the Worth Reviewing panel with dismissable observations.

## [0.16.02452] - 2026-08-31

### Added
- Added the Chapters panel with keywords and summaries for each chapter.

## [0.16.02451] - 2026-08-31

### Added
- Added the Pacing panel with a whole-book tempo strip and per-chapter bars.

## [0.16.02450] - 2026-08-31

### Added
- Added the Prose panel with sentence rhythm, sentence length, and readability.

## [0.16.02449] - 2026-08-31

### Removed
- Removed the Plot Notes, Timeline, Beat Sheet, Foreshadowing Ledger, and Knowledge Matrix sidebars.

## [0.16.02448] - 2026-08-31

### Added
- Added local story analysis that runs while you are idle.
- Added analysis progress and status to the status bar.

## [0.16.02447] - 2026-08-31

### Added
- Added chapter version history inside the project file.
- Added File > Chapter History with comparison and restore.

## [0.16.02446] - 2026-08-31

### Security
- Claude Code and Codex now run as text-only processes.
- Draftline now only runs its own pinned Claude Code and Codex installs.

## [0.16.02445] - 2026-08-31

### Added
- Licensed Draftline under the MIT License.
- Added a project README.

## [0.16.02442] - 2026-08-31

### Changed
- Typing no longer recounts the whole book's words on every keystroke.

## [0.16.02440] - 2026-08-31

### Fixed
- Fixed invalid print-ready PDF files.

## [0.16.02439] - 2026-08-31

### Fixed
- Applying an AI diff no longer removes formatting from the chapter.
- Inserting or deleting a paragraph no longer marks every later paragraph as changed.

## [0.16.02438] - 2026-08-31

### Fixed
- Re-indexing a book now clears outdated relationship data.

## [0.16.02437] - 2026-08-31

### Fixed
- Dialogue detection now works with curly quotes.

## [0.16.02436] - 2026-08-30

### Fixed
- AI requests no longer hang when a CLI prints a very long line.

## [0.16.02435] - 2026-08-30

### Security
- Limited the number of chapter references a project file can declare.

## [0.16.02434] - 2026-08-30

### Fixed
- Chapters that fail to load now show an error instead of opening empty.

## [0.16.02433] - 2026-08-30

### Fixed
- Rapid settings changes no longer overwrite each other.

## [0.16.02432] - 2026-08-30

### Fixed
- Fixed race conditions in shared backend state.

## [0.16.02431] - 2026-08-30

### Security
- Limited AI provider response size.

## [0.16.02430] - 2026-08-30

### Security
- The AI activity log no longer shows manuscript text from Claude Code.

## [0.16.02429] - 2026-08-30

### Security
- Backups and the recent projects list are now readable only by the current user.

## [0.16.02427] - 2026-08-30

### Fixed
- Errors from background tasks are no longer silently ignored.

## [0.16.02423] - 2026-08-30

### Added
- The tools sidebar remembers whether it was open and which pane was active.

## [0.16.02422] - 2026-08-30

### Changed
- Line Edit now only changes sentences with a clear problem.
- Copy Edit now only fixes spelling, grammar, and punctuation.
- Line Edit and Copy Edit use faster models.

## [0.16.02421] - 2026-08-30

### Fixed
- Switching to Codex no longer passes a model name from another provider.
- Codex errors now show a short message instead of raw output.

## [0.16.02419] - 2026-08-30

### Changed
- Redesigned the AI Studio sidebar with a provider switcher and mode cards.

## [0.16.02418] - 2026-08-30

### Added
- Added a Copy Edit AI mode that fixes errors without changing style.

## [0.16.02416] - 2026-08-30

### Fixed
- Contractions with curly apostrophes are no longer marked as misspelled.
- Character names are no longer marked as misspelled.

## [0.16.02415] - 2026-08-30

### Added
- Added Codex setup and ChatGPT sign-in to Settings.

## [0.16.02414] - 2026-08-30

### Added
- Added OpenAI Codex as an AI provider for ChatGPT accounts.

## [0.16.02413] - 2026-08-30

### Changed
- Redesigned the character sidebar to show characters in the current chapter first.

## [0.16.02412] - 2026-08-30

### Fixed
- Aligned the tools sidebar header with the rest of the toolbar.

## [0.16.02411] - 2026-08-30

### Fixed
- The character sidebar now only shows confirmed characters.
- Relationships with unconfirmed characters are now hidden.

## [0.16.02410] - 2026-08-30

### Changed
- Replaced character merge mode with a searchable Merge Character dialog.

## [0.16.02409] - 2026-08-30

### Added
- Rejected character candidates stay rejected after re-detection.

## [0.16.02408] - 2026-08-30

### Added
- Added a Needs Review view for uncertain character candidates.

## [0.16.02407] - 2026-08-30

### Fixed
- Character detection no longer merges different characters who share a word in their names.

## [0.16.02406] - 2026-08-30

### Fixed
- Character analysis now ignores headings, contents, copyright pages, and other non-story sections.

## [0.16.02405] - 2026-08-30

### Fixed
- Character analysis, relationships, and corrections are now saved with the book.
- Re-detecting characters keeps roles, descriptions, and notes.

## [0.16.02404] - 2026-08-30

### Fixed
- Saving an imported book no longer overwrites the previously open project.

## [0.16.02403] - 2026-08-30

### Fixed
- Fixed button alignment in the Characters view.

## [0.16.02402] - 2026-08-30

### Fixed
- The Characters view now fills the workspace.

## [0.16.02401] - 2026-08-30

### Added
- The Characters view remembers Grid or Heatmap.

## [0.16.02400] - 2026-08-30

### Changed
- Heatmap view now sorts by mentions.

## [0.16.02399] - 2026-08-30

### Changed
- The tools sidebar hides while the Characters view is open.

## [0.16.02398] - 2026-08-30

### Fixed
- Long names and text no longer overflow in the Characters view.

## [0.16.02396] - 2026-08-30

### Added
- Added role selection to each row in Heatmap view.

## [0.16.02395] - 2026-08-30

### Added
- Added a relationship map, chapter presence strip, and mention excerpts to the character detail pane.

## [0.16.02394] - 2026-08-30

### Changed
- Replaced the character relationship graph with a chapter grid and heatmap.

## [0.16.02393] - 2026-08-30

### Changed
- Moved to version 0.16 and renamed Cast to Characters.

## [0.15.02392] - 2026-08-30

### Fixed
- Typing no longer re-renders the whole application.

## [0.15.02391] - 2026-08-30

### Fixed
- Cleared AI diffs no longer reappear.

## [0.15.02390] - 2026-08-30

### Fixed
- An autosave from a closed project can no longer affect the next project.

## [0.15.02387] - 2026-08-30

### Changed
- Reduced unnecessary re-renders in the editor and status bar.

## [0.15.02386] - 2026-08-30

### Changed
- Reduced the main bundle size.

## [0.15.02384] - 2026-08-30

### Fixed
- A failed or cancelled save no longer discards unsaved changes when opening or creating a book.
- Closing a project now stops if the save fails.

## [0.15.02383] - 2026-08-30

### Fixed
- Edits made during an autosave are no longer marked as saved.
- Saves now run one at a time.

## [0.15.02382] - 2026-08-30

### Fixed
- The inline prompt can now be cancelled during generation.

## [0.15.02381] - 2026-08-30

### Fixed
- Only one AI request can run at a time.
- Cancel now works for Local, OpenAI, Gemini, and Grok.

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
