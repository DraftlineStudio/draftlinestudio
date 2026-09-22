package export

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

// The narration script.
//
// This is not the reading copy under another name, and building it as one was
// the mistake it replaces. A narrator works from a page differently from a
// reader: they mark it up, they need the line they are on to be findable after
// a retake, they need to be told where a scene turns in a word they can act on
// rather than shown an ornament they cannot say, and they need somewhere to
// write down how a name is pronounced before they reach it.
//
// So the script gets its own page. Large type, open leading, ragged right —
// justification moves the words between takes — a number beside every
// paragraph, a slate before each chapter, and the outer margin left empty when
// the narrator wants it for notes.

func AudioScriptBytes(book types.BookData, options types.AudioOptions, cover *CoverArt) ([]byte, error) {
	doc, err := BuildDocument(book, options.ExportOptions)
	if err != nil {
		return nil, err
	}
	spec := audioScriptSpec(options)
	if options.CoverPage {
		spec.Cover = cover
	}
	data, err := renderPublicationPDF(doc, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to render the narration script: %w", err)
	}
	return data, nil
}

// audioScriptSpec is the page a narrator reads from.
func audioScriptSpec(options types.AudioOptions) publicationPDFSpec {
	w, h := readingPageSize(options.PageSize)
	fontSize := float64(options.FontSize)
	if fontSize <= 0 {
		fontSize = 14
	}
	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.8
	}
	font := resolvePDFFont(options.FontFamily)
	if strings.TrimSpace(options.FontFamily) == "" {
		font = resolvePDFFont("lato")
	}

	// The outer margin is the pronunciation column. It is not a separate
	// layout: it is room, which is what a narrator actually writes in.
	outer := pointsPerInch
	if options.PronunciationColumn {
		outer = 2.25 * pointsPerInch
	}

	return publicationPDFSpec{
		TrimWidth:     w,
		TrimHeight:    h,
		GutterMargin:  pointsPerInch,
		OuterMargin:   outer,
		TopMargin:     pointsPerInch,
		BottomMargin:  pointsPerInch,
		Font:          font,
		HeadingFont:   font,
		FurnitureFont: font,
		TitlePageFont: font,
		CodeFont:      embeddedPDFFonts["ibmplexmono"],
		FontSize:      fontSize,
		LineHeight:    fontSize * lineHeight,
		// No first-line indent and no drop cap: a script is read aloud, and an
		// indent is a convention for the eye.
		ParagraphIndent: 0,
		ParagraphSpacing: audioParagraphSpacing(options.ParagraphSpacing) *
			fontSize * lineHeight,
		// Ragged right, always. Justification moves words between takes and a
		// narrator reading the same line twice should see the same line.
		TextAlign:          "left",
		PageNumberPosition: "bottom-center",
		// A slate is a chapter on a page of its own — the cue a take is
		// recorded against.
		ChapterStartsRecto: false,
		SlatePage:          options.SlatePage,
		NumberParagraphs:   options.NumberParagraphs,
		SceneBreakStyle:    audioSceneBreak(options.PauseBreaks),
		ChapterWordCount:   options.ChapterWordCount,
		OmitTitlePage:      options.OmitTitlePage,
		TitlePageStyle:     "minimal",

		TitlePageShowAuthor:    true,
		TitlePageShowPublisher: false,
	}
}

// audioParagraphSpacing is the gap between paragraphs, in lines.
func audioParagraphSpacing(words string) float64 {
	switch strings.ToLower(strings.TrimSpace(words)) {
	case "half", "half line between":
		return 0.5
	case "two", "two lines between":
		return 2
	default:
		return 1
	}
}

// audioSceneBreak is the one place a script deliberately differs from every
// other export: an asterism is silent, and a narrator needs to be told.
func audioSceneBreak(pause bool) string {
	if pause {
		return "pause"
	}
	return "space"
}
