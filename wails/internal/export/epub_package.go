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

// Where the cover lives inside the package, and what the manifest calls it.
// The id is the one EPUB 2 reading systems look for in <meta name="cover">,
// and the one EPUB 3 marks properties="cover-image".
const (
	epubCoverItemID  = "cover-image"
	epubCoverPageID  = "cover-page"
	epubCoverPageDoc = "cover.xhtml"
	epubNCXName      = "toc.ncx"
)

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

func buildEPUBEntries(doc Document, book types.BookData, options types.EPUBOptions, cover *CoverArt, modified time.Time) ([]epubEntry, []epubEntry) {
	profile := documentEPUBProfile(doc)
	sections := make([]epubSection, 0, len(doc.Sections))
	for index, section := range doc.Sections {
		id := fmt.Sprintf("section-%04d", index+1)
		fileName := id + ".xhtml"
		sections = append(sections, epubSection{
			ID: id, FileName: fileName, Title: section.Title, Role: section.Role,
			XHTML: renderEPUBSection(doc, section, options),
		})
	}

	identifier := epubIdentifier(doc, book)
	entries := []epubEntry{
		{Name: "META-INF/container.xml", Data: []byte(epubContainerXML)},
		{Name: "OEBPS/styles/book.css", Data: []byte(renderEPUBCSS(options))},
		{Name: "OEBPS/nav.xhtml", Data: []byte(renderEPUBNav(doc, sections))},
		{Name: "OEBPS/content.opf", Data: []byte(renderEPUBPackage(doc, identifier, sections, options, cover, modified))},
	}
	// EPUB 2 has no navigation document: its table of contents is an NCX, and
	// a reading system of that vintage finds nothing without one.
	if !profile.Three {
		entries = append(entries, epubEntry{Name: "OEBPS/" + epubNCXName, Data: []byte(renderEPUBNCX(doc, identifier, sections))})
	}
	if cover.Usable() {
		entries = append(entries,
			epubEntry{Name: "OEBPS/images/" + cover.FileName, Data: cover.Data},
			epubEntry{Name: "OEBPS/text/" + epubCoverPageDoc, Data: []byte(renderEPUBCoverPage(doc, cover))},
		)
	}
	for _, section := range sections {
		entries = append(entries, epubEntry{Name: "OEBPS/text/" + section.FileName, Data: []byte(section.XHTML)})
	}
	return entries, epubFontEntries(options.FontFamily)
}

// epubIdentifier is what the package declares itself as.
//
// A registered edition is identified by its own ISBN, as the plain number
// without the hyphens an author groups it with — that is the form the Editions
// screen shows, and the screen and the file saying the same thing is the whole
// point of showing it. A book exported from scratch falls back to the eBook
// ISBN in the book's own identifier list, and then to a random UUID.
func epubIdentifier(doc Document, book types.BookData) string {
	if doc.Edition != nil && doc.Edition.Identifier != "" {
		return doc.Edition.Identifier
	}
	if isbn := types.NormalizeISBN(book.Metadata.ISBNFor("ebook")); isbn != "" {
		return "urn:isbn:" + isbn
	}
	return "urn:uuid:" + GenerateUUID()
}

const epubContainerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

func renderEPUBPackage(doc Document, identifier string, sections []epubSection, options types.EPUBOptions, cover *CoverArt, modified time.Time) string {
	profile := documentEPUBProfile(doc)
	var manifest strings.Builder
	var spine strings.Builder

	// The navigation document. EPUB 3 marks it in the manifest and leaves it
	// out of the reading order; EPUB 2 has no properties attribute at all, so
	// the same file ships as a plain document beside the NCX.
	if profile.Three {
		manifest.WriteString("    <item id=\"nav\" href=\"nav.xhtml\" media-type=\"application/xhtml+xml\" properties=\"nav\"/>\n")
	} else {
		manifest.WriteString("    <item id=\"nav\" href=\"nav.xhtml\" media-type=\"application/xhtml+xml\"/>\n")
		fmt.Fprintf(&manifest, "    <item id=\"ncx\" href=\"%s\" media-type=\"application/x-dtbncx+xml\"/>\n", epubNCXName)
	}
	manifest.WriteString("    <item id=\"css\" href=\"styles/book.css\" media-type=\"text/css\"/>\n")

	if cover.Usable() {
		properties := ""
		if profile.Three {
			properties = ` properties="cover-image"`
		}
		fmt.Fprintf(&manifest, "    <item id=\"%s\" href=\"images/%s\" media-type=\"%s\"%s/>\n",
			epubCoverItemID, EscapeXML(cover.FileName), EscapeXML(cover.MediaType), properties)
		fmt.Fprintf(&manifest, "    <item id=\"%s\" href=\"text/%s\" media-type=\"application/xhtml+xml\"/>\n",
			epubCoverPageID, epubCoverPageDoc)
		// The cover page is the first thing in the reading order, which is
		// what puts the artwork on screen when the book is opened.
		fmt.Fprintf(&spine, "    <itemref idref=\"%s\"/>\n", epubCoverPageID)
	}

	for _, section := range sections {
		fmt.Fprintf(&manifest, "    <item id=\"%s\" href=\"text/%s\" media-type=\"application/xhtml+xml\"/>\n", section.ID, section.FileName)
		fmt.Fprintf(&spine, "    <itemref idref=\"%s\"/>\n", section.ID)
	}

	spineAttrs := ""
	if !profile.Three {
		spineAttrs = ` toc="ncx"`
	}
	guide := ""
	if !profile.Three && cover.Usable() {
		// EPUB 2 reading systems find the cover through the guide, not through
		// a manifest property.
		guide = fmt.Sprintf("\n  <guide>\n    <reference type=\"cover\" title=\"Cover\" href=\"text/%s\"/>\n  </guide>", epubCoverPageDoc)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="%s" unique-identifier="uid" xml:lang="%s">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
%s  </metadata>
  <manifest>
%s%s  </manifest>
  <spine%s>
%s  </spine>%s
</package>`,
		profile.PackageVersion, EscapeXML(doc.Language),
		epubMetadata(doc, identifier, cover, modified),
		manifest.String(), epubFontManifest(options.FontFamily),
		spineAttrs, spine.String(), guide)
}

// epubMetadata writes only the Dublin Core a book actually carries. An empty
// element is not neutral: EPUBCheck reports it, and a reader can show a blank
// author line where it would otherwise show nothing at all.
func epubMetadata(doc Document, identifier string, cover *CoverArt, modified time.Time) string {
	profile := documentEPUBProfile(doc)
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, "    "+format+"\n", args...) }

	line(`<dc:identifier id="uid">%s</dc:identifier>`, EscapeXML(identifier))
	line(`<dc:title id="title">%s</dc:title>`, EscapeXML(doc.Title))
	if doc.Subtitle != "" {
		if profile.Three {
			line(`<meta refines="#title" property="title-type">main</meta>`)
		}
		line(`<dc:title id="subtitle">%s</dc:title>`, EscapeXML(doc.Subtitle))
		if profile.Three {
			line(`<meta refines="#subtitle" property="title-type">subtitle</meta>`)
		}
	}
	if doc.Author != "" {
		if profile.Three {
			line(`<dc:creator id="creator">%s</dc:creator>`, EscapeXML(doc.Author))
			line(`<meta refines="#creator" property="role" scheme="marc:relators">aut</meta>`)
		} else {
			line(`<dc:creator opf:role="aut">%s</dc:creator>`, EscapeXML(doc.Author))
		}
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
	// An edition carries the two facts a book alone cannot: when this object
	// was published, and under what rights it was published.
	if doc.Edition != nil {
		if doc.Edition.Date != "" {
			line(`<dc:date>%s</dc:date>`, EscapeXML(doc.Edition.Date))
		}
		if doc.Edition.Rights != "" {
			line(`<dc:rights>%s</dc:rights>`, EscapeXML(doc.Edition.Rights))
		}
		// EPUB 3's title-type vocabulary has "edition" in it, which is exactly
		// what "Second edition, revised" is: a title of the publication rather
		// than of the work. EPUB 2 has no equivalent and gets none invented.
		if statement := epubEditionStatement(doc.Edition); statement != "" && profile.Three {
			line(`<dc:title id="edition-statement">%s</dc:title>`, EscapeXML(statement))
			line(`<meta refines="#edition-statement" property="title-type">edition</meta>`)
		}
	}
	if doc.SeriesName != "" && profile.Three {
		line(`<meta property="belongs-to-collection" id="series">%s</meta>`, EscapeXML(doc.SeriesName))
		line(`<meta refines="#series" property="collection-type">series</meta>`)
		if doc.SeriesNumber != "" {
			line(`<meta refines="#series" property="group-position">%s</meta>`, EscapeXML(doc.SeriesNumber))
		}
	}
	line(`<dc:language>%s</dc:language>`, EscapeXML(doc.Language))
	// EPUB 2 reading systems — which includes most e-ink hardware still in a
	// drawer — find the cover only through this, and EPUB 3 ones tolerate it.
	if cover.Usable() {
		line(`<meta name="cover" content="%s"/>`, epubCoverItemID)
	}
	if profile.Three {
		if profile.ConformsTo != "" {
			line(`<meta property="dcterms:conformsTo">%s</meta>`, EscapeXML(profile.ConformsTo))
		}
		line(`<meta property="dcterms:modified">%s</meta>`, modified.Format("2006-01-02T15:04:05Z"))
	}
	return b.String()
}

// epubEditionStatement is the wording that names this publication: the
// format's own statement when it carries one, and the edition's label
// otherwise.
func epubEditionStatement(edition *DocumentEdition) string {
	if edition == nil {
		return ""
	}
	if edition.Statement != "" {
		return edition.Statement
	}
	return edition.Label
}

// renderEPUBCoverPage is the page the artwork sits on. The image is sized
// against the viewport rather than given a width in pixels, so it fills a
// phone and a six-inch e-ink screen alike without being upscaled past its own
// resolution.
func renderEPUBCoverPage(doc Document, cover *CoverArt) string {
	epubType := ""
	if documentEPUBProfile(doc).Three {
		epubType = ` epub:type="cover"`
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="%s" lang="%s">
<head><meta charset="UTF-8"/><title>Cover</title>
<style type="text/css">body { margin: 0; padding: 0; text-align: center; }
img { max-width: 100%%; max-height: 100%%; height: auto; }</style></head>
<body class="cover"%s><div><img src="../images/%s" alt="%s"/></div></body>
</html>`,
		EscapeXML(doc.Language), EscapeXML(doc.Language), epubType,
		EscapeXML(cover.FileName), EscapeXML(coverAltText(doc)))
}

func coverAltText(doc Document) string {
	if strings.TrimSpace(doc.Title) == "" {
		return "Cover"
	}
	return "Cover: " + strings.TrimSpace(doc.Title)
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

// renderEPUBNCX is the EPUB 2 table of contents. It is written only for a
// package that declares version 2.0, where it is the only navigation a reading
// system knows how to follow.
func renderEPUBNCX(doc Document, identifier string, sections []epubSection) string {
	var points strings.Builder
	for index, section := range sections {
		fmt.Fprintf(&points, `    <navPoint id="nav-%04d" playOrder="%d">
      <navLabel><text>%s</text></navLabel>
      <content src="text/%s"/>
    </navPoint>
`, index+1, index+1, EscapeXML(displaySectionTitle(section.Title)), section.FileName)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE ncx PUBLIC "-//NISO//DTD ncx 2005-1//EN" "http://www.daisy.org/z3986/2005/ncx-2005-1.dtd">
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1" xml:lang="%s">
  <head>
    <meta name="dtb:uid" content="%s"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle><text>%s</text></docTitle>
  <navMap>
%s  </navMap>
</ncx>`, EscapeXML(doc.Language), EscapeXML(identifier), EscapeXML(ncxDocTitle(doc.Title)), points.String())
}

func ncxDocTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Untitled"
	}
	return strings.TrimSpace(title)
}

func displaySectionTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Untitled section"
	}
	return strings.TrimSpace(title)
}
