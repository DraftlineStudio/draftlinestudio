package export

import (
	_ "embed"
	"strings"

	"codeberg.org/go-pdf/fpdf"
)

// The full font programs are embedded in Draftline and subset by fpdf in each
// output. PDF generation therefore never depends on a machine-local font.

//go:embed fonts/Merriweather-Regular.ttf
var merriweatherRegular []byte

//go:embed fonts/Merriweather-Bold.ttf
var merriweatherBold []byte

//go:embed fonts/Merriweather-Italic.ttf
var merriweatherItalic []byte

//go:embed fonts/Merriweather-BoldItalic.ttf
var merriweatherBoldItalic []byte

//go:embed fonts/Lato-Regular.ttf
var latoRegular []byte

//go:embed fonts/Lato-Bold.ttf
var latoBold []byte

//go:embed fonts/Lato-Italic.ttf
var latoItalic []byte

//go:embed fonts/Lato-BoldItalic.ttf
var latoBoldItalic []byte

type embeddedFontFamily struct {
	ID          string
	DisplayName string
	Regular     []byte
	Bold        []byte
	Italic      []byte
	BoldItalic  []byte
}

var embeddedPDFFonts = map[string]embeddedFontFamily{
	"merriweather": {
		ID: "Merriweather", DisplayName: "Merriweather",
		Regular: merriweatherRegular, Bold: merriweatherBold,
		Italic: merriweatherItalic, BoldItalic: merriweatherBoldItalic,
	},
	"lato": {
		ID: "Lato", DisplayName: "Lato",
		Regular: latoRegular, Bold: latoBold,
		Italic: latoItalic, BoldItalic: latoBoldItalic,
	},
}

func resolvePDFFont(name string) embeddedFontFamily {
	if family, ok := embeddedPDFFonts[strings.ToLower(strings.TrimSpace(name))]; ok {
		return family
	}
	return embeddedPDFFonts["merriweather"]
}

func registerPDFFonts(pdf *fpdf.Fpdf) {
	for _, family := range embeddedPDFFonts {
		pdf.AddUTF8FontFromBytes(family.ID, "", family.Regular)
		pdf.AddUTF8FontFromBytes(family.ID, "B", family.Bold)
		pdf.AddUTF8FontFromBytes(family.ID, "I", family.Italic)
		pdf.AddUTF8FontFromBytes(family.ID, "BI", family.BoldItalic)
	}
}
