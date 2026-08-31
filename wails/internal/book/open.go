// Package book provides book lifecycle operations (open, save).
package book

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"

	"draftline/internal/types"
	"draftline/internal/ziputil"
)

const (
	// MaxManifestRefs caps the combined number of chapter/section references a
	// manifest may declare. ziputil bounds the ZIP's entry count and byte size,
	// but a manifest can reference the same valid entry an unbounded number of
	// times, loading it into memory on each reference and bypassing the
	// archive-level aggregate bound. 5000 is far beyond any real manuscript.
	MaxManifestRefs = 5000
	// MaxTotalLoadedBytes caps the cumulative decompressed content actually
	// loaded across one open operation. This closes the amplification gap where
	// N references to a single large-but-legal entry multiply memory use.
	MaxTotalLoadedBytes = ziputil.MaxTotalSize
)

// ReadZipEntry reads a named entry from a ZIP archive, enforcing size limits.
func ReadZipEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	return ziputil.ReadNamed(r.File, name, false)
}

// byteBudget tracks cumulative decompressed bytes loaded across one open
// operation and fails once the aggregate exceeds MaxTotalLoadedBytes. A single
// entry is individually bounded by ziputil, but repeated references to the same
// entry are not, so the budget must be enforced at the aggregate level.
type byteBudget struct {
	used int64
}

func (b *byteBudget) add(n int) error {
	b.used += int64(n)
	if b.used > MaxTotalLoadedBytes {
		return fmt.Errorf("archive references more than %d bytes of content when expanded (limit exceeded)", MaxTotalLoadedBytes)
	}
	return nil
}

// readChapterContent reads a manifest-referenced chapter entry. A missing,
// oversized, or otherwise unreadable entry is a hard error rather than an empty
// chapter: silently substituting empty content would let a subsequent autosave
// overwrite the (still intact) source with nothing. The error names the chapter
// and file, and distinguishes an absent entry from a read/size failure. The
// running budget guards against reference amplification across the whole open.
func readChapterContent(r *zip.ReadCloser, title, file string, budget *byteBudget) (string, error) {
	content, err := ReadZipEntry(r, file)
	if err != nil {
		if errors.Is(err, ziputil.ErrEntryNotFound) {
			return "", fmt.Errorf("chapter %q references entry %q which is missing from the archive", title, file)
		}
		return "", fmt.Errorf("chapter %q entry %q could not be read: %w", title, file, err)
	}
	if err := budget.add(len(content)); err != nil {
		return "", err
	}
	return string(content), nil
}

// Open reads a .draftline file and returns the BookData.
func Open(path string) (types.BookData, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.BookData{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = r.Close() }()

	if err := ziputil.CheckArchive(r.File); err != nil {
		return types.BookData{}, fmt.Errorf("refusing to open archive: %w", err)
	}

	manifestData, err := ReadZipEntry(r, "manifest.json")
	if err != nil {
		return types.BookData{}, fmt.Errorf("invalid .draftline file: missing manifest.json")
	}

	var raw struct {
		Version  string         `json:"version"`
		Metadata types.Metadata `json:"metadata"`
		Chapters []struct {
			Title string `json:"title"`
			File  string `json:"file"`
		} `json:"chapters,omitempty"`
		FrontMatter []struct {
			ID       string `json:"id,omitempty"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"front_matter,omitempty"`
		Body []struct {
			ID       string `json:"id,omitempty"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"body,omitempty"`
		BackMatter []struct {
			ID       string `json:"id,omitempty"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"back_matter,omitempty"`
		WritingGoals types.WritingGoals        `json:"writing_goals,omitempty"`
		StyleOptions types.WritingStyleOptions `json:"style_options,omitempty"`
		IsIndexed    bool                      `json:"is_indexed,omitempty"`
		LastIndexed  string                    `json:"last_indexed,omitempty"`
	}

	if err := json.Unmarshal(manifestData, &raw); err != nil {
		return types.BookData{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Bound the combined number of manifest references before loading any
	// content. Without this, a small manifest can reference the same valid
	// entry thousands of times and amplify memory use far beyond the ZIP's
	// own entry/size limits.
	totalRefs := len(raw.Chapters) + len(raw.FrontMatter) + len(raw.Body) + len(raw.BackMatter)
	if totalRefs > MaxManifestRefs {
		return types.BookData{}, fmt.Errorf("manifest declares %d chapter/section references (limit %d)", totalRefs, MaxManifestRefs)
	}
	budget := &byteBudget{}

	book := types.BookData{
		Version:      raw.Version,
		Metadata:     raw.Metadata,
		FilePath:     path,
		FrontMatter:  []types.ChapterItem{},
		Body:         []types.ChapterItem{},
		BackMatter:   []types.ChapterItem{},
		WritingGoals: raw.WritingGoals,
		StyleOptions: raw.StyleOptions,
		IsIndexed:    raw.IsIndexed,
		LastIndexed:  raw.LastIndexed,
	}

	// v1.0 migration: chapters/ -> body/
	if raw.Version == "1.0" {
		for _, ch := range raw.Chapters {
			content, err := readChapterContent(r, ch.Title, ch.File, budget)
			if err != nil {
				return types.BookData{}, err
			}
			book.Body = append(book.Body, types.ChapterItem{
				Title:   ch.Title,
				Type:    "Chapter",
				Content: content,
			})
		}
		EnsureBookChapterIDs(&book)
		return book, nil
	}

	// v2.0: copyright.html is written by convention (not in the manifest chapter
	// arrays). Its absence is tolerated for legacy files, but if it exists and
	// cannot be read (oversized/corrupt) we must not silently drop author text.
	if copyright, err := ReadZipEntry(r, "copyright.html"); err == nil {
		book.Copyright = string(copyright)
	} else if !errors.Is(err, ziputil.ErrEntryNotFound) {
		return types.BookData{}, fmt.Errorf("copyright entry %q could not be read: %w", "copyright.html", err)
	}

	for _, item := range raw.FrontMatter {
		content, err := readChapterContent(r, item.Title, item.File, budget)
		if err != nil {
			return types.BookData{}, err
		}
		book.FrontMatter = append(book.FrontMatter, types.ChapterItem{
			ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: content,
		})
	}
	for _, item := range raw.Body {
		content, err := readChapterContent(r, item.Title, item.File, budget)
		if err != nil {
			return types.BookData{}, err
		}
		book.Body = append(book.Body, types.ChapterItem{
			ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: content,
		})
	}
	for _, item := range raw.BackMatter {
		content, err := readChapterContent(r, item.Title, item.File, budget)
		if err != nil {
			return types.BookData{}, err
		}
		book.BackMatter = append(book.BackMatter, types.ChapterItem{
			ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: content,
		})
	}
	EnsureBookChapterIDs(&book)

	// story_bible.json (optional — not present in older files)
	if bibleData, err := ReadZipEntry(r, "story_bible.json"); err == nil {
		var bible types.StoryBible
		if json.Unmarshal(bibleData, &bible) == nil {
			if bible.Characters == nil {
				bible.Characters = []types.Character{}
			}
			book.StoryBible = bible
		}
	}
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}

	// analysis.json was introduced in archive v2.1. Its absence is valid for
	// older projects; those books remain openable and can rebuild the cache on
	// the next Detect Characters run.
	if analysisData, err := ReadZipEntry(r, "analysis.json"); err == nil {
		var analysis types.AnalysisData
		if json.Unmarshal(analysisData, &analysis) == nil {
			book.Analysis = analysis
		}
	}
	if book.IsIndexed && book.Analysis.EntityResolution == nil {
		// A legacy manifest may claim to be indexed even though older writers
		// never persisted the corresponding analysis. Do not expose stale UI.
		book.IsIndexed = false
		book.LastIndexed = ""
	}

	// beat_sheet.json (optional)
	if beatData, err := ReadZipEntry(r, "beat_sheet.json"); err == nil {
		var beatSheet types.BeatSheet
		if json.Unmarshal(beatData, &beatSheet) == nil {
			book.BeatSheet = beatSheet
		}
	}

	// foreshadowing.json (optional)
	if foreshadowData, err := ReadZipEntry(r, "foreshadowing.json"); err == nil {
		var ledger types.ForeshadowingLedger
		if json.Unmarshal(foreshadowData, &ledger) == nil {
			book.ForeshadowingLedger = ledger
		}
	}

	// knowledge_matrix.json (optional)
	if matrixData, err := ReadZipEntry(r, "knowledge_matrix.json"); err == nil {
		var matrix types.KnowledgeMatrix
		if json.Unmarshal(matrixData, &matrix) == nil {
			book.KnowledgeMatrix = matrix
		}
	}

	return book, nil
}
