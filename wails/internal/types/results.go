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
	Type       string `json:"type"` // "book" | "universe"
	Path       string `json:"path"`
	Name       string `json:"name"`
	LastOpened string `json:"lastOpened"` // ISO 8601 timestamp
	// CoverKey addresses this project's cover art at /recent-cover/<key>.
	// It is derived from the path and is stable while the path is, so it
	// survives the list reordering when a book is opened -- addressing the
	// art by POSITION did not, and every book briefly wore the art of
	// whichever one had moved into its place. The path itself never crosses
	// to the webview: a handler taking one would read any file on request.
	CoverKey string             `json:"coverKey"`
	Stats    RecentProjectStats `json:"stats"`
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

// BookLockInfo says whether a book looks open on another device.
//
// Every field is a hint, never a fact: the claim it comes from travels by
// whatever syncs the author's folder, and that can be minutes behind. The
// wording in Message is hedged for that reason and the UI should not
// un-hedge it. See internal/booklock.
type BookLockInfo struct {
	// Held is false when nothing claims the book, or when this session does.
	Held bool `json:"held"`
	// Stale means nobody has refreshed the claim recently, so the device
	// holding it has most likely stopped running.
	Stale    bool   `json:"stale"`
	Device   string `json:"device"`
	Platform string `json:"platform"`
	App      string `json:"app"`
	LastSeen string `json:"last_seen"`
	// Message is the sentence to show the author.
	Message string `json:"message"`
}

// BookTakeoverRequest is another device asking for the open book, as it
// reaches the frontend that has to put the question to whoever is here.
type BookTakeoverRequest struct {
	Device      string `json:"device"`
	Platform    string `json:"platform"`
	App         string `json:"app"`
	RequestedAt string `json:"requested_at"`
	// Message is the sentence to show whoever is at this machine.
	Message string `json:"message"`
}

// BookTakeoverStatus is what the asking device polls while it waits.
//
// Arrived is the only field that promises anything. Everything else is a hint
// in the same way BookLockInfo is: it comes out of a sidecar that travels by
// whatever syncs the folder.
type BookTakeoverStatus struct {
	// Asked means a request of ours is still beside the book.
	Asked bool `json:"asked"`
	// HeldElsewhere means the claim is still there, answered or not.
	HeldElsewhere bool `json:"held_elsewhere"`
	Answered      bool `json:"answered"`
	Granted       bool `json:"granted"`
	Declined      bool `json:"declined"`
	// Arrived means the book beside us hashes to what the other device wrote,
	// which is the only proof that the copy is current rather than whatever
	// the sync client has not replaced yet.
	Arrived bool `json:"arrived"`
	// Unverifiable means the handover carried no fingerprint, so there is
	// nothing to check this copy against.
	Unverifiable  bool   `json:"unverifiable"`
	LocalBytes    int64  `json:"local_bytes"`
	ExpectedBytes int64  `json:"expected_bytes"`
	Responder     string `json:"responder,omitempty"`
	// Note is why a handover carried no fingerprint, when that happened.
	Note string `json:"note,omitempty"`
	// Message is the sentence to show the author.
	Message string `json:"message"`
}

// BookTakeoverResult is what came of answering a request for the open book.
type BookTakeoverResult struct {
	Granted  bool `json:"granted"`
	Declined bool `json:"declined"`
	// Withdrawn means nobody is waiting any more, so the book stays here. The
	// frontend must not close it.
	Withdrawn bool `json:"withdrawn"`
	// Fingerprinted is false when the book could not be read to prove what was
	// handed over. The handover still happened.
	Fingerprinted bool   `json:"fingerprinted"`
	Device        string `json:"device,omitempty"`
	Error         string `json:"error,omitempty"`
}
