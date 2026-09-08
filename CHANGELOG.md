# Changelog

All notable changes to Draftline are listed here, newest first.

## [0.17.02572] - 2026-09-07

### Changed
- Rebuilt Export Book as a four-step edition builder.
- Split print design into Page, Typography, and Book Furniture sections.
- Print interiors now default to exact trim with bleed and crop marks off.

## [0.17.02571] - 2026-09-07

### Changed
- The update indicator is now a green Update Available chip.
- Draftline now closes after launching a downloaded installer.

## [0.17.02570] - 2026-09-07

### Changed
- Reordered the title bar controls and added the author chip to the welcome screen.
- The recent projects list now scrolls inside its card.

## [0.17.02569] - 2026-09-07

### Added
- Draftline now checks for updates hourly and shows an indicator in the title bar.

### Fixed
- Fresh installs now open the right sidebar at its minimum width.

## [0.17.02568] - 2026-09-07

### Added
- Books can now store one ISBN per format.

## [0.17.02567] - 2026-09-07

### Changed
- The author chip now opens an identity card with a link to Settings.
- The right sidebar opens at its minimum width by default.

## [0.17.02566] - 2026-09-07

### Added
- Added a theme toggle to the title bar.

### Changed
- Added a Settings button to the title bar on every screen.

## [0.17.02565] - 2026-09-07

### Fixed
- Claude Code edits no longer open a blank terminal window on Windows.

## [0.17.02564] - 2026-09-07

### Added
- Added a manual update checker with verified downloads in Settings.

## [0.17.02563] - 2026-09-07

### Fixed
- Codex sign-in is now detected when credentials are stored in the system keychain.

## [0.17.02562] - 2026-09-07

### Changed
- Upgraded the spelling dictionary to SCOWL size 70.

### Fixed
- Spell check now handles accented words correctly.

## [0.17.02543] - 2026-09-05

### Fixed
- Selected-text AI edits no longer leave empty paragraphs at the edges of the selection.
- Fixed Claude Code requests failing with an invalid MCP configuration.

### Removed
- Removed the duplicate Read Aloud button from the sidebar rail.

## [0.17.02542] - 2026-09-05

### Changed
- Selected-text AI reviews now open in a dialog over the manuscript.

## [0.17.02541] - 2026-09-05

### Fixed
- Fixed applying a selected-text AI edit deleting the rest of the chapter.
- Applied AI changes can now be undone with Ctrl+Z.
- Chapter history now records a version before and after each applied AI pass.

## [0.17.02540] - 2026-09-04

### Changed
- Renamed the AI diff buttons to Reject Change and Accept Change.

## [0.17.02539] - 2026-09-04

### Fixed
- Fixed every Codex request failing on an invalid argument.

## [0.17.02536] - 2026-09-04

### Added
- AI Studio can now route each editing task to a specific provider.

### Changed
- Provider routing no longer falls back to a different service.
- A fresh install with no AI provider now shows Set Up an AI Provider.

### Fixed
- The AI sidebar now refreshes Claude Code and Codex status after closing Settings.

## [0.17.02535] - 2026-09-03

### Fixed
- Improved Read Aloud voice casting for dialogue tags split into narration.
- Voice casting now only considers characters who can appear in the current chapter.

## [0.17.02534] - 2026-09-03

### Fixed
- Voice casting now recognizes dialogue tags that use part of a character's name.

## [0.17.02533] - 2026-09-03

### Fixed
- Cast mode no longer reads narrator asides in a character's voice.

## [0.17.02532] - 2026-09-03

### Fixed
- Improved consistency of Read Aloud speaker attribution.

## [0.17.02531] - 2026-09-03

### Changed
- Moved the Read Aloud glow toggle to Settings and extended the speed range to 2.0x.

## [0.17.02530] - 2026-09-03

### Added
- Added the Voice Cast panel with a voice picker, preview, and line counts for each speaker.

## [0.17.02529] - 2026-09-03

### Added
- Added an expanded Read Aloud panel with the current sentence, voice cast, and progress.

## [0.17.02528] - 2026-09-03

### Added
- Cast mode now reads each character's dialogue in their assigned voice.

## [0.17.02527] - 2026-09-03

### Changed
- Redesigned the Read Aloud player with seek, speed, volume, and a current speaker readout.

### Added
- Added a button to reopen the Read Aloud player.

## [0.17.02526] - 2026-09-03

### Added
- Voice cast assignments are saved inside the book.
- Added automatic voice casting that never overrides manual choices.
- Added elapsed and remaining time to Read Aloud.
- Added Read Aloud volume control.

## [0.17.02525] - 2026-09-03

### Added
- Added speaker attribution for dialogue in Read Aloud.

## [0.17.02524] - 2026-09-03

### Fixed
- Changing voice or speed while the player is idle now takes effect on the next play.
- Starting Read Aloud from a specific sentence now begins at that sentence.

## [0.17.02523] - 2026-09-03

### Changed
- Read Aloud now defaults to 1.1x speed.

## [0.17.02522] - 2026-09-03

### Fixed
- Restored natural pauses between sentences in Read Aloud.

## [0.17.02521] - 2026-09-03

### Fixed
- Removed a metallic whine from Read Aloud audio.
- Fixed gaps between sentences in Read Aloud.

## [0.17.02520] - 2026-09-03

### Changed
- Read Aloud now uses only the native voice engine.
- The voice bundle is now a single download for each platform.

### Fixed
- Fixed native Read Aloud failing with a missing sample rate error.

## [0.17.02519] - 2026-09-03

### Added
- Added a native Read Aloud voice engine for Windows, macOS, and Linux.

### Changed
- Native Read Aloud starts playback as soon as the first sentence is ready.

## [0.17.02516] - 2026-09-03

### Fixed
- Read Aloud now buffers enough audio before playback to avoid stalls.
- Closing Read Aloud during buffering now cancels pending work.

## [0.17.02515] - 2026-09-03

### Fixed
- Reduced gaps between sentences in Read Aloud.
- Long sentences now start playing sooner.
- Read Aloud prepares audio while the player is open, so playback starts immediately.

## [0.17.02513] - 2026-09-03

### Changed
- Analysis performance settings now control worker count.

## [0.17.02512] - 2026-09-03

### Fixed
- Fixed Read Aloud using excessive memory during long sessions.

## [0.17.02511] - 2026-09-03

### Added
- Voice model installs can now be verified and repaired.

### Fixed
- Read Aloud files are now served locally so the voice engine loads reliably.

## [0.17.02510] - 2026-09-03

### Fixed
- Fixed a Read Aloud engine crash and repeated pipeline rebuilds.

### Changed
- Read Aloud starts playback faster.

## [0.17.02508] - 2026-09-03

### Changed
- Read Aloud synthesis is much faster.
- The GPU option now uses the full precision voice model.

### Added
- Added a Read Aloud performance check to Settings.

## [0.17.02506] - 2026-09-03

### Fixed
- Fixed distorted Read Aloud audio.

### Changed
- Replaced the floating Read Aloud widget with a docked player.
- The current sentence highlight is easier to see.
- Rebuilt Read Aloud settings with guided model download.

## [0.17.02504] - 2026-09-03

### Added
- Read Aloud now works in the editor with live sentence highlighting.

## [0.17.02503] - 2026-09-03

### Added
- Added offline voice synthesis for Read Aloud.

## [0.17.02502] - 2026-09-03

### Added
- Added sentence detection for Read Aloud that handles dialogue, abbreviations, and initials.

## [0.17.02501] - 2026-09-03

### Added
- Added the Read Aloud plugin with eight voices and speed control, off by default.

## [0.17.02500] - 2026-09-03

### Added
- Added the local voice model download for Read Aloud.

## [0.17.02499] - 2026-09-02

### Added
- Added a hierarchical Story Map with overview, sequence, scene, and detail zoom levels.
- Added Story Time and Manuscript Order views to the Story Map.

### Changed
- Moved to version 0.17.
- Unchanged chapters are no longer reanalyzed.

## [0.16.02498] - 2026-09-01

### Added
- Redesigned the bottom bar with Story Map, Threads, Review, Continuity, and Ask Draftline tabs.

### Removed
- Removed the Story Graph tab, replaced by the Story Map.

## [0.16.02497] - 2026-09-01

### Added
- Added character voice profiles built from attributed dialogue.
- Added per-character voice notes.

## [0.16.02496] - 2026-09-01

### Added
- Added continuity checks for conflicting character details, locations, and dormant plot threads.
- Added story questions about chronology, character locations, and what characters know.

## [0.16.02495] - 2026-09-01

### Added
- Added plot threads based on questions, promises, goals, and threats in the story.
- Added author checkpoints for tracking plot requirements.

## [0.16.02494] - 2026-09-01

### Changed
- Improved event detection by combining nearby sentences that describe the same event.

## [0.16.02493] - 2026-09-01

### Added
- Added story chronology that separates story time from manuscript order.
- Author corrections now survive reanalysis.

## [0.16.02492] - 2026-09-01

### Added
- Added CPU profiles for background analysis.

### Changed
- The Intertwined view now shows the full Story Graph with story beats.
- Character rails now show gaps where a character is absent.
- Added zoom and layer toggles to the Intertwined view.

### Fixed
- Background analyses no longer run at the same time and compete for CPU.

## [0.16.02491] - 2026-09-01

### Added
- Added an Intertwined character timeline.

### Changed
- Story Graph threads now follow story evidence instead of cast combinations.
- The Character Center remembers the last view used.

## [0.16.02490] - 2026-09-01

### Changed
- Rebuilt Continuity as a question queue with a detail pane.

### Added
- Continuity questions can be marked reviewed or dismissed, and decisions are saved.

## [0.16.02489] - 2026-09-01

### Removed
- Removed the Timeline view, replaced by the Story Graph.

## [0.16.02488] - 2026-09-01

### Added
- Added the Story Graph, showing story beats across chapters by thread or character.

### Changed
- Redesigned the bottom bar and Ask Draftline layout.

## [0.16.02487] - 2026-09-01

### Added
- Added the Continuity workspace with checks for character names, knowledge, physical details, and time references.

## [0.16.02486] - 2026-09-01

### Added
- Added an automatic Story Timeline with filters for characters, places, and event types.

## [0.16.02485] - 2026-09-01

### Added
- Added the Ask Draftline explorer with suggested questions about the manuscript.

## [0.16.02484] - 2026-09-01

### Added
- Added character knowledge tracking and knowledge questions in search.

## [0.16.02483] - 2026-09-01

### Changed
- Opening a .storiverse file now shows a notice that support is coming.

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
