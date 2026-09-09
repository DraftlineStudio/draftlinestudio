package export

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"draftline/internal/types"
)

// entityAt reports whether s starting at the '&' in position i forms a valid
// XML/HTML entity reference (named, decimal, or hex). Go's RE2 has no
// lookahead, so raw-ampersand detection is done by scanning instead.
func entityAt(s string, i int) bool {
	j := i + 1
	if j < len(s) && s[j] == '#' {
		j++
		if j < len(s) && (s[j] == 'x' || s[j] == 'X') {
			j++
			start := j
			for j < len(s) && isHex(s[j]) {
				j++
			}
			return j > start && j < len(s) && s[j] == ';'
		}
		start := j
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		return j > start && j < len(s) && s[j] == ';'
	}
	// Named entity: letter followed by letters/digits, terminated by ';'.
	if j >= len(s) || !isAlpha(s[j]) {
		return false
	}
	for j < len(s) && (isAlpha(s[j]) || (s[j] >= '0' && s[j] <= '9')) {
		j++
	}
	return j < len(s) && s[j] == ';'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// findRawAmpersand returns the byte index of the first '&' that does not begin
// a valid entity reference, or -1 if none. Such an ampersand would make an XML
// document non-well-formed.
func findRawAmpersand(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '&' && !entityAt(s, i) {
			return i
		}
	}
	return -1
}

// sampleBook returns a small book whose text deliberately contains characters
// that must be escaped exactly once: an ampersand, angle brackets, and a
// numeric character reference already present in the source HTML.
func sampleBook() types.BookData {
	return types.BookData{
		Metadata: types.Metadata{
			Title:     "Smith & Sons: <Tales>",
			Author:    "A. B. \"Author\"",
			Publisher: "Books & More",
		},
		Copyright: "<p>Copyright &copy; 2026 Smith &amp; Sons</p>",
		Body: []types.ChapterItem{
			{
				Title:    "Chapter 1 & Beginnings",
				Subtitle: "In which <things> happen",
				Type:     "chapter",
				Content:  "<p>Tom &amp; Jerry &mdash; it&#8217;s a &lt;test&gt; of Smith &amp; Sons.</p>",
			},
			{
				Title:   "Chapter 2",
				Type:    "chapter",
				Content: "<p>The second chapter mentions R&amp;D and math like 3 &lt; 5.</p>",
			},
		},
	}
}

func defaultExportOptions() types.ExportOptions {
	return types.ExportOptions{
		IncludeCopyright:   true,
		IncludeFrontMatter: true,
		IncludeBackMatter:  true,
	}
}

// readZipPart opens a zip archive at path and returns the bytes of the named
// entry, failing the test if the archive or entry cannot be read.
func readZipPart(t *testing.T, path, name string) []byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip %s: %v", path, err)
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open zip entry %s: %v", name, err)
			}
			defer func() { _ = rc.Close() }()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read zip entry %s: %v", name, err)
			}
			return b
		}
	}
	t.Fatalf("zip entry %q not found in %s", name, path)
	return nil
}

func TestDOCXExportStructureAndEscaping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.docx")

	res := DOCX(path, sampleBook(), defaultExportOptions())
	if !res.Success {
		t.Fatalf("DOCX export failed: %s", res.Error)
	}

	// The archive must be a valid zip and contain the mandatory OOXML parts.
	required := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/_rels/document.xml.rels",
		"word/styles.xml",
		"word/document.xml",
	}
	for _, name := range required {
		b := readZipPart(t, path, name)
		if len(b) == 0 {
			t.Errorf("required part %q is empty", name)
		}
	}

	doc := string(readZipPart(t, path, "word/document.xml"))

	// Chapter text must be present, decoded from source entities and re-escaped
	// exactly once for XML.
	wants := []string{
		"Tom &amp; Jerry", // &amp; decoded to & then re-escaped once
		"it’s a &lt;test&gt; of Smith &amp; Sons.", // &#8217; decoded to unicode, &lt;/&gt; re-escaped
		"R&amp;D and math like 3 &lt; 5.",
		"Chapter 1 &amp; Beginnings", // heading escaped once
		"In which &lt;things&gt; happen",
	}
	for _, w := range wants {
		if !strings.Contains(doc, w) {
			t.Errorf("word/document.xml missing expected text %q", w)
		}
	}

	// No double-escaping anywhere.
	if strings.Contains(doc, "&amp;amp;") {
		t.Errorf("word/document.xml contains double-escaped ampersand")
	}
	if strings.Contains(doc, "&amp;lt;") || strings.Contains(doc, "&amp;gt;") {
		t.Errorf("word/document.xml contains double-escaped angle bracket")
	}

	// No raw, unescaped ampersands.
	if p := findRawAmpersand(doc); p >= 0 {
		start := p - 10
		if start < 0 {
			start = 0
		}
		end := p + 10
		if end > len(doc) {
			end = len(doc)
		}
		t.Errorf("word/document.xml contains raw '&' near: %q", doc[start:end])
	}

	// Metadata title (also entity-laden) must be escaped once.
	if !strings.Contains(doc, "Smith &amp; Sons: &lt;Tales&gt;") {
		t.Errorf("word/document.xml missing correctly escaped title")
	}
}

func TestEPUBExportStructureAndEscaping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")

	res := EPUB(path, sampleBook(), types.EPUBOptions{ExportOptions: defaultExportOptions()})
	if !res.Success {
		t.Fatalf("EPUB export failed: %s", res.Error)
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer func() { _ = zr.Close() }()

	if len(zr.File) == 0 {
		t.Fatal("epub archive is empty")
	}
	// mimetype must be the first entry and stored uncompressed.
	first := zr.File[0]
	if first.Name != "mimetype" {
		t.Errorf("first epub entry is %q, want \"mimetype\"", first.Name)
	}
	if first.Method != zip.Store {
		t.Errorf("mimetype must be stored (uncompressed), got method %d", first.Method)
	}
	rc, err := first.Open()
	if err != nil {
		t.Fatalf("open mimetype: %v", err)
	}
	mime, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(mime) != "application/epub+zip" {
		t.Errorf("mimetype content = %q", string(mime))
	}

	// content.opf must exist and carry escaped metadata.
	opf := string(readZipPart(t, path, "OEBPS/content.opf"))
	if !strings.Contains(opf, "<dc:title>Smith &amp; Sons: &lt;Tales&gt;</dc:title>") {
		t.Errorf("content.opf missing correctly escaped title, got:\n%s", opf)
	}
	if !strings.Contains(opf, "<dc:publisher>Books &amp; More</dc:publisher>") {
		t.Errorf("content.opf missing correctly escaped publisher")
	}
	if findRawAmpersand(opf) >= 0 {
		t.Errorf("content.opf contains raw '&'")
	}
	if strings.Contains(opf, "&amp;amp;") {
		t.Errorf("content.opf contains double-escaped ampersand")
	}
	// Chapter parts must be declared in manifest and spine.
	for _, part := range []string{"section-0001.xhtml", "section-0002.xhtml", "section-0003.xhtml"} {
		if !strings.Contains(opf, part) {
			t.Errorf("content.opf manifest/spine missing %q", part)
		}
	}

	// Chapter xhtml must contain the chapter body and an escaped heading.
	ch0 := string(readZipPart(t, path, "OEBPS/text/section-0002.xhtml"))
	if !strings.Contains(ch0, "<h1>Chapter 1 &amp; Beginnings</h1>") {
		t.Errorf("chapter0.xhtml missing escaped heading, got:\n%s", ch0)
	}
	// Source HTML is normalized through the shared document model. Its text
	// and Unicode punctuation survive without carrying HTML-only entities into
	// the XHTML package.
	if !strings.Contains(ch0, "Tom &amp; Jerry — it’s a &lt;test&gt; of Smith &amp; Sons.") {
		t.Errorf("chapter0.xhtml body entities altered, got:\n%s", ch0)
	}
	if strings.Contains(ch0, "&amp;amp;") {
		t.Errorf("chapter0.xhtml contains double-escaped ampersand")
	}
	if findRawAmpersand(ch0) >= 0 {
		t.Errorf("chapter0.xhtml contains raw '&'")
	}
}

// pdfObjectCount returns the number of "N 0 obj" markers in a PDF byte slice.
func pdfObjectCount(pdf []byte) int {
	return len(regexp.MustCompile(`(?m)^\d+ 0 obj`).FindAll(pdf, -1))
}

// assertParseablePDF checks the standard PDF envelope: %PDF header, %%EOF
// trailer, a startxref whose offset points at the xref keyword, and internal
// agreement between the xref-declared object count, the trailer /Size, and the
// actual number of object markers.
func assertParseablePDF(t *testing.T, pdf []byte) {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF does not start with %%PDF- header")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatalf("PDF missing %%%%EOF trailer marker")
	}

	// startxref -> xref keyword.
	idx := bytes.LastIndex(pdf, []byte("startxref"))
	if idx < 0 {
		t.Fatal("PDF missing startxref")
	}
	fields := strings.Fields(string(pdf[idx+len("startxref"):]))
	if len(fields) == 0 {
		t.Fatal("no value after startxref")
	}
	xrefOffset, err := strconv.Atoi(fields[0])
	if err != nil {
		t.Fatalf("startxref value not integer: %q", fields[0])
	}
	if xrefOffset < 0 || xrefOffset+4 > len(pdf) {
		t.Fatalf("startxref offset %d out of bounds (len %d)", xrefOffset, len(pdf))
	}
	if got := string(pdf[xrefOffset : xrefOffset+4]); got != "xref" {
		t.Fatalf("startxref offset %d points at %q, want \"xref\"", xrefOffset, got)
	}

	// xref subsection header: "0 N".
	xrefBody := string(pdf[xrefOffset+len("xref\n"):])
	headerLine := strings.SplitN(xrefBody, "\n", 2)[0]
	header := strings.Fields(headerLine)
	if len(header) != 2 || header[0] != "0" {
		t.Fatalf("unexpected xref subsection header: %q", headerLine)
	}
	declaredSize, err := strconv.Atoi(header[1])
	if err != nil {
		t.Fatalf("xref count not integer: %q", header[1])
	}

	// Trailer /Size must match the xref-declared size.
	sizeRe := regexp.MustCompile(`/Size (\d+)`)
	m := sizeRe.FindSubmatch(pdf)
	if m == nil {
		t.Fatal("trailer missing /Size")
	}
	trailerSize, _ := strconv.Atoi(string(m[1]))
	if trailerSize != declaredSize {
		t.Errorf("trailer /Size %d != xref size %d", trailerSize, declaredSize)
	}

	// Actual object markers = declaredSize - 1 (object 0 is the free head and
	// has no "0 0 obj" marker).
	objCount := pdfObjectCount(pdf)
	if objCount != declaredSize-1 {
		t.Errorf("found %d object markers, expected %d (xref size %d minus free object)",
			objCount, declaredSize-1, declaredSize)
	}

	// Catalog and Pages roots must be present.
	if !bytes.Contains(pdf, []byte("/Type /Catalog")) {
		t.Errorf("PDF missing /Catalog object")
	}
	if !bytes.Contains(pdf, []byte("/Type /Pages")) {
		t.Errorf("PDF missing /Pages object")
	}
}

func TestPDFExportStructure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.pdf")

	opts := types.PDFOptions{
		ExportOptions: defaultExportOptions(),
		PageSize:      "letter",
		FontSize:      12,
	}
	res := PDF(path, sampleBook(), opts)
	if !res.Success {
		t.Fatalf("PDF export failed: %s", res.Error)
	}
	pdf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pdf: %v", err)
	}
	assertParseablePDF(t, pdf)
}

func TestPrintPDFExportStructure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book-print.pdf")

	opts := types.PrintPDFOptions{
		PDFOptions: types.PDFOptions{
			ExportOptions: defaultExportOptions(),
			PageSize:      "6x9",
			FontSize:      12,
		},
		TrimSize:           "6x9",
		PageNumberPosition: "bottom-center",
		RunningHeaders:     true,
		GenerateHalfTitle:  true,
	}
	res := PrintPDF(path, sampleBook(), opts)
	if !res.Success {
		t.Fatalf("PrintPDF export failed: %s", res.Error)
	}
	pdf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read print pdf: %v", err)
	}
	assertParseablePDF(t, pdf)
}
