package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)


// ImportEPUBDialog shows file picker for EPUB files and imports the selected file
func (a *App) ImportEPUBDialog() types.ImportResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Import EPUB",
		DefaultDirectory: a.settings.DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "EPUB Files (*.epub)", Pattern: "*.epub"},
		},
	})
	if err != nil {
		return types.ImportResult{Success: false, Error: err.Error()}
	}
	if path == "" {
		return types.ImportResult{Success: false, Error: "cancelled"}
	}
	return a.ImportEPUB(path)
}

// ImportEPUB parses an EPUB file and converts it to types.BookData
func (a *App) ImportEPUB(path string) types.ImportResult {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to open EPUB: %v", err)}
	}
	defer func() { _ = r.Close() }()

	// Step 1: Find the OPF file via container.xml
	opfPath, err := findOPFPath(r)
	if err != nil {
		return types.ImportResult{Success: false, Error: err.Error()}
	}

	// Step 2: Parse the OPF file for metadata and spine
	opfData, err := readEpubEntry(r, opfPath)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to read OPF: %v", err)}
	}

	opfDir := filepath.Dir(opfPath)
	metadata, spine, manifest, err := parseOPF(opfData)
	if err != nil {
		return types.ImportResult{Success: false, Error: err.Error()}
	}

	// Step 3: Extract chapters in spine order
	var chapters []types.ChapterItem
	chapterNum := 1

	for _, itemRef := range spine {
		// Find the item in manifest
		item, ok := manifest[itemRef.IDRef]
		if !ok {
			continue
		}

		// Skip non-content items (CSS, images, etc.)
		if !strings.Contains(item.MediaType, "html") && !strings.Contains(item.MediaType, "xml") {
			continue
		}

		// Read the content file
		contentPath := item.Href
		if opfDir != "." && opfDir != "" {
			contentPath = opfDir + "/" + item.Href
		}
		contentPath = strings.ReplaceAll(contentPath, "\\", "/")

		contentBytes, err := readEpubEntry(r, contentPath)
		if err != nil {
			continue
		}

		// Extract title and body from XHTML
		title, body := parseXHTMLContent(string(contentBytes))
		if strings.TrimSpace(body) == "" {
			continue
		}

		// Use extracted title or generate one
		if title == "" {
			title = fmt.Sprintf("Chapter %d", chapterNum)
		}

		chapters = append(chapters, types.ChapterItem{
			Title:   title,
			Type:    "Chapter",
			Content: body,
		})
		chapterNum++
	}

	if len(chapters) == 0 {
		return types.ImportResult{Success: false, Error: "No readable chapters found in EPUB"}
	}

	// Build the types.BookData
	now := time.Now().Format(time.RFC3339)
	book := types.BookData{
		Version: "2.0",
		Metadata: types.Metadata{
			Title:     metadata.Title,
			Author:    metadata.Creator,
			Publisher: metadata.Publisher,
			Created:   now,
			Modified:  now,
		},
		Copyright:   "",
		FrontMatter: []types.ChapterItem{},
		Body:        chapters,
		BackMatter:  []types.ChapterItem{},
		StoryBible:  types.StoryBible{Characters: []types.Character{}},
	}

	// Use filename as fallback title
	if book.Metadata.Title == "" {
		book.Metadata.Title = strings.TrimSuffix(filepath.Base(path), ".epub")
	}

	return types.ImportResult{Success: true, Book: book}
}

// readEpubEntry reads a file from the EPUB ZIP with case-insensitive path matching
func readEpubEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, f := range r.File {
		fName := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.EqualFold(fName, name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			return data, err
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// findOPFPath reads container.xml to locate the OPF file
func findOPFPath(r *zip.ReadCloser) (string, error) {
	containerData, err := readEpubEntry(r, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("invalid EPUB: missing META-INF/container.xml")
	}

	var container struct {
		RootFiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}

	if err := xml.Unmarshal(containerData, &container); err != nil {
		return "", fmt.Errorf("failed to parse container.xml: %v", err)
	}

	if len(container.RootFiles) == 0 {
		return "", fmt.Errorf("no rootfile found in container.xml")
	}

	return container.RootFiles[0].FullPath, nil
}

// OPF metadata structure
type opfMetadata struct {
	Title     string
	Creator   string
	Publisher string
}

type opfManifestItem struct {
	ID        string
	Href      string
	MediaType string
}

type opfSpineItem struct {
	IDRef string
}

// parseOPF extracts metadata, manifest, and spine from the OPF file
func parseOPF(data []byte) (opfMetadata, []opfSpineItem, map[string]opfManifestItem, error) {
	var opf struct {
		Metadata struct {
			Title     []string `xml:"title"`
			Creator   []string `xml:"creator"`
			Publisher []string `xml:"publisher"`
		} `xml:"metadata"`
		Manifest struct {
			Items []struct {
				ID        string `xml:"id,attr"`
				Href      string `xml:"href,attr"`
				MediaType string `xml:"media-type,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
		Spine struct {
			ItemRefs []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}

	if err := xml.Unmarshal(data, &opf); err != nil {
		return opfMetadata{}, nil, nil, fmt.Errorf("failed to parse OPF: %v", err)
	}

	meta := opfMetadata{}
	if len(opf.Metadata.Title) > 0 {
		meta.Title = opf.Metadata.Title[0]
	}
	if len(opf.Metadata.Creator) > 0 {
		meta.Creator = opf.Metadata.Creator[0]
	}
	if len(opf.Metadata.Publisher) > 0 {
		meta.Publisher = opf.Metadata.Publisher[0]
	}

	manifest := make(map[string]opfManifestItem)
	for _, item := range opf.Manifest.Items {
		manifest[item.ID] = opfManifestItem{
			ID:        item.ID,
			Href:      item.Href,
			MediaType: item.MediaType,
		}
	}

	var spine []opfSpineItem
	for _, ref := range opf.Spine.ItemRefs {
		spine = append(spine, opfSpineItem{IDRef: ref.IDRef})
	}

	return meta, spine, manifest, nil
}

// parseXHTMLContent extracts the title and body HTML from XHTML content
func parseXHTMLContent(xhtml string) (title string, body string) {
	// Extract title from <title> tag
	titleRe := regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)
	if match := titleRe.FindStringSubmatch(xhtml); len(match) > 1 {
		title = strings.TrimSpace(match[1])
	}

	// Try to extract from <h1>, <h2>, or <h3> if no title
	if title == "" {
		headingRe := regexp.MustCompile(`(?i)<h[123][^>]*>([^<]*)</h[123]>`)
		if match := headingRe.FindStringSubmatch(xhtml); len(match) > 1 {
			title = strings.TrimSpace(match[1])
		}
	}

	// Extract body content
	bodyRe := regexp.MustCompile(`(?is)<body[^>]*>(.*)</body>`)
	if match := bodyRe.FindStringSubmatch(xhtml); len(match) > 1 {
		body = match[1]
	} else {
		// No body tag, use the whole content
		body = xhtml
	}

	// Clean up the body HTML
	body = cleanHTML(body)

	return title, body
}

// cleanHTML normalizes HTML content for the editor
func cleanHTML(html string) string {
	// Remove XML declarations and doctype
	html = regexp.MustCompile(`(?i)<\?xml[^>]*\?>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)<!DOCTYPE[^>]*>`).ReplaceAllString(html, "")

	// Remove HTML namespace declarations
	html = regexp.MustCompile(`\s+xmlns[^=]*="[^"]*"`).ReplaceAllString(html, "")

	// Remove epub:type attributes
	html = regexp.MustCompile(`\s+epub:[^=]*="[^"]*"`).ReplaceAllString(html, "")

	// Remove class and id attributes (optional - keeps HTML cleaner)
	// html = regexp.MustCompile(`\s+(class|id)="[^"]*"`).ReplaceAllString(html, "")

	// Convert common EPUB elements to standard HTML
	html = regexp.MustCompile(`(?i)<section[^>]*>`).ReplaceAllString(html, "<div>")
	html = regexp.MustCompile(`(?i)</section>`).ReplaceAllString(html, "</div>")

	// Remove empty paragraphs
	html = regexp.MustCompile(`(?i)<p[^>]*>\s*</p>`).ReplaceAllString(html, "")

	// Normalize whitespace
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")

	// Trim
	html = strings.TrimSpace(html)

	return html
}

// ═══════════════════════════════════════════════════════════════════════════════
// DOCX IMPORT
// ═══════════════════════════════════════════════════════════════════════════════

// ImportDOCXDialog shows file picker for DOCX files and imports the selected file
func (a *App) ImportDOCXDialog() types.ImportResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Import Word Document",
		DefaultDirectory: a.settings.DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Word Documents (*.docx)", Pattern: "*.docx"},
		},
	})
	if err != nil {
		return types.ImportResult{Success: false, Error: err.Error()}
	}
	if path == "" {
		return types.ImportResult{Success: false, Error: "cancelled"}
	}
	return a.ImportDOCX(path)
}

// ImportDOCX parses a DOCX file and converts it to types.BookData
func (a *App) ImportDOCX(path string) types.ImportResult {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to open DOCX: %v", err)}
	}
	defer func() { _ = r.Close() }()

	// Extract metadata from docProps/core.xml
	meta := extractDOCXMetadata(r)

	// Parse document.xml for content
	docData, err := readDocxEntry(r, "word/document.xml")
	if err != nil {
		return types.ImportResult{Success: false, Error: "Invalid DOCX: missing word/document.xml"}
	}

	// Parse the document XML
	chapters, err := parseDOCXDocument(docData)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to parse document: %v", err)}
	}

	if len(chapters) == 0 {
		return types.ImportResult{Success: false, Error: "No content found in document"}
	}

	// Build the types.BookData
	now := time.Now().Format(time.RFC3339)
	book := types.BookData{
		Version: "2.0",
		Metadata: types.Metadata{
			Title:     meta.title,
			Author:    meta.author,
			Publisher: "",
			Created:   now,
			Modified:  now,
		},
		Copyright:   "",
		FrontMatter: []types.ChapterItem{},
		Body:        chapters,
		BackMatter:  []types.ChapterItem{},
		StoryBible:  types.StoryBible{Characters: []types.Character{}},
	}

	// Use filename as fallback title
	if book.Metadata.Title == "" {
		book.Metadata.Title = strings.TrimSuffix(filepath.Base(path), ".docx")
	}

	return types.ImportResult{Success: true, Book: book}
}

// readDocxEntry reads a file from the DOCX ZIP
func readDocxEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range r.File {
		if strings.EqualFold(f.Name, name) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			return data, err
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

type docxMeta struct {
	title  string
	author string
}

// extractDOCXMetadata reads metadata from docProps/core.xml
func extractDOCXMetadata(r *zip.ReadCloser) docxMeta {
	meta := docxMeta{}

	coreData, err := readDocxEntry(r, "docProps/core.xml")
	if err != nil {
		return meta
	}

	// Parse core.xml - it uses Dublin Core namespace
	var core struct {
		Title   string `xml:"title"`
		Creator string `xml:"creator"`
	}

	if xml.Unmarshal(coreData, &core) == nil {
		meta.title = core.Title
		meta.author = core.Creator
	}

	return meta
}

// parseDOCXDocument parses word/document.xml and extracts chapters
func parseDOCXDocument(data []byte) ([]types.ChapterItem, error) {
	// DOCX uses the WordprocessingML namespace
	// Structure: <w:document><w:body><w:p>...</w:p>...</w:body></w:document>

	type RunProps struct {
		Bold      *struct{} `xml:"b"`
		Italic    *struct{} `xml:"i"`
		Underline *struct{} `xml:"u"`
	}

	type Run struct {
		Props *RunProps `xml:"rPr"`
		Text  []string  `xml:"t"`
	}

	type ParagraphProps struct {
		Style struct {
			Val string `xml:"val,attr"`
		} `xml:"pStyle"`
	}

	type Paragraph struct {
		Props *ParagraphProps `xml:"pPr"`
		Runs  []Run           `xml:"r"`
	}

	type Body struct {
		Paragraphs []Paragraph `xml:"p"`
	}

	type Document struct {
		Body Body `xml:"body"`
	}

	var doc Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse document XML: %v", err)
	}

	// Group paragraphs into chapters based on heading styles
	var chapters []types.ChapterItem
	currentChapter := types.ChapterItem{Title: "Chapter 1", Type: "Chapter", Content: ""}
	chapterNum := 1
	hasContent := false

	for _, para := range doc.Body.Paragraphs {
		// Check if this is a heading
		isHeading := false
		styleVal := ""
		if para.Props != nil {
			styleVal = strings.ToLower(para.Props.Style.Val)
			isHeading = strings.HasPrefix(styleVal, "heading") ||
				styleVal == "title" ||
				strings.HasPrefix(styleVal, "heading1") ||
				strings.HasPrefix(styleVal, "heading2")
		}

		// Extract text from runs
		paraText := ""
		paraHTML := ""
		for _, run := range para.Runs {
			runText := strings.Join(run.Text, "")
			if runText == "" {
				continue
			}

			// Apply formatting
			formatted := runText
			if run.Props != nil {
				if run.Props.Bold != nil {
					formatted = "<strong>" + formatted + "</strong>"
				}
				if run.Props.Italic != nil {
					formatted = "<em>" + formatted + "</em>"
				}
				if run.Props.Underline != nil {
					formatted = "<u>" + formatted + "</u>"
				}
			}
			paraText += runText
			paraHTML += formatted
		}

		paraText = strings.TrimSpace(paraText)
		paraHTML = strings.TrimSpace(paraHTML)

		if paraText == "" {
			continue
		}

		// If this is a heading, start a new chapter
		if isHeading && (styleVal == "heading1" || styleVal == "title" ||
			strings.Contains(styleVal, "heading 1") || strings.Contains(styleVal, "heading1")) {
			// Save current chapter if it has content
			if hasContent {
				chapters = append(chapters, currentChapter)
			}
			// Start new chapter
			chapterNum++
			currentChapter = types.ChapterItem{
				Title:   paraText,
				Type:    "Chapter",
				Content: "",
			}
			hasContent = false
		} else {
			// Add paragraph to current chapter
			if currentChapter.Content != "" {
				currentChapter.Content += "\n"
			}
			currentChapter.Content += "<p>" + paraHTML + "</p>"
			hasContent = true
		}
	}

	// Don't forget the last chapter
	if hasContent {
		chapters = append(chapters, currentChapter)
	}

	// If no chapters were created from headings, treat the whole doc as one chapter
	if len(chapters) == 0 && currentChapter.Content != "" {
		chapters = append(chapters, currentChapter)
	}

	return chapters, nil
}

