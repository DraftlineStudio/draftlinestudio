package export

import (
	"fmt"
	"strings"
	"time"

	"draftline/internal/types"
)

type epubEntry struct {
	Name string
	Data []byte
}

type epubSection struct {
	ID       string
	FileName string
	Title    string
	Role     SectionRole
	XHTML    string
}

func normalizeEPUBOptions(options types.EPUBOptions) types.EPUBOptions {
	switch strings.ToLower(strings.TrimSpace(options.FontFamily)) {
	case "merriweather", "lato":
		options.FontFamily = strings.ToLower(strings.TrimSpace(options.FontFamily))
	default:
		options.FontFamily = "reader"
	}
	switch strings.ToLower(strings.TrimSpace(options.ParagraphStyle)) {
	case "spaced":
		options.ParagraphStyle = "spaced"
	default:
		options.ParagraphStyle = "indented"
	}
	switch strings.ToLower(strings.TrimSpace(options.TextAlign)) {
	case "left", "justify":
		options.TextAlign = strings.ToLower(strings.TrimSpace(options.TextAlign))
	default:
		options.TextAlign = "reader"
	}
	switch strings.ToLower(strings.TrimSpace(options.ChapterStyle)) {
	case "minimal":
		options.ChapterStyle = "minimal"
	default:
		options.ChapterStyle = "classic"
	}
	switch strings.ToLower(strings.TrimSpace(options.SceneBreakStyle)) {
	case "rule", "space":
		options.SceneBreakStyle = strings.ToLower(strings.TrimSpace(options.SceneBreakStyle))
	default:
		options.SceneBreakStyle = "asterism"
	}
	return options
}

func buildEPUBEntries(doc Document, book types.BookData, options types.EPUBOptions, modified time.Time) ([]epubEntry, []epubEntry) {
	sections := make([]epubSection, 0, len(doc.Sections))
	for index, section := range doc.Sections {
		id := fmt.Sprintf("section-%04d", index+1)
		fileName := id + ".xhtml"
		sections = append(sections, epubSection{
			ID: id, FileName: fileName, Title: section.Title, Role: section.Role,
			XHTML: renderEPUBSection(doc, section, options),
		})
	}

	// An ISBN is declared as the plain number, without the hyphens an author
	// groups it with. That is the form the Editions screen shows, and the screen
	// and the file saying the same thing is the whole point of showing it.
	identifier := "urn:uuid:" + GenerateUUID()
	if isbn := types.NormalizeISBN(book.Metadata.ISBNFor("ebook")); isbn != "" {
		identifier = "urn:isbn:" + isbn
	}
	entries := []epubEntry{
		{Name: "META-INF/container.xml", Data: []byte(epubContainerXML)},
		{Name: "OEBPS/styles/book.css", Data: []byte(renderEPUBCSS(options))},
		{Name: "OEBPS/nav.xhtml", Data: []byte(renderEPUBNav(doc, sections))},
		{Name: "OEBPS/content.opf", Data: []byte(renderEPUBPackage(doc, identifier, sections, options, modified))},
	}
	for _, section := range sections {
		entries = append(entries, epubEntry{Name: "OEBPS/text/" + section.FileName, Data: []byte(section.XHTML)})
	}
	return entries, epubFontEntries(options.FontFamily)
}

const epubContainerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

func renderEPUBPackage(doc Document, identifier string, sections []epubSection, options types.EPUBOptions, modified time.Time) string {
	var manifest strings.Builder
	var spine strings.Builder
	for _, section := range sections {
		fmt.Fprintf(&manifest, "    <item id=\"%s\" href=\"text/%s\" media-type=\"application/xhtml+xml\"/>\n", section.ID, section.FileName)
		fmt.Fprintf(&spine, "    <itemref idref=\"%s\"/>\n", section.ID)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid" xml:lang="%s">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
%s  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="css" href="styles/book.css" media-type="text/css"/>
%s%s  </manifest>
  <spine>
%s  </spine>
</package>`, EscapeXML(doc.Language), epubMetadata(doc, identifier, modified), manifest.String(), epubFontManifest(options.FontFamily), spine.String())
}

// epubMetadata writes only the Dublin Core a book actually carries. An empty
// element is not neutral: EPUBCheck reports it, and a reader can show a blank
// author line where it would otherwise show nothing at all.
func epubMetadata(doc Document, identifier string, modified time.Time) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, "    "+format+"\n", args...) }

	line(`<dc:identifier id="uid">%s</dc:identifier>`, EscapeXML(identifier))
	line(`<dc:title id="title">%s</dc:title>`, EscapeXML(doc.Title))
	if doc.Subtitle != "" {
		line(`<meta refines="#title" property="title-type">main</meta>`)
		line(`<dc:title id="subtitle">%s</dc:title>`, EscapeXML(doc.Subtitle))
		line(`<meta refines="#subtitle" property="title-type">subtitle</meta>`)
	}
	if doc.Author != "" {
		line(`<dc:creator id="creator">%s</dc:creator>`, EscapeXML(doc.Author))
		line(`<meta refines="#creator" property="role" scheme="marc:relators">aut</meta>`)
	}
	if doc.Contributors != "" {
		line(`<dc:contributor>%s</dc:contributor>`, EscapeXML(doc.Contributors))
	}
	if doc.Publisher != "" {
		line(`<dc:publisher>%s</dc:publisher>`, EscapeXML(doc.Publisher))
	}
	if doc.Description != "" {
		line(`<dc:description>%s</dc:description>`, EscapeXML(doc.Description))
	}
	for _, subject := range doc.Subjects {
		line(`<dc:subject>%s</dc:subject>`, EscapeXML(subject))
	}
	if doc.SeriesName != "" {
		line(`<meta property="belongs-to-collection" id="series">%s</meta>`, EscapeXML(doc.SeriesName))
		line(`<meta refines="#series" property="collection-type">series</meta>`)
		if doc.SeriesNumber != "" {
			line(`<meta refines="#series" property="group-position">%s</meta>`, EscapeXML(doc.SeriesNumber))
		}
	}
	line(`<dc:language>%s</dc:language>`, EscapeXML(doc.Language))
	line(`<meta property="dcterms:modified">%s</meta>`, modified.Format("2006-01-02T15:04:05Z"))
	return b.String()
}

func renderEPUBNav(doc Document, sections []epubSection) string {
	var items strings.Builder
	for _, section := range sections {
		fmt.Fprintf(&items, "      <li><a href=\"text/%s\">%s</a></li>\n", section.FileName, EscapeXML(displaySectionTitle(section.Title)))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="%s" lang="%s">
<head><meta charset="UTF-8"/><title>Contents</title><link rel="stylesheet" type="text/css" href="styles/book.css"/></head>
<body class="navigation"><nav epub:type="toc" id="toc"><h1>Contents</h1><ol>
%s    </ol></nav></body>
</html>`, EscapeXML(doc.Language), EscapeXML(doc.Language), items.String())
}

func displaySectionTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Untitled section"
	}
	return strings.TrimSpace(title)
}
