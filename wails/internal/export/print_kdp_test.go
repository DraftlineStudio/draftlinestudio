package export

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func kdpOptions() types.PrintPDFOptions {
	return types.PrintPDFOptions{
		PDFOptions:      types.PDFOptions{FontSize: 10},
		TrimSize:        "6x9",
		Bleed:           "0.125",
		GutterMargin:    "0.875",
		OuterMargin:     "0.375",
		TopMargin:       "0.375",
		BottomMargin:    "0.375",
		MirroredMargins: true,
	}
}

func TestKDPPreflightRejectsMarksAndNonstandardBleed(t *testing.T) {
	doc := Document{}
	options := kdpOptions()
	options.IncludeCropMarks = true
	if err := validateKDPPreflight(doc, options, printPDFSpec(options)); err == nil || !strings.Contains(err.Error(), "crop") {
		t.Fatalf("crop marks error = %v", err)
	}
	options.IncludeCropMarks = false
	options.Bleed = "0.2"
	if err := validateKDPPreflight(doc, options, printPDFSpec(options)); err == nil || !strings.Contains(err.Error(), "0.125") {
		t.Fatalf("bleed error = %v", err)
	}
}

func TestKDPMarginsFollowTheRenderedPageCount(t *testing.T) {
	options := kdpOptions()
	options.Bleed = "0"
	options.GutterMargin = "0.5"
	if err := validateKDPPageCount(Document{}, options, 300); err != nil {
		t.Fatalf("300-page gutter was refused: %v", err)
	}
	if err := validateKDPPageCount(Document{}, options, 301); err == nil || !strings.Contains(err.Error(), "0.625") {
		t.Fatalf("301-page gutter error = %v", err)
	}
}

func TestKDPHardcoverUsesItsOwnPageAndTrimLimits(t *testing.T) {
	doc := Document{Edition: &DocumentEdition{Format: "Hardcover"}}
	options := kdpOptions()
	if err := validateKDPPageCount(doc, options, 74); err == nil || !strings.Contains(err.Error(), "75") {
		t.Fatalf("hardcover page error = %v", err)
	}
	options.TrimSize = "5x8"
	if err := validateKDPPreflight(doc, options, printPDFSpec(options)); err == nil || !strings.Contains(err.Error(), "hardcover trim") {
		t.Fatalf("hardcover trim error = %v", err)
	}
}
