package types

// ExportOptions contains common export settings.
type ExportOptions struct {
	IncludeCopyright   bool `json:"includeCopyright"`
	IncludeFrontMatter bool `json:"includeFrontMatter"`
	IncludeBackMatter  bool `json:"includeBackMatter"`
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
