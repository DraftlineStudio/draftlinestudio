// Package book provides book lifecycle operations (open, save).
package book

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"

	"draftline/internal/types"
)

// ReadZipEntry reads a named entry from a ZIP archive.
func ReadZipEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			return data, err
		}
	}
	return nil, fmt.Errorf("entry %q not found", name)
}

// Open reads a .draftline file and returns the BookData.
func Open(path string) (types.BookData, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.BookData{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = r.Close() }()

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
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"front_matter,omitempty"`
		Body []struct {
			Title    string `json:"title"`
			Subtitle string `json:"subtitle,omitempty"`
			Type     string `json:"type"`
			File     string `json:"file"`
		} `json:"body,omitempty"`
		BackMatter []struct {
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

	book := types.BookData{
		Version:      "2.0",
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
			content, _ := ReadZipEntry(r, ch.File)
			book.Body = append(book.Body, types.ChapterItem{
				Title:   ch.Title,
				Type:    "Chapter",
				Content: string(content),
			})
		}
		return book, nil
	}

	// v2.0
	copyright, _ := ReadZipEntry(r, "copyright.html")
	book.Copyright = string(copyright)

	for _, item := range raw.FrontMatter {
		content, _ := ReadZipEntry(r, item.File)
		book.FrontMatter = append(book.FrontMatter, types.ChapterItem{
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.Body {
		content, _ := ReadZipEntry(r, item.File)
		book.Body = append(book.Body, types.ChapterItem{
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
		})
	}
	for _, item := range raw.BackMatter {
		content, _ := ReadZipEntry(r, item.File)
		book.BackMatter = append(book.BackMatter, types.ChapterItem{
			Title: item.Title, Subtitle: item.Subtitle, Type: item.Type, Content: string(content),
		})
	}

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
