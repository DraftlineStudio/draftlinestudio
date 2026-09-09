package export

import (
	"strconv"
	"strings"

	"draftline/internal/types"
)

const pointsPerInch = 72.0

type publicationPDFSpec struct {
	Print                  bool
	TrimWidth              float64
	TrimHeight             float64
	Bleed                  float64
	CropMarks              bool
	GutterMargin           float64
	OuterMargin            float64
	TopMargin              float64
	BottomMargin           float64
	MirroredMargins        bool
	Font                   embeddedFontFamily
	HeadingFont            embeddedFontFamily
	FurnitureFont          embeddedFontFamily
	TitlePageFont          embeddedFontFamily
	CodeFont               embeddedFontFamily
	FontSize               float64
	LineHeight             float64
	ParagraphIndent        float64
	TextAlign              string
	ChapterStartsRecto     bool
	DropCap                bool
	DropCapLines           int
	RunningHeaders         bool
	HeaderStyle            string
	PageNumberPosition     string
	GenerateHalfTitle      bool
	GenerateTOC            bool
	TitlePageStyle         string
	TitlePageShowAuthor    bool
	TitlePageShowPublisher bool
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
		TrimWidth:              w,
		TrimHeight:             h,
		GutterMargin:           pointsPerInch,
		OuterMargin:            pointsPerInch,
		TopMargin:              pointsPerInch,
		BottomMargin:           pointsPerInch,
		Font:                   bodyFont,
		HeadingFont:            bodyFont,
		FurnitureFont:          bodyFont,
		TitlePageFont:          bodyFont,
		CodeFont:               codeFont,
		FontSize:               fontSize,
		LineHeight:             fontSize * lineHeight,
		ParagraphIndent:        parseInches(options.ParagraphIndent, 0.25),
		TextAlign:              normalizedAlignment(options.TextAlign, "left"),
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
		RunningHeaders:         options.RunningHeaders,
		HeaderStyle:            options.HeaderStyle,
		PageNumberPosition:     normalizedPageNumberPosition(options.PageNumberPosition),
		GenerateHalfTitle:      options.GenerateHalfTitle,
		GenerateTOC:            options.GenerateTOC,
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
		return parseInches(customWidth, 5.5), parseInches(customHeight, 8.5)
	default:
		return 5.5 * pointsPerInch, 8.5 * pointsPerInch
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
