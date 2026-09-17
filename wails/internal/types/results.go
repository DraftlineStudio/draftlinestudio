package types

// SaveResult contains the result of a save operation.
type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
	// Warnings lists what the save could not do without failing, such as
	// preserved data it could not read out of the project it saved from. A
	// successful save with warnings wrote the book but left something behind.
	Warnings []string `json:"warnings,omitempty"`
}

// ExportResult contains the result of an export operation.
type ExportResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ImportResult holds the result of an import operation.
type ImportResult struct {
	Success bool     `json:"success"`
	Book    BookData `json:"book,omitempty"`
	Error   string   `json:"error,omitempty"`
	// Warnings lists non-fatal issues (skipped or truncated content) so the
	// frontend can show what the import left out instead of losing it silently.
	Warnings []string `json:"warnings,omitempty"`
}

// AIRewriteResult contains the result of an AI rewrite operation.
type AIRewriteResult struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// IndexResult contains the results of character indexing.
type IndexResult struct {
	Success         bool        `json:"success"`
	Error           string      `json:"error,omitempty"`
	CharactersFound int         `json:"characters_found"`
	NewCharacters   int         `json:"new_characters"`
	Characters      []Character `json:"characters,omitempty"`
	// Book is the updated book (characters + entity resolution data).
	Book BookData `json:"book"`
}

// SplitEntityResult contains the result of splitting an entity.
type SplitEntityResult struct {
	Success    bool        `json:"success"`
	Error      string      `json:"error,omitempty"`
	Book       BookData    `json:"book,omitempty"`
	Characters []Character `json:"characters,omitempty"`
}

// BackupInfo contains metadata about a backup file.
type BackupInfo struct {
	Number   int    `json:"number"`
	Path     string `json:"path"`
	Modified string `json:"modified"`
	Size     int64  `json:"size"`
}

// RecentProject represents a recently opened project in the start screen.
type RecentProject struct {
	Type       string             `json:"type"` // "book" | "universe"
	Path       string             `json:"path"`
	Name       string             `json:"name"`
	LastOpened string             `json:"lastOpened"` // ISO 8601 timestamp
	// CoverKey addresses this project's cover art at /recent-cover/<key>.
	// It is derived from the path and is stable while the path is, so it
	// survives the list reordering when a book is opened -- addressing the
	// art by POSITION did not, and every book briefly wore the art of
	// whichever one had moved into its place. The path itself never crosses
	// to the webview: a handler taking one would read any file on request.
	CoverKey   string             `json:"coverKey"`
	Stats      RecentProjectStats `json:"stats"`
}

// RecentProjectStats contains statistics for a recent project.
type RecentProjectStats struct {
	Books    *int `json:"books,omitempty"`
	Chapters int  `json:"chapters"`
	Words    int  `json:"words"`
}

// RevealResult is the answer to showing a file in the system file browser:
// it either happened or it did not, and there is one sentence to say when
// something was different from what the record expected.
type RevealResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Note    string `json:"note,omitempty"`
}
