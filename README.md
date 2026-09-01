# Draftline

Draftline is a desktop writing environment for novelists. It combines a focused rich-text editor with offline spelling and grammar checks, character and relationship indexing, story-planning tools, and optional AI-assisted editing.

Draftline is under active pre-1.0 development. Windows is the primary development and testing target.

[Official website](https://draftline.ink) · [Changelog](CHANGELOG.md) · [Documentation](docs/README.md)

## Features

- Rich chapter-based manuscript editing powered by TipTap and ProseMirror
- Fast offline spelling and grammar diagnostics
- Character detection, entity resolution, aliases, appearances, and relationship analysis
- Private local chapter structure, pacing, readability, dialogue, keyword, and extractive-summary analysis
- Story bible, timelines, beat sheets, foreshadowing, and knowledge tracking
- Chapter-scoped find and replace
- Local evidence-backed Ask Draftline explorer with manuscript-generated starting points, natural questions, confirmed-character aliases, chapter and character-knowledge trails, missing-name and singleton signals, exact excerpts, and source navigation
- Persistent local fact/event evidence indexing with exact source sentences, durable author review decisions, a ranked review queue, and an inspectable Evidence Archive
- Optional activity-based local saves, embedded chapter version history with side-by-side restore, and rolling manuscript backups
- EPUB and DOCX import, with export support currently being redesigned
- Optional AI line editing, copy editing, rewriting, and inline assistance
- Bundled plugins with capability/resource metadata and a marketplace foundation for optional local analysis packs

## Privacy and AI

Draftline stores manuscripts locally in `.draftline` project files. Spelling, grammar, character indexing, relationships, and bundled story/pacing analysis run locally.

AI features are optional and disabled until configured. When an AI feature is used, the selected manuscript text and its editing instructions are sent to the provider or command-line tool selected in Settings. Supported paths include local endpoints, provider APIs, Claude Code, and OpenAI Codex. Those services have their own privacy and data-retention policies.

Draftline installs and runs tested, version-pinned Claude Code and Codex CLI releases from its local application-data directory. AI editing requests run in disposable workspaces with model-facing shell, filesystem, browser, application, and sub-agent tools disabled. Global CLI installations on `PATH` are not used by Draftline.

AI debug logging is opt-in because its logs can contain prompts and manuscript text. Draftline also creates local rolling backups containing the complete manuscript.

## Technology

- Go backend using [Wails v2](https://wails.io/)
- React 18 and TypeScript frontend
- TipTap/ProseMirror editor
- Zustand state management
- prose/v3 local natural-language processing
- Vitest and Go test suites

## Building from source

### Requirements

- Go 1.25 or newer
- Node.js with npm
- Wails CLI 2.15.0
- The platform dependencies listed in the [Wails installation guide](https://wails.io/docs/gettingstarted/installation/)

Install the matching Wails CLI if needed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
```

Run Draftline in development mode:

```powershell
cd wails
wails dev
```

Build a desktop executable:

```powershell
cd wails
wails build
```

## Testing

From `wails`:

```powershell
go test ./...
go vet ./...
```

From `wails/frontend`:

```powershell
npm test
npx tsc --noEmit
npm run build
```

## Repository layout

```text
Draftline/
├── docs/                  Architecture, data-model, feature, and roadmap documentation
├── wails/                 Desktop application and Go backend
│   ├── frontend/          React and TypeScript user interface
│   └── internal/          Book, backup, indexing, export, and support packages
├── CHANGELOG.md           Linear build-version history
├── ROADMAP.md             Current product roadmap and refactoring plans
└── LICENSE                MIT License
```

The `capacitor` directory contains experimental work and is not the primary desktop application.

## The `.draftline` format

A `.draftline` project is a ZIP-based archive containing a manifest, chapter HTML, book metadata, embedded chapter history, and optional analysis and story-planning data. See the [data-model documentation](docs/data-model/DATA-MODEL.md) and [book backend documentation](docs/backend/book/BOOK.md) for details.

Keep independent backups of important manuscripts. Compatibility and migration behavior may still change before the first stable release.

## Versioning

Draftline uses linear build versioning in the form `MAJOR.MINOR.BUILD`. The build component increases for each change on the main branch; major and minor components identify manually selected milestones. It does not claim Semantic Versioning compatibility.

See [docs/VERSION.md](docs/VERSION.md) for the version-maintenance process.

## License

Draftline is available under the [MIT License](LICENSE).
