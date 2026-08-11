package types

// SaveResult contains the result of a save operation.
type SaveResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
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
	Stats      RecentProjectStats `json:"stats"`
}

// RecentProjectStats contains statistics for a recent project.
type RecentProjectStats struct {
	Books    *int `json:"books,omitempty"`
	Chapters int  `json:"chapters"`
	Words    int  `json:"words"`
}

// InlineGenerateRequest contains the context for inline AI generation.
type InlineGenerateRequest struct {
	Instruction   string   `json:"instruction"`    // User's prompt
	BeforeContext string   `json:"before_context"` // Text before cursor (2-3 paragraphs)
	AfterContext  string   `json:"after_context"`  // Text after cursor (1-2 paragraphs)
	Characters    []string `json:"characters"`     // Character names from story bible
	ChapterTitle  string   `json:"chapter_title"`  // Current chapter title
}
