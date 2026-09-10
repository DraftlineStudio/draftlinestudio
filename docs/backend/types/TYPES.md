# Types Package

`internal/types` contains all shared data structures used throughout the Draftline backend.

## Files

| File | Contents |
|------|----------|
| `book.go` | Core book structures: `BookData`, `Metadata`, `ChapterItem`, `WritingGoals`, `WritingStyleOptions` |
| `characters.go` | Story bible types: `Character`, `StoryBible` |
| `structure.go` | Plotting tools: `Beat`, `BeatSheet`, `ForeshadowingItem`, `ForeshadowingLedger`, `SecretInfo`, `KnowledgeEntry`, `KnowledgeMatrix` |
| `settings.go` | Configuration: `AppSettings`, `ClaudeCodeStatus` |
| `results.go` | Operation results: `SaveResult`, `ExportResult`, `ImportResult`, `AIRewriteResult`, `IndexResult`, `BackupInfo`, `RecentProject`, `InlineGenerateRequest` |
| `export.go` | Export options: `ExportOptions`, `PDFOptions`, `PrintPDFOptions`, `EPUBOptions` |
| `entities.go` | Entity resolution records and the `AnalysisData` container (entity resolution, relationships, story metrics, evidence, fingerprint, continuity decisions) |
| `evidence.go` | `EvidenceData` — the rebuildable, source-located local fact/event index |
| `relationships.go` | Scene records and character relationship/interaction data |
| `fingerprint.go` | `StoryFingerprint` — the durable manuscript-memory model (typed frames, ledgers, developments) |
| `continuity.go` | Continuity signals, exact source locations, and author decisions |
| `storyanalysis.go` | `StoryAnalysisData` — rebuildable evidence-based manuscript metrics |
| `storysearch.go` | Story/Detail Search request and result types |
| `storytimeline.go` | Timeline events and derived facets |

## Usage

All Go files in the `wails/` directory import this package:

```go
import "draftline/internal/types"

func (a *App) NewBook() types.BookData {
    return types.BookData{
        Version: "2.0",
        Metadata: types.Metadata{
            Title: "Untitled",
        },
        // ...
    }
}
```

## Design Principles

1. **No circular dependencies** - Types package has no imports from other internal packages
2. **JSON serialization** - All types use `json` struct tags for frontend communication
3. **Nullable fields** - Use pointers (`*int`) for optional fields that can be `null` in JSON
4. **Consistent naming** - PascalCase for exported types, matching frontend TypeScript interfaces

## Type Relationships

```
BookData
├── Metadata
├── ChapterItem[]
├── StoryBible
│   └── Character[]
├── WritingGoals
├── WritingStyleOptions
├── BeatSheet
│   └── Beat[]
├── ForeshadowingLedger
│   └── ForeshadowingItem[]
├── KnowledgeMatrix
│   ├── SecretInfo[]
│   └── KnowledgeEntry[]
└── AnalysisData
    ├── EntityData (entity resolution)
    ├── RelationshipData
    ├── StoryAnalysisData
    ├── EvidenceData
    ├── StoryFingerprint
    └── ContinuityData (author decisions)
```

## Result Types

All backend operations return result structs rather than using Go's error pattern directly. This allows clean JSON serialization for the frontend:

```go
type SaveResult struct {
    Success  bool   `json:"success"`
    FilePath string `json:"file_path"`
    Error    string `json:"error,omitempty"`
}
```

Frontend can then simply check `result.success` without try/catch.
