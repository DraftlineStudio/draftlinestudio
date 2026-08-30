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
	// Create backup of existing file before overwriting
	if err := backup.Create(path); err != nil {
		// Log but don't fail the save - backup is best-effort
		fmt.Printf("Backup warning: %v\n", err)
	}

	book.Metadata.Modified = time.Now().Format(time.RFC3339)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	type entry struct {
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
		Version:      "2.1",
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

	for i, item := range book.FrontMatter {
		file := fmt.Sprintf("front_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.FrontMatter = append(mf.FrontMatter, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.Body {
		file := fmt.Sprintf("body/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.Body = append(mf.Body, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
	}
	for i, item := range book.BackMatter {
		file := fmt.Sprintf("back_matter/%03d.html", i)
		if err := addEntry(file, item.Content); err != nil {
			return types.SaveResult{Success: false, Error: err.Error()}
		}
		mf.BackMatter = append(mf.BackMatter, entry{Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, File: file})
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

	if err := fsutil.WriteFileAtomic(path, buf.Bytes(), 0644); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	return types.SaveResult{Success: true, FilePath: path}
}
