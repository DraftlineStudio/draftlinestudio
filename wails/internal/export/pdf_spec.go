package export

import (
	"fmt"
	"strconv"
	"strings"

	"draftline/internal/types"
)

const pointsPerInch = 72.0

type publicationPDFSpec struct {
	Print              bool
	TrimWidth          float64
	TrimHeight         float64
	Bleed              float64
	CropMarks          bool
	GutterMargin       float64
	OuterMargin        float64
	TopMargin          float64
	BottomMargin       float64
	MirroredMargins    bool
	Font               embeddedFontFamily
	HeadingFont        embeddedFontFamily
	FurnitureFont      embeddedFontFamily
	TitlePageFont      embeddedFontFamily
	CodeFont           embeddedFontFamily
	FontSize           float64
	LineHeight         float64
	ParagraphIndent    float64
	TextAlign          string
	ChapterStartsRecto bool
	DropCap            bool
	DropCapLines       int
	SceneBreakStyle    string
	// ParagraphSpacing is blank space after each paragraph, in points. Zero
	// for a book, which separates paragraphs by indenting them.
	ParagraphSpacing float64
	// The narration script's own furniture. None of these apply to a book.
	SlatePage              bool
	NumberParagraphs       bool
	ChapterWordCount       bool
	ChapterStyle           string
	RunningHeaders         bool
	HeaderStyle            string
	HeaderContent          string
	PageNumberPosition     string
	GenerateHalfTitle      bool
	GenerateTOC            bool
	OmitTitlePage          bool
	DraftWatermark         bool
	TitlePageStyle         string
	TitlePageShowAuthor    bool
	TitlePageShowPublisher bool
	// Cover is the edition's artwork, placed on a page of its own before the
	// half title. Nil for an export made from no edition, and for an edition
	// with no cover attached.
	Cover *CoverArt
}

// The bounds a custom trim has to fall inside, in inches.
//
// The lower bound is a little under the smallest mass-market paperback; the
// upper is a little over the largest page any print-on-demand service will
// bind — 8.5 by 11.69 at Amazon, 8.5 by 11 at Ingram. Between them sits every
// book anybody prints. Outside them sits a typing mistake: "99" in a width
// field is a ninety-nine inch page, which is worth refusing rather than
// typesetting in full and handing over without a word.
const (
	minTrimInches = 3.0
	maxTrimInches = 12.0
)

// validateTrim refuses a custom trim that is not a printable page.
//
// Only a custom trim can be wrong: the named sizes are Draftline's own. The
// message names the field and the bounds, because the author has to change a
// number and needs to know which one and to what.
func validateTrim(options types.PrintPDFOptions) error {
	if strings.ToLower(strings.TrimSpace(options.TrimSize)) != "custom" {
		return nil
	}
	if err := checkTrimSide("width", options.CustomWidth); err != nil {
		return err
	}
	return checkTrimSide("height", options.CustomHeight)
}

func checkTrimSide(side, value string) error {
	trimmed := strings.TrimSpace(value)
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		if trimmed == "" {
			return fmt.Errorf("this export has no trim %s. Give one between %g and %g inches", side, minTrimInches, maxTrimInches)
		}
		return fmt.Errorf("%q is not a trim %s. Give a measurement in inches, between %g and %g", trimmed, side, minTrimInches, maxTrimInches)
	}
	if parsed < minTrimInches || parsed > maxTrimInches {
		return fmt.Errorf("a trim %s of %g inches cannot be printed. Give one between %g and %g inches", side, parsed, minTrimInches, maxTrimInches)
	}
	return nil
}

func readingPDFSpec(options types.PDFOptions) publicationPDFSpec {
	w, h := readingPageSize(options.PageSize)
	fontSize := float64(options.FontSize)
	if fontSize <= 0 {
		fontSize = 12
	}
	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.5
	}
	bodyFont := resolvePDFFont(options.FontFamily)
	codeFont := embeddedPDFFonts["ibmplexmono"]
	return publicationPDFSpec{
		TrimWidth:       w,
		TrimHeight:      h,
		GutterMargin:    pointsPerInch,
		OuterMargin:     pointsPerInch,
		TopMargin:       pointsPerInch,
		BottomMargin:    pointsPerInch,
		Font:            bodyFont,
		HeadingFont:     bodyFont,
		FurnitureFont:   bodyFont,
		TitlePageFont:   bodyFont,
		CodeFont:        codeFont,
		FontSize:        fontSize,
		LineHeight:      fontSize * lineHeight,
		ParagraphIndent: parseInches(options.ParagraphIndent, 0.25),
		TextAlign:       normalizedAlignment(options.TextAlign, "left"),
		// A reading PDF had no page numbers at all: the field was left empty
		// and drawFurniture reads an empty position as "print nothing". A
		// reviewer marking up a fixed-layout copy has no way to say where they
		// are without them, and a reading copy is the one export whose whole
		// purpose is being read and commented on. Centred at the foot, because
		// a reading copy is read as single pages rather than as spreads and
		// has no outside edge to sit against.
		PageNumberPosition:     readingFolioPosition(options),
		DraftWatermark:         options.DraftWatermark,
		OmitTitlePage:          options.OmitTitlePage,
		TitlePageStyle:         "classic",
		TitlePageShowAuthor:    true,
		TitlePageShowPublisher: true,
	}
}

func printPDFSpec(options types.PrintPDFOptions) publicationPDFSpec {
	w, h := trimPageSize(options.TrimSize, options.CustomWidth, options.CustomHeight)
	fontSize := float64(options.FontSize)
	if fontSize <= 0 {
		fontSize = 9
	}
	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.4
	}
	dropLines := options.DropCapLines
	if dropLines < 2 || dropLines > 4 {
		dropLines = 3
	}
	bodyFont := resolvePDFFont(options.FontFamily)
	codeFont := embeddedPDFFonts["ibmplexmono"]
	headingChoice := options.HeadingFont
	titleChoice := options.TitlePageFont
	if strings.TrimSpace(headingChoice) == "" {
		headingChoice = "classic"
	}
	if strings.TrimSpace(titleChoice) == "" {
		titleChoice = headingChoice
	}
	showAuthor := options.TitlePageShowAuthor
	showPublisher := options.TitlePageShowPublisher
	if strings.TrimSpace(options.TitlePageStyle) == "" {
		showAuthor = true
		showPublisher = true
	}
	return publicationPDFSpec{
		Print:                  true,
		TrimWidth:              w,
		TrimHeight:             h,
		Bleed:                  parseInches(options.Bleed, 0),
		CropMarks:              options.IncludeCropMarks,
		GutterMargin:           parseInches(options.GutterMargin, 0.875),
		OuterMargin:            parseInches(options.OuterMargin, 0.625),
		TopMargin:              parseInches(options.TopMargin, 0.75),
		BottomMargin:           parseInches(options.BottomMargin, 0.625),
		MirroredMargins:        options.MirroredMargins,
		Font:                   bodyFont,
		HeadingFont:            resolveDisplayFont(headingChoice, bodyFont),
		FurnitureFont:          resolveDisplayFont(options.FurnitureFont, bodyFont),
		TitlePageFont:          resolveDisplayFont(titleChoice, bodyFont),
		CodeFont:               codeFont,
		FontSize:               fontSize,
		LineHeight:             fontSize * lineHeight,
		ParagraphIndent:        parseInches(options.ParagraphIndent, 0.25),
		TextAlign:              normalizedAlignment(options.TextAlign, "left"),
		ChapterStartsRecto:     options.ChapterStartsRecto,
		DropCap:                options.DropCap,
		DropCapLines:           dropLines,
		SceneBreakStyle:        options.SceneBreakStyle,
		ChapterStyle:           options.ChapterStyle,
		RunningHeaders:         options.RunningHeaders,
		HeaderStyle:            options.HeaderStyle,
		HeaderContent:          options.HeaderContent,
		PageNumberPosition:     normalizedPageNumberPosition(options.PageNumberPosition),
		GenerateHalfTitle:      options.GenerateHalfTitle,
		GenerateTOC:            options.GenerateTOC,
		OmitTitlePage:          options.OmitTitlePage,
		TitlePageStyle:         normalizedTitlePageStyle(options.TitlePageStyle),
		TitlePageShowAuthor:    showAuthor,
		TitlePageShowPublisher: showPublisher,
	}
}

func normalizedTitlePageStyle(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "minimal", "dramatic":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "classic"
	}
}

func readingPageSize(name string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "a4":
		return 595.28, 841.89
	case "5x8":
		return 5 * pointsPerInch, 8 * pointsPerInch
	case "5.5x8.5":
		return 5.5 * pointsPerInch, 8.5 * pointsPerInch
	case "6x9":
		return 6 * pointsPerInch, 9 * pointsPerInch
	default:
		return 8.5 * pointsPerInch, 11 * pointsPerInch
	}
}

func trimPageSize(name, customWidth, customHeight string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "5x8":
		return 5 * pointsPerInch, 8 * pointsPerInch
	case "5.25x8":
		return 5.25 * pointsPerInch, 8 * pointsPerInch
	case "6x9":
		return 6 * pointsPerInch, 9 * pointsPerInch
	case "custom":
		// A backstop, not the check. PrintPDF refuses an out-of-range trim
		// outright and says so; this only keeps a renderer that is reached
		// some other way from being handed a page it cannot lay out.
		return clampTrim(parseInches(customWidth, 5.5)), clampTrim(parseInches(customHeight, 8.5))
	default:
		return 5.5 * pointsPerInch, 8.5 * pointsPerInch
	}
}

func clampTrim(points float64) float64 {
	switch {
	case points < minTrimInches*pointsPerInch:
		return minTrimInches * pointsPerInch
	case points > maxTrimInches*pointsPerInch:
		return maxTrimInches * pointsPerInch
	default:
		return points
	}
}

func parseInches(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0 {
		parsed = fallback
	}
	return parsed * pointsPerInch
}

func normalizedAlignment(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "center", "right", "justify":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return fallback
	}
}

func normalizedPageNumberPosition(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bottom-outside", "top-outside":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "bottom-center"
	}
}

// readingFolioPosition is where a reading copy's page numbers sit, or nowhere.
// A reviewer marking up a fixed-layout copy has no way to say where they are
// without them, so they are on unless the author turns them off.
func readingFolioPosition(options types.PDFOptions) string {
	if options.HideFolios {
		return ""
	}
	return "bottom-center"
}
