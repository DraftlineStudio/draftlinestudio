# Changelog

All notable changes to Draftline are listed here, newest first.

## [0.20.02648] - 2026-09-16

### Fixed
- EPUBs without an ISBN now get a random identifier, so two quick exports no longer share one.
- ISBNs typed with hyphens are exported as plain digits.

### Changed
- The editions screen now notes which features are not connected to export yet.

## [0.20.02647] - 2026-09-16

### Added
- Added editions, each holding its formats with their own ISBN, specification, and price.
- Added a copyright page generated from the edition record.
- ISBN-10 and spine width are now calculated automatically.
- Added Duplicate as New Edition, which copies a format's specification without its ISBNs.
- Editions save with the book through normal autosave.

### Removed
- Removed the copyright template from Settings.

## [0.20.02646] - 2026-09-15

### Fixed
- A project file that can no longer be opened can now be saved to a new location.
- Fixed a project that opened normally but failed on every save.

## [0.20.02645] - 2026-09-15

### Fixed
- Save As now keeps chapter history.
- Saving now keeps data in the project file that the current save did not write.

## [0.20.02644] - 2026-09-15

### Added
- Renamed Book Info to Book & Editions.
- ISBNs are now validated as you type.

### Changed
- Exports now declare the book's language instead of always using English.
- Exports print the imprint when set, and the publisher otherwise.
- EPUB packages now declare only the content the book actually includes.

## [0.20.02643] - 2026-09-15

### Fixed
- Fixed styling on synopsis traces.
- The Dead ideas note keeps its name.

## [0.20.02642] - 2026-09-15

### Fixed
- Centered the Planner's empty timeline message.

## [0.20.02640] - 2026-09-15

### Fixed
- The analysis progress bar no longer overlaps the status bar buttons.

### Changed
- The character codex now fills the whole window.

## [0.20.02639] - 2026-09-15

### Added
- The Planner scratchpad can now stay open beside the manuscript.

## [0.20.02638] - 2026-09-15

### Changed
- Renamed the bottom tool window to Plot Inspections and made it open on Continuity.
- Plot Inspections and Character Map have their own status bar buttons.

## [0.20.02637] - 2026-09-15

### Changed
- Moved to version 0.20 for the Planner.

## [0.19.02636] - 2026-09-15

### Removed
- Removed the Story Map, Threads, and Review tabs from the bottom window.
- Removed card reconciliation from the Planner. Cards are now planned, or drafted once linked to a scene.
- Removed the Developments row from the Characters view.

### Changed
- Manuscript analysis runs faster and saves a smaller analysis file.

## [0.19.02619] - 2026-09-15

### Changed
- Renamed the Planner notes view to Scratchpad.
- Deleting a Scratchpad note now asks for confirmation.

## [0.19.02618] - 2026-09-15

### Added
- Added the Planner, a story line timeline across chapters with plot cards, lanes, and a Board view.
- Added Import Outline for Markdown, numbered lists, beat sheets, and chapter synopses.
- Added scratch notes, a Dead ideas note for deleted cards, and a Synopsis view with Markdown export.
- Planner data is saved inside the project file.

## [0.19.02616] - 2026-09-15

### Fixed
- A failed open or Save As no longer releases the lock on the currently open book.

## [0.19.02607] - 2026-09-14

### Fixed
- Fixed Linux package builds by using a pinned prebuilt nfpm.

## [0.19.02606] - 2026-09-14

### Added
- Linux releases now include .deb and .rpm packages.
- The updater can install Linux updates directly.

### Fixed
- Linux packages now launch on Ubuntu 22.04.
- The AppImage no longer requires libfuse2.

## [0.19.02605] - 2026-09-14

### Added
- Added Help & Documentation and Report an Issue to the project menu.

### Changed
- Book accent colors now skip a leading "The" in the title.

### Fixed
- The Edit button on the author card now opens the Author tab.

## [0.19.02604] - 2026-09-14

### Removed
- Removed the Ctrl+L inline prose generation prompt.

## [0.19.02603] - 2026-09-14

### Changed
- The Windows installer can relaunch Draftline after an in-app update.

### Fixed
- Launching a downloaded installer no longer flashes a console window.

## [0.19.02602] - 2026-09-14

### Changed
- Added an Author tab to Settings.
- The update chip now opens Settings with the download ready.

## [0.19.02601] - 2026-09-13

### Fixed
- Closing Settings no longer flashes a console window on Windows.

## [0.19.02600] - 2026-09-13

### Fixed
- Added optics, weapon light, and accessory brands to the spelling supplement.

## [0.19.02599] - 2026-09-13

### Fixed
- A book that fails to open now returns to the welcome screen instead of an empty editor.

## [0.19.02598] - 2026-09-13

### Fixed
- Added about 190 common brand, gear, tech, and slang words to spell check.

## [0.19.02597] - 2026-09-13

### Added
- Added block quote and code block buttons to the editor toolbar.

## [0.19.02596] - 2026-09-13

### Added
- Added manual chapter snapshots with a custom label.

### Changed
- Chapter history compare now shows scene breaks.

## [0.19.02595] - 2026-09-13

### Fixed
- AI chapter passes no longer remove scene breaks, block quotes, or code blocks.
- Partially accepted AI edits inside block quotes, code blocks, and headings keep their formatting.

## [0.19.02593] - 2026-09-12

### Fixed
- Worked around a stale text cursor after switching themes.

## [0.19.02592] - 2026-09-12

### Fixed
- The text cursor now matches the writing canvas in the light theme.
- The page backdrop now fades with the rest of the interface during theme changes.
- Switching themes again mid-animation now restarts the sky animation cleanly.

## [0.19.02591] - 2026-09-12

### Fixed
- Fixed the text cursor staying white over the light theme when Windows is in dark mode.

## [0.19.02590] - 2026-09-12

### Changed
- Draftline can now run multiple windows, and each book can only be open in one window at a time.

## [0.19.02589] - 2026-09-12

### Fixed
- Plugin bundles now load in development builds.

## [0.19.02588] - 2026-09-11

### Added
- Added the plugin platform: plugin discovery, settings, sidecar processes, and a permission-gated frontend API.
- Added plugin developer documentation.

## [0.19.02586] - 2026-09-09

### Changed
- Moved to version 0.19 for the new launch screen and open flow.

## [0.18.02585] - 2026-09-09

### Changed
- The manuscript word count is now saved in the book's metadata.

## [0.18.02584] - 2026-09-09

### Added
- Added a loading overlay while a book opens.

### Fixed
- Clicking open twice while a book is loading no longer opens the wrong book.

## [0.18.02583] - 2026-09-09

### Changed
- Redesigned the launch screen around the last opened book.

## [0.18.02582] - 2026-09-09

### Fixed
- The recent projects list is up to date after closing a book.

## [0.18.02581] - 2026-09-09

### Fixed
- The updater now reports correctly when a rejected download could not be deleted.

## [0.18.02580] - 2026-09-09

### Added
- Added a monospace print font for code blocks.

### Changed
- Print-ready body text now defaults to about 9 points.

### Fixed
- Print scene breaks no longer render as missing glyph boxes.
- Drop caps now enlarge the first letter instead of an opening quotation mark.
- Paragraph, heading, and code block alignment now overrides the edition default.

## [0.18.02579] - 2026-09-09

### Added
- Added separate print typefaces for headings, running heads, page numbers, and title pages.
- Added classic, minimal, and dramatic title page layouts.

### Changed
- Print PDFs now embed only the fonts the chosen design uses.

## [0.18.02578] - 2026-09-09

### Changed
- Print-ready editions now default to 10 point, left-aligned body text.

### Fixed
- Fixed the last manuscript page after generating a table of contents.

## [0.18.02577] - 2026-09-08

### Changed
- DOCX export now uses the shared export document model.
- DOCX files now include metadata, named Word styles, real list numbering, and hyperlinks.

### Fixed
- PDF hyperlinks are now clickable.
- A failed DOCX export no longer leaves a partial file behind.

## [0.18.02576] - 2026-09-08

### Added
- Added EPUB options for embedded fonts, paragraph style, alignment, chapter openings, and scene breaks.

### Changed
- Rebuilt EPUB export on the shared export document model.
- Moved to version 0.18 for the new export system.

### Fixed
- EPUBs no longer include editor-only markup or unsafe links.

## [0.17.02575] - 2026-09-08

### Added
- Added a new Unicode PDF renderer with bundled Merriweather and Lato fonts.
- Reading PDFs now have page size, typeface, spacing, alignment, and indent options.

### Changed
- Print-ready PDF options now all apply to the exported file, including bleed, crop marks, mirrored margins, drop caps, and running heads.

### Fixed
- Fixed wrong fonts, broken punctuation, bad wrapping, and blank pages in PDF exports.
- A failed PDF export no longer leaves a partial file behind.

## [0.17.02574] - 2026-09-08

### Added
- Added a shared export document model used by every export format.

## [0.17.02573] - 2026-09-08

### Fixed
- Copyright, front matter, and back matter can be selected for export even when empty.
- PDFs now render curly quotes, dashes, ellipses, and accented letters correctly.

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
