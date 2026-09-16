package types

// Frozen manuscripts.
//
// An edition record says which ISBN a book was sold under. It does not say
// which words went out under it, and those are not the same thing: an author
// who publishes a first edition, spends a year revising, and then needs to
// re-export the first edition for a new retailer would otherwise ship the
// second edition's text under the first edition's number.
//
// A snapshot is the manuscript as it stood at one moment: every section's
// HTML, the order and titles the manifest gave them, the author's own
// copyright page, and the metadata. Anything less and "as it was" is not true.
//
// Two rules, and both are about where the bytes are:
//
//   - A snapshot's text never joins BookData. It is frozen text, often a whole
//     novel, and BookData is serialised to JSON across the Wails bridge on
//     every autosave. Only the small record below travels; the text goes to
//     the writer as a sibling parameter, exactly as cover art does.
//   - A snapshot is content-addressed. Its identifier IS the hash of what it
//     contains, so freezing the same unchanged text from a paperback and from
//     an ebook produces one identifier and one stored copy, referenced twice.

// EditionSnapshot is the catalogue record of one frozen manuscript: enough to
// describe it on screen and to account for what it occupies, and nothing that
// would need the text itself to be read.
//
// It lives in editions/index.json, beside the editions, because the reference
// count that decides whether the bytes are kept has to be a fact the record
// states rather than one inferred from the archive.
type EditionSnapshot struct {
	// ID is the SHA-256 of the frozen content, which is also the folder the
	// content is filed under.
	ID string `json:"id"`
	// Frozen is when this text was first frozen, RFC 3339. Freezing the same
	// text again keeps the original moment: it is the same snapshot.
	Frozen string `json:"frozen"`
	// Title is the book's title as it stood, so a record can be described
	// without reading the text back.
	Title     string `json:"title,omitempty"`
	WordCount int    `json:"word_count"`
	Sections  int    `json:"sections"`
	// Members and Bytes are what this snapshot occupies inside the project
	// file. The Editions screen reports them, because a project with several
	// frozen editions is the first thing Draftline has ever written that can
	// approach the archive's own limits.
	Members int   `json:"members"`
	Bytes   int64 `json:"bytes"`
}

// SnapshotResult is what freezing hands back to the screen. The record is the
// catalogue entry to add to the book; the bytes it describes stay behind.
type SnapshotResult struct {
	Success  bool             `json:"success"`
	Error    string           `json:"error,omitempty"`
	Snapshot *EditionSnapshot `json:"snapshot,omitempty"`
	// Reused is true when this exact text had already been frozen. The project
	// file then holds one copy that two formats point at, and the screen says
	// so rather than implying a second copy was made.
	Reused bool `json:"reused,omitempty"`
}

// FindSnapshot returns the catalogue record with the given identifier.
func (idx *EditionIndex) FindSnapshot(id string) (EditionSnapshot, bool) {
	if idx == nil || id == "" {
		return EditionSnapshot{}, false
	}
	for _, snapshot := range idx.Snapshots {
		if snapshot.ID == id {
			return snapshot, true
		}
	}
	return EditionSnapshot{}, false
}

// SnapshotReferences lists the formats exported from one frozen manuscript.
//
// This is the reference count, and it is the whole of the retention policy.
// Bytes are kept while this is non-empty and dropped by the next save when it
// is empty. Nothing else decides: an over-eager drop destroys the text a
// published ISBN refers to, and a missed one is copied into the project file
// and into every rolling backup on every autosave for the life of the book.
func (idx *EditionIndex) SnapshotReferences(id string) []string {
	if idx == nil || id == "" {
		return nil
	}
	var formats []string
	for _, edition := range idx.Editions {
		for _, format := range edition.Formats {
			if format.SnapshotID == id {
				formats = append(formats, format.ID)
			}
		}
	}
	return formats
}

// ReferencedSnapshots is the set of frozen manuscripts at least one format was
// exported from.
func (idx *EditionIndex) ReferencedSnapshots() map[string]bool {
	referenced := map[string]bool{}
	if idx == nil {
		return referenced
	}
	for _, edition := range idx.Editions {
		for _, format := range edition.Formats {
			if format.SnapshotID != "" {
				referenced[format.SnapshotID] = true
			}
		}
	}
	return referenced
}

// SnapshotFootprint is what every frozen manuscript in this book occupies:
// archive members and uncompressed bytes.
func (idx *EditionIndex) SnapshotFootprint() (members int, bytes int64) {
	if idx == nil {
		return 0, 0
	}
	for _, snapshot := range idx.Snapshots {
		members += snapshot.Members
		bytes += snapshot.Bytes
	}
	return members, bytes
}
