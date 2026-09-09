package export

import (
	"reflect"
	"testing"
)

func TestDropCapUsesFirstLetterAndKeepsOpeningPunctuation(t *testing.T) {
	tests := []struct {
		name       string
		runs       []DocumentRun
		wantPrefix string
		wantCap    string
		wantRest   string
	}{
		{name: "curly quote", runs: []DocumentRun{{Text: "\u201cArrival was quiet."}}, wantPrefix: "\u201c", wantCap: "A", wantRest: "rrival was quiet."},
		{name: "straight quote in separate run", runs: []DocumentRun{{Text: "\""}, {Text: "Captain answered."}}, wantPrefix: "\"", wantCap: "C", wantRest: "aptain answered."},
		{name: "plain capital", runs: []DocumentRun{{Text: "Morning came."}}, wantCap: "M", wantRest: "orning came."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prefix, cap, rest := takeDropCap(test.runs)
			if prefix != test.wantPrefix || cap != test.wantCap {
				t.Fatalf("takeDropCap = prefix %q cap %q, want %q %q", prefix, cap, test.wantPrefix, test.wantCap)
			}
			block := DocumentBlock{Runs: rest}
			if block.PlainText() != test.wantRest {
				t.Fatalf("remaining text = %q, want %q", block.PlainText(), test.wantRest)
			}
		})
	}
}

func TestCodeWrappingPreservesAuthoredWhitespace(t *testing.T) {
	body := resolvePDFFont("merriweather")
	spec := publicationPDFSpec{
		TrimWidth: 396, TrimHeight: 612,
		GutterMargin: 63, OuterMargin: 45, TopMargin: 54, BottomMargin: 45,
		Font: body, CodeFont: embeddedPDFFonts["ibmplexmono"], FontSize: 9, LineHeight: 12.6,
	}
	renderer := newPublicationPDFRenderer(Document{Title: "Code"}, spec)
	renderer.pdf.SetFont(renderer.spec.CodeFont.ID, "", 7.74)
	got := renderer.wrapCodeText("E:/USB_DRIVE:\n  /Official_Maps/\n\t> README.txt", 500)
	want := []string{"E:/USB_DRIVE:", "  /Official_Maps/", "    > README.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("code lines = %#v, want %#v", got, want)
	}
}

func TestCodeFontIsOnlyRequiredByDocumentsThatUseCode(t *testing.T) {
	plain := Document{Sections: []DocumentSection{{Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{{Text: "Plain"}}}}}}}
	blockCode := Document{Sections: []DocumentSection{{Blocks: []DocumentBlock{{Kind: BlockCode, Runs: []DocumentRun{{Text: "command"}}}}}}}
	inlineCode := Document{Sections: []DocumentSection{{Blocks: []DocumentBlock{{Kind: BlockParagraph, Runs: []DocumentRun{{Text: "command", Code: true}}}}}}}
	if documentUsesCode(plain) {
		t.Fatal("plain document requested the embedded code font")
	}
	if !documentUsesCode(blockCode) || !documentUsesCode(inlineCode) {
		t.Fatal("code usage did not request the embedded code font")
	}
}

func TestAuthoredBlockAlignmentOverridesEditionDefault(t *testing.T) {
	centered := DocumentBlock{Alignment: "center"}
	right := DocumentBlock{Alignment: "right"}
	if got := blockAlignment(centered, "left"); got != "center" {
		t.Fatalf("centered block alignment = %q", got)
	}
	if got := blockAlignment(right, "left"); got != "right" {
		t.Fatalf("right-aligned block alignment = %q", got)
	}
	if got := alignedLineStart(50, 200, 80, "center"); got != 110 {
		t.Fatalf("centered line starts at %g, want 110", got)
	}
	if got := alignedLineStart(50, 200, 80, "right"); got != 170 {
		t.Fatalf("right-aligned line starts at %g, want 170", got)
	}
}
