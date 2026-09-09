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

//go:embed fonts/EBGaramond.ttf
var ebGaramond []byte

//go:embed fonts/GreatVibes-Regular.ttf
var greatVibes []byte

//go:embed fonts/Orbitron.ttf
var orbitron []byte

//go:embed fonts/CinzelDecorative-Regular.ttf
var cinzelDecorative []byte

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
	"ebgaramond": {
		ID: "EBGaramond", DisplayName: "EB Garamond",
		Regular: ebGaramond, Bold: ebGaramond,
		Italic: ebGaramond, BoldItalic: ebGaramond,
	},
	"greatvibes": {
		ID: "GreatVibes", DisplayName: "Great Vibes",
		Regular: greatVibes, Bold: greatVibes,
		Italic: greatVibes, BoldItalic: greatVibes,
	},
	"orbitron": {
		ID: "Orbitron", DisplayName: "Orbitron",
		Regular: orbitron, Bold: orbitron,
		Italic: orbitron, BoldItalic: orbitron,
	},
	"cinzel": {
		ID: "CinzelDecorative", DisplayName: "Cinzel Decorative",
		Regular: cinzelDecorative, Bold: cinzelDecorative,
		Italic: cinzelDecorative, BoldItalic: cinzelDecorative,
	},
}

func resolvePDFFont(name string) embeddedFontFamily {
	if family, ok := embeddedPDFFonts[strings.ToLower(strings.TrimSpace(name))]; ok {
		return family
	}
	return embeddedPDFFonts["merriweather"]
}

func resolveDisplayFont(name string, body embeddedFontFamily) embeddedFontFamily {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "classic":
		return embeddedPDFFonts["ebgaramond"]
	case "modern":
		return embeddedPDFFonts["lato"]
	case "romance":
		return embeddedPDFFonts["greatvibes"]
	case "scifi":
		return embeddedPDFFonts["orbitron"]
	case "fantasy":
		return embeddedPDFFonts["cinzel"]
	default:
		return body
	}
}

func registerPDFFonts(pdf *fpdf.Fpdf, families ...embeddedFontFamily) {
	registered := make(map[string]struct{}, len(families))
	for _, family := range families {
		if family.ID == "" {
			continue
		}
		if _, exists := registered[family.ID]; exists {
			continue
		}
		registered[family.ID] = struct{}{}
		pdf.AddUTF8FontFromBytes(family.ID, "", family.Regular)
		pdf.AddUTF8FontFromBytes(family.ID, "B", family.Bold)
		pdf.AddUTF8FontFromBytes(family.ID, "I", family.Italic)
		pdf.AddUTF8FontFromBytes(family.ID, "BI", family.BoldItalic)
	}
}
