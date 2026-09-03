package book

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"draftline/internal/backup"
	"draftline/internal/fsutil"
	"draftline/internal/types"
	"draftline/internal/ziputil"
)

// Write saves a BookData to a .draftline file at the specified path.
// appVersion should be the current application version string.
func Write(path string, book types.BookData, appVersion string) types.SaveResult {
	return WriteWithSnapshots(path, book, appVersion, nil)
}

// WriteWithSnapshots saves the book while preserving its embedded chapter
// history and optionally appending changed chapter snapshots.
func WriteWithSnapshots(path string, book types.BookData, appVersion string, snapshots []types.ChapterSnapshotRequest) types.SaveResult {
	history := historyArchive{Entries: []types.ChapterHistoryEntry{}, Contents: map[string][]byte{}}
	if len(snapshots) > 0 {
		var err error
		history, err = loadHistory(path)
		if err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		addHistorySnapshots(&history, snapshots)
	}
	EnsureBookChapterIDs(&book)

	// Create backup of existing file before overwriting
	if err := backup.Create(path); err != nil {
		// Log but don't fail the save - backup is best-effort
		fmt.Printf("Backup warning: %v\n", err)
	}

	book.Metadata.Modified = time.Now().Format(time.RFC3339)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	type entry struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Subtitle string `json:"subtitle,omitempty"`
		Type     string `json:"type"`
		File     string `json:"file"`
	}
	type manifest struct {
		Version      string                    `json:"version"`
		AppVersion   string                    `json:"app_version"`
		Metadata     types.Metadata            `json:"metadata"`
		FrontMatter  []entry                   `json:"front_matter"`
		Body         []entry                   `json:"body"`
		BackMatter   []entry                   `json:"back_matter"`
		WritingGoals types.WritingGoals        `json:"writing_goals,omitempty"`
		StyleOptions types.WritingStyleOptions `json:"style_options,omitempty"`
		IsIndexed    bool                      `json:"is_indexed,omitempty"`
		LastIndexed  string                    `json:"last_indexed,omitempty"`
	}

	mf := manifest{
		Version:      "2.2",
		AppVersion:   appVersion,
		Metadata:     book.Metadata,
		FrontMatter:  []entry{},
		IsIndexed:    book.IsIndexed,
		LastIndexed:  book.LastIndexed,
		Body:         []entry{},
		BackMatter:   []entry{},
		WritingGoals: book.WritingGoals,
		StyleOptions: book.StyleOptions,
	}

	addEntry := func(name, content string) error {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		return err
	}

	if len(snapshots) == 0 {
		if err := copyHistoryEntries(w, path); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	} else {
		for _, historyEntry := range history.Entries {
			content, ok := history.Contents[historyEntry.File]
			if !ok {
				return types.SaveResult{Success: false, Error: "chapter history content is missing"}
			}
			if err := addEntry(historyEntry.File, string(content)); err != nil {
				return types.SaveResult{Success: false, Error: err.Error()}
			}
		}
	}
	if len(snapshots) > 0 && len(history.Entries) > 0 {
		historyJSON, err := json.MarshalIndent(struct {
			Entries []types.ChapterHistoryEntry `json:"entries"`
		}{Entries: history.Entries}, "", "  ")
		if err != nil {
			return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode chapter history: %v", err)}
		}
		if err := addEntry(historyIndexFile, string(historyJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	if err := addEntry("copyright.html", book.Copyright); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}
	if book.StoryBible.Characters == nil {
		book.StoryBible.Characters = []types.Character{}
	}
	bibleJSON, _ := json.MarshalIndent(book.StoryBible, "", "  ")
	if err := addEntry("story_bible.json", string(bibleJSON)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Analysis is stored separately from the author-owned story bible. It is
	// rebuildable, but persisting it keeps mention locations, relationships,
	// and manual merge/split decisions available after reopening a project.
	analysisJSON, err := json.MarshalIndent(book.Analysis, "", "  ")
	if err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to encode analysis: %v", err)}
	}
	if err := addEntry("analysis.json", string(analysisJSON)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Save beat_sheet.json if there are beats
	if len(book.BeatSheet.Beats) > 0 {
		beatJSON, _ := json.MarshalIndent(book.BeatSheet, "", "  ")
		if err := addEntry("beat_sheet.json", string(beatJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save foreshadowing.json if there are items
	if len(book.ForeshadowingLedger.Items) > 0 {
		foreshadowJSON, _ := json.MarshalIndent(book.ForeshadowingLedger, "", "  ")
		if err := addEntry("foreshadowing.json", string(foreshadowJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save knowledge_matrix.json if there are secrets or entries
	if len(book.KnowledgeMatrix.Secrets) > 0 || len(book.KnowledgeMatrix.Entries) > 0 {
		matrixJSON, _ := json.MarshalIndent(book.KnowledgeMatrix, "", "  ")
		if err := addEntry("knowledge_matrix.json", string(matrixJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	// Save read_aloud_cast.json if cast mode was ever configured
	if book.ReadAloudCast.CastMode || len(book.ReadAloudCast.Voices) > 0 {
		castJSON, _ := json.MarshalIndent(book.ReadAloudCast, "", "  ")
		if err := addEntry("read_aloud_cast.json", string(castJSON)); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
	}

	for i, item := range book.FrontMatter {
		file := fmt.Sprintf("front_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.FrontMatter = append(mf.FrontMatter, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.Body {
		file := fmt.Sprintf("body/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.Body = append(mf.Body, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.BackMatter {
		file := fmt.Sprintf("back_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.BackMatter = append(mf.BackMatter, entry{ID: item.ID, Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}

	manifestBytes, _ := json.MarshalIndent(mf, "", "  ")
	if err := addEntry("manifest.json", string(manifestBytes)); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	if err := w.Close(); err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("failed to finalize archive: %v", err)}
	}

	// Validate the produced archive before letting it near the user's file.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("internal error: produced archive is invalid: %v", err)}
	}
	if _, err := ziputil.ReadNamed(zr.File, "manifest.json", false); err != nil {
		return types.SaveResult{Success: false, Error: "internal error: produced archive is missing manifest.json"}
	}
	if err := ziputil.CheckArchive(zr.File); err != nil {
		return types.SaveResult{Success: false, Error: fmt.Sprintf("archive exceeds safety limits: %v", err)}
	}

	if err := fsutil.WriteFileAtomic(path, buf.Bytes(), 0644); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	return types.SaveResult{Success: true, FilePath: path}
}
