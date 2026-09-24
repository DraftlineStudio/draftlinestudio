package types

// ExportOptions contains common export settings.
type ExportOptions struct {
	IncludeCopyright   bool `json:"includeCopyright"`
	IncludeFrontMatter bool `json:"includeFrontMatter"`
	IncludeBackMatter  bool `json:"includeBackMatter"`
	// OmitTitlePage drops the generated title page: the book's name, the
	// author, the imprint. It is stated the negative way round on purpose —
	// every export has always produced one, so absence has to keep meaning
	// "produce it", including for a record written before this field existed.
	OmitTitlePage bool `json:"omitTitlePage"`
	// EditionID and FormatID name the registered format this export is made
	// against. They travel in the options rather than as a separate argument
	// so that everything the wizard decided about one export arrives in one
	// object — which is also what gets frozen onto the format record.
	//
	// An export that names no format is the export Draftline has always made:
	// the book alone, no ISBN beyond the book's own identifier list, no cover.
	EditionID string `json:"editionID,omitempty"`
	FormatID  string `json:"formatID,omitempty"`
}

// EPUBOptions controls semantic, reflowable ebook presentation. The reader
// font setting leaves typography entirely to the reading app; named families are
// embedded so the edition has a consistent publisher default while remaining
// overridable by accessible EPUB readers.
type EPUBOptions struct {
	ExportOptions
	FontFamily      string `json:"fontFamily"`      // reader, merriweather, lato
	ParagraphStyle  string `json:"paragraphStyle"`  // indented, spaced
	TextAlign       string `json:"textAlign"`       // reader, left, justify
	ChapterStyle    string `json:"chapterStyle"`    // classic, minimal
	SceneBreakStyle string `json:"sceneBreakStyle"` // asterism, rule, space
	// DropCap sets the first letter of each chapter's opening paragraph as a
	// raised initial. It is ordinary CSS (::first-letter) that reading
	// systems including Kindle honour, so an ebook gets the same choice a
	// printed page does rather than being told it cannot have one.
	DropCap bool `json:"dropCap"`
	// Version is "EPUB 3.3", "EPUB 3.0" or "EPUB 2.0.1". An export made from
	// a registered edition takes it from the record instead; this is for one
	// made from no edition, which has no record to read.
	Version string `json:"version,omitempty"`
}

// PDFOptions extends ExportOptions with PDF-specific settings.
type PDFOptions struct {
	ExportOptions
	PageSize        string  `json:"pageSize"`        // letter, a4, 6x9, 5.5x8.5, 5x8
	FontFamily      string  `json:"fontFamily"`      // merriweather, lato
	FontSize        int     `json:"fontSize"`        // 9, 10, 11, 12, 14
	LineHeight      float64 `json:"lineHeight"`      // 1.3, 1.4, 1.5, 1.6
	ParagraphIndent string  `json:"paragraphIndent"` // inches
	TextAlign       string  `json:"textAlign"`       // justify, left
	// HideFolios drops the page numbers. Stated the negative way round so a
	// record written before this field existed keeps its folios.
	HideFolios bool `json:"hideFolios,omitempty"`
	// DraftWatermark prints DRAFT across every page, for a copy going out
	// for comment that must not be mistaken for the finished book.
	DraftWatermark bool `json:"draftWatermark,omitempty"`
}

// DOCXOptions is the editable manuscript: the file that goes to an editor and
// comes back marked up. Its settings are about that round trip, not about
// looking like a book.
type DOCXOptions struct {
	ExportOptions
	// BodyStyle is "normal" (a readable 12 pt proportional page) or
	// "manuscript" (12 pt Courier, double spaced) — the format an agent or a
	// submissions desk still asks for.
	BodyStyle string `json:"bodyStyle"`
	// ChapterBreak is "page" (each chapter starts a new page) or "run"
	// (chapters follow one another).
	ChapterBreak string `json:"chapterBreak"`
	// TrackChanges turns revision recording on inside the file, so an editor
	// opening it is already marking up rather than silently rewriting.
	TrackChanges bool `json:"trackChanges"`
	// HashSceneBreaks writes a scene break as "#", which is what a manuscript
	// uses; an asterism is a typeset ornament.
	HashSceneBreaks bool `json:"hashSceneBreaks"`
}

// AudioOptions is the narration script: the file a voice actor reads a book
// from. It is NOT the reading copy with a different name. A narrator marks up
// a page, needs the line they are on to be findable after a retake, and needs
// to be told where a scene turns rather than shown a typographic ornament they
// cannot say out loud. So it gets its own page: large type, open leading,
// ragged right, numbered paragraphs, a slate before each chapter, and room in
// the margin to write a pronunciation down.
type AudioOptions struct {
	ExportOptions
	PageSize         string  `json:"pageSize"`         // letter, a4
	FontFamily       string  `json:"fontFamily"`       // lato, merriweather
	FontSize         int     `json:"fontSize"`         // 12, 14, 16
	LineHeight       float64 `json:"lineHeight"`       // 1.5, 1.8, 2.0
	ParagraphSpacing string  `json:"paragraphSpacing"` // half, one, two lines between
	// SlatePage puts each chapter's title on a page of its own, which is the
	// cue a narrator records against.
	SlatePage bool `json:"slatePage"`
	// NumberParagraphs prints a number beside every paragraph so a retake can
	// be asked for by number instead of by reading the line back.
	NumberParagraphs bool `json:"numberParagraphs"`
	// PauseBreaks writes a scene break as [PAUSE]. An asterism is silent.
	PauseBreaks bool `json:"pauseBreaks"`
	// PronunciationColumn reserves the outer margin for the narrator to write
	// names and stresses into.
	PronunciationColumn bool `json:"pronunciationColumn"`
	// CoverPage opens the script with the ebook cover, so the narrator has the
	// book in front of them.
	CoverPage bool `json:"coverPage"`
	// ChapterWordCount prints the length of each chapter under its slate, for
	// estimating session time.
	ChapterWordCount bool `json:"chapterWordCount"`
}

// PrintPDFOptions extends PDFOptions with print-ready settings.
type PrintPDFOptions struct {
	PDFOptions
	TrimSize         string `json:"trimSize"` // 5x8, 5.25x8, 5.5x8.5, 6x9, custom
	CustomWidth      string `json:"customWidth"`
	CustomHeight     string `json:"customHeight"`
	Bleed            string `json:"bleed"`
	GutterMargin     string `json:"gutterMargin"`
	OuterMargin      string `json:"outerMargin"`
	TopMargin        string `json:"topMargin"`
	BottomMargin     string `json:"bottomMargin"`
	IncludeCropMarks bool   `json:"includeCropMarks"`
	// SkipKDPChecks allows an author preparing a file for another printer to
	// use that printer's own rules. The zero value deliberately keeps the KDP
	// checks on, including for settings saved by an older Draftline.
	SkipKDPChecks bool `json:"skipKDPChecks,omitempty"`
	// Typography
	// Chapter styling
	ChapterStartsRecto bool `json:"chapterStartsRecto"`
	DropCap            bool `json:"dropCap"`
	DropCapLines       int  `json:"dropCapLines"` // 2, 3, 4
	// SceneBreakStyle and ChapterStyle are the same two choices an ebook
	// offers, asked of a printed page. They are preferences rather than
	// deviations: a book set with a short rule instead of an asterism is still
	// set the industry-standard way.
	SceneBreakStyle string `json:"sceneBreakStyle"` // asterism, rule, space
	ChapterStyle    string `json:"chapterStyle"`    // classic, compact
	// Headers & footers
	RunningHeaders bool   `json:"runningHeaders"`
	HeaderStyle    string `json:"headerStyle"` // smallcaps, italic, normal
	// HeaderContent is what the running head says on each side of the spread:
	//
	//   author-title   the author on the verso, the book on the recto — which,
	//                  with folios set top-outside, reads across the spread as
	//                  "137 | A. Marsh        Wide Water | 138"
	//   title-chapter  the book on the verso, the chapter on the recto
	//   chapter        the chapter on both
	HeaderContent      string `json:"headerContent"`
	PageNumberPosition string `json:"pageNumberPosition"` // bottom-center, bottom-outside, top-outside
	// Front matter
	GenerateHalfTitle bool `json:"generateHalfTitle"`
	GenerateTOC       bool `json:"generateTOC"`
	MirroredMargins   bool `json:"mirroredMargins"` // Critical: swap gutter/outer for odd/even pages
	// Display typography
	HeadingFont            string `json:"headingFont"` // body, classic, modern, romance, scifi, fantasy
	FurnitureFont          string `json:"furnitureFont"`
	TitlePageFont          string `json:"titlePageFont"`
	TitlePageStyle         string `json:"titlePageStyle"` // classic, minimal, dramatic
	TitlePageShowAuthor    bool   `json:"titlePageShowAuthor"`
	TitlePageShowPublisher bool   `json:"titlePageShowPublisher"`
}
