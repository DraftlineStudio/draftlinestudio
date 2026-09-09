package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"draftline/internal/types"
	"draftline/internal/ziputil"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ImportEPUBDialog shows file picker for EPUB files and imports the selected file
func (a *App) ImportEPUBDialog() types.ImportResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Import EPUB",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
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

// safeImport converts a panic anywhere inside an importer into a failed
// ImportResult instead of killing the process: main.go binds App directly, so
// an unrecovered panic in a bound method takes down the whole Wails app.
func safeImport(kind string, fn func() types.ImportResult) (res types.ImportResult) {
	defer func() {
		if r := recover(); r != nil {
			res = types.ImportResult{
				Success: false,
				Error:   fmt.Sprintf("%s import failed unexpectedly: %v", kind, r),
			}
		}
	}()
	return fn()
}

// maxSpineDocBytes caps a single spine document's raw size. Anything larger is
// not a text chapter and would stall the sanitizer and the JSON bridge.
const maxSpineDocBytes = 8 << 20

// ImportEPUB parses an EPUB file and converts it to types.BookData
func (a *App) ImportEPUB(path string) types.ImportResult {
	return safeImport("EPUB", func() types.ImportResult { return a.importEPUB(path) })
}

func (a *App) importEPUB(path string) types.ImportResult {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to open EPUB: %v", err)}
	}
	defer func() { _ = r.Close() }()

	if err := ziputil.CheckArchive(r.File); err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Refusing to import EPUB: %v", err)}
	}

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

	// Step 3: Extract sections in spine order, routed to their destination.
	front := []types.ChapterItem{}
	body := []types.ChapterItem{}
	back := []types.ChapterItem{}
	copyright := ""
	var warnings []string
	imagesDropped := 0
	totalHTML := 0
	capReached := false

	for _, itemRef := range spine {
		// Find the item in manifest
		item, ok := manifest[itemRef.IDRef]
		if !ok {
			continue
		}

		// Only real chapter documents. A contains-check let image/svg+xml
		// covers and other XML resources through as garbage chapters.
		if !isImportableSpineType(item.MediaType) {
			continue
		}

		// EPUB 3 navigation documents are a machine-readable TOC, not content.
		if hasManifestProperty(item.Properties, "nav") {
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
		if len(contentBytes) > maxSpineDocBytes {
			warnings = append(warnings, fmt.Sprintf(
				"Skipped %s: document is larger than %d MB", item.Href, maxSpineDocBytes>>20))
			continue
		}

		// Decode, parse, and sanitize the document down to the editor's
		// dialect before it can reach TipTap.
		title, blocks, imgDropped, err := parseSpineDoc(contentBytes)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Skipped %s: %v", item.Href, err))
			continue
		}
		imagesDropped += imgDropped
		if len(blocks) == 0 {
			continue
		}

		// EPUBs commonly repeat the book title in every spine item's <title>;
		// that is not a chapter name. Untitled sections are named after
		// routing (type label or "Chapter N").
		if bookTitle := strings.TrimSpace(metadata.Title); bookTitle != "" &&
			strings.EqualFold(strings.TrimSpace(title), bookTitle) {
			title = ""
		}

		parts, partWarnings := assembleChapters(title, blocks)
		warnings = append(warnings, partWarnings...)
		for _, part := range parts {
			if len(front)+len(body)+len(back) >= maxImportedChapters {
				warnings = append(warnings, fmt.Sprintf(
					"Import stopped at %d chapters; remaining spine documents were skipped", maxImportedChapters))
				capReached = true
				break
			}
			totalHTML += len(part.Content)
			if totalHTML > maxImportedBookHTML {
				warnings = append(warnings, fmt.Sprintf(
					"Import stopped after %d MB of content; remaining spine documents were skipped", maxImportedBookHTML>>20))
				capReached = true
				break
			}
			route, typeLabel := routeImportedSection(part.Title, plainImportedText(part.Content))
			untitled := strings.TrimSpace(part.Title) == ""
			switch route {
			case routeSkip:
				continue
			case routeCopyright:
				// The first copyright page fills the book's dedicated
				// copyright section; any further ones stay visible up front.
				if copyright == "" {
					copyright = part.Content
					continue
				}
				part.Type = "Copyright"
				if untitled {
					part.Title = "Copyright"
				}
				front = append(front, part)
			case routeFront:
				part.Type = typeLabel
				if untitled {
					part.Title = typeLabel
				}
				front = append(front, part)
			case routeBack:
				part.Type = typeLabel
				if untitled {
					part.Title = typeLabel
				}
				back = append(back, part)
			default:
				part.Type = typeLabel
				if untitled {
					part.Title = fmt.Sprintf("Chapter %d", len(body)+1)
				}
				body = append(body, part)
			}
		}
		if capReached {
			break
		}
	}

	if len(front)+len(body)+len(back) == 0 && copyright == "" {
		return types.ImportResult{Success: false, Error: "No readable chapters found in EPUB"}
	}

	if imagesDropped > 0 {
		noun := "images were"
		if imagesDropped == 1 {
			noun = "image was"
		}
		warnings = append(warnings, fmt.Sprintf(
			"%d %s removed during import (Draftline does not support embedded images)", imagesDropped, noun))
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
		Copyright:   copyright,
		FrontMatter: front,
		Body:        body,
		BackMatter:  back,
		StoryBible:  types.StoryBible{Characters: []types.Character{}},
	}

	// Use filename as fallback title
	if book.Metadata.Title == "" {
		book.Metadata.Title = strings.TrimSuffix(filepath.Base(path), ".epub")
	}

	// The imported book is a NEW, unsaved project. Clear the session's current
	// file so Save cannot silently overwrite whatever project was open before
	// the import — Ctrl+S on an imported book must go through Save As.
	a.setCurrentFile("")

	return types.ImportResult{Success: true, Book: book, Warnings: warnings}
}

// isImportableSpineType reports whether a spine item's media type is a
// chapter document (XHTML/HTML) rather than an image, stylesheet, or other
// resource that happens to be XML.
func isImportableSpineType(mediaType string) bool {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "application/xhtml+xml", "text/html":
		return true
	default:
		return false
	}
}

// hasManifestProperty reports whether a space-separated OPF properties
// attribute contains the given token (e.g. "nav", "cover-image").
func hasManifestProperty(properties, token string) bool {
	for p := range strings.FieldsSeq(properties) {
		if strings.EqualFold(p, token) {
			return true
		}
	}
	return false
}

// readEpubEntry reads a file from the EPUB ZIP with case-insensitive path
// matching, enforcing size limits on the decompressed content.
func readEpubEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, f := range r.File {
		fName := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.EqualFold(fName, name) {
			return ziputil.ReadEntry(f)
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
	ID         string
	Href       string
	MediaType  string
	Properties string
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
				ID         string `xml:"id,attr"`
				Href       string `xml:"href,attr"`
				MediaType  string `xml:"media-type,attr"`
				Properties string `xml:"properties,attr"`
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
			ID:         item.ID,
			Href:       item.Href,
			MediaType:  item.MediaType,
			Properties: item.Properties,
		}
	}

	var spine []opfSpineItem
	for _, ref := range opf.Spine.ItemRefs {
		spine = append(spine, opfSpineItem{IDRef: ref.IDRef})
	}

	return meta, spine, manifest, nil
}

func plainImportedText(fragment string) string {
	withoutTags := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(fragment, "")
	return strings.TrimSpace(html.UnescapeString(withoutTags))
}

// sectionRoute is the destination of an imported section within BookData.
type sectionRoute int

const (
	routeBody sectionRoute = iota
	routeFront
	routeBack
	routeCopyright
	routeSkip
)

// routeImportedSection classifies a section by its title (and, for unlabeled
// sections, its leading text) and returns its destination plus the canonical
// Type label the frontend's section vocabulary uses.
func routeImportedSection(title, plainText string) (sectionRoute, string) {
	normalized := strings.ToLower(strings.TrimSpace(title))
	normalized = strings.Trim(normalized, " .:;,-–—")
	prefix := strings.ToLower(plainText)
	if len(prefix) > 500 {
		prefix = prefix[:500]
	}

	switch normalized {
	case "cover", "contents", "table of contents", "toc", "index", "landmarks", "guide", "nav", "navigation":
		return routeSkip, ""
	case "copyright", "copyright page":
		return routeCopyright, ""
	case "title page", "half title", "half title page":
		return routeFront, "Title Page"
	case "dedication":
		return routeFront, "Dedication"
	case "epigraph":
		return routeFront, "Epigraph"
	case "foreword":
		return routeFront, "Foreword"
	case "preface":
		return routeFront, "Preface"
	case "introduction":
		return routeFront, "Introduction"
	case "prologue":
		return routeFront, "Prologue"
	case "author's note", "authors note", "author's notes", "a note on the text", "note to the reader":
		return routeFront, "Author's Note"
	case "epilogue":
		return routeBack, "Epilogue"
	case "afterword":
		return routeBack, "Afterword"
	case "appendix", "notes", "endnotes":
		return routeBack, "Appendix"
	case "acknowledgments", "acknowledgements":
		return routeBack, "Acknowledgments"
	case "about the author":
		return routeBack, "About the Author"
	case "glossary":
		return routeBack, "Glossary"
	case "colophon":
		return routeBack, "Colophon"
	}
	if strings.HasPrefix(normalized, "also by") {
		return routeFront, "Also By"
	}

	// Content-prefix heuristics for sections whose title is unhelpful.
	for _, marker := range []string{"acknowledgments", "acknowledgements"} {
		if strings.HasPrefix(prefix, marker) {
			return routeBack, "Acknowledgments"
		}
	}
	if strings.HasPrefix(prefix, "table of contents") {
		return routeSkip, ""
	}
	if strings.Contains(prefix, "all rights reserved") ||
		(strings.Contains(prefix, "publishing plc") && strings.Contains(prefix, "trademark")) {
		return routeCopyright, ""
	}

	return routeBody, "Chapter"
}

// ═══════════════════════════════════════════════════════════════════════════════
// DOCX IMPORT
// ═══════════════════════════════════════════════════════════════════════════════

// ImportDOCXDialog shows file picker for DOCX files and imports the selected file
func (a *App) ImportDOCXDialog() types.ImportResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Import Word Document",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
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
	return safeImport("DOCX", func() types.ImportResult { return a.importDOCX(path) })
}

func (a *App) importDOCX(path string) types.ImportResult {
	r, err := zip.OpenReader(path)
	if err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Failed to open DOCX: %v", err)}
	}
	defer func() { _ = r.Close() }()

	if err := ziputil.CheckArchive(r.File); err != nil {
		return types.ImportResult{Success: false, Error: fmt.Sprintf("Refusing to import DOCX: %v", err)}
	}

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

	// See ImportEPUB: an imported book must never inherit the previous
	// project's save target.
	a.setCurrentFile("")

	return types.ImportResult{Success: true, Book: book}
}

// readDocxEntry reads a file from the DOCX ZIP, enforcing size limits.
func readDocxEntry(r *zip.ReadCloser, name string) ([]byte, error) {
	return ziputil.ReadNamed(r.File, name, true)
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
