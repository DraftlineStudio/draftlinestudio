package types

import (
	"encoding/json"
	"math"
	"testing"
)

// The ISBNs below are invented and pass their own check digits.
const (
	invented978 = "978-1-9471345-1-5"
	invented979 = "979-8-1234567-1-2"
)

func TestADerivedISBN10FollowsThe978NumberAndAnsweringNothingFor979(t *testing.T) {
	paperback := EditionFormat{ID: "pb", Kind: EditionKindPrint, ISBN13: invented978}
	if got := paperback.DerivedISBN10(false); got != "1947134515" {
		t.Fatalf("ISBN-10 of %s: got %q, want %q", invented978, got, "1947134515")
	}
	if got := paperback.DerivedISBN10(true); got != "1-94713451-5" {
		t.Fatalf("hyphenated ISBN-10: got %q", got)
	}

	// A 979 ISBN has no ISBN-10. The screen shows nothing, not an error and
	// not a wrong number.
	ebook := EditionFormat{ID: "eb", Kind: EditionKindEbook, ISBN13: invented979}
	if got := ebook.DerivedISBN10(true); got != "" {
		t.Fatalf("a 979 ISBN has no ISBN-10, got %q", got)
	}

	// A number that is not an ISBN at all derives nothing either.
	typo := EditionFormat{ID: "x", ISBN13: "978-1-9471345-1-6"}
	if got := typo.DerivedISBN10(false); got != "" {
		t.Fatalf("a mistyped check digit produced an ISBN-10: %q", got)
	}
}

func TestSpineWidthMatchesTheStockAndBindingTable(t *testing.T) {
	cases := []struct {
		name  string
		f     EditionFormat
		want  float64
		label string
	}{
		{
			// KDP's cream-paper formula is page count x 0.0025 in.
			name:  "perfect-bound paperback on cream",
			f:     EditionFormat{Kind: EditionKindPrint, PageCount: "412", PaperStock: "Cream, 55#", Binding: "Perfect bound"},
			want:  1.030,
			label: "1.030 in",
		},
		{
			// A hardcover must use the generated case-wrap template.
			name:  "case-laminate hardcover on white",
			f:     EditionFormat{Kind: EditionKindPrint, PageCount: "428", PaperStock: "White, 60#", Binding: "Case laminate"},
			want:  0,
			label: "",
		},
		{
			name:  "groundwood stock for a long book",
			f:     EditionFormat{Kind: EditionKindPrint, PageCount: "600", PaperStock: "Groundwood, 45#", Binding: "Perfect bound"},
			want:  1.410,
			label: "1.410 in",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SpineWidthInches(tc.f); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("spine width: got %.6f, want %.6f", got, tc.want)
			}
			if got := SpineWidthLabel(tc.f); got != tc.label {
				t.Errorf("spine label: got %q, want %q", got, tc.label)
			}
		})
	}
}

func TestNothingWithoutPagesHasASpine(t *testing.T) {
	for _, f := range []EditionFormat{
		{Kind: EditionKindEbook, PageCount: "412", PaperStock: "Cream, 55#", Binding: "Perfect bound"},
		{Kind: EditionKindAudio, PageCount: "412"},
		{Kind: EditionKindPrint, PageCount: "", PaperStock: "Cream, 55#"},
		{Kind: EditionKindPrint, PageCount: "not a number"},
		{Kind: EditionKindPrint, PageCount: "0"},
	} {
		if got := SpineWidthLabel(f); got != "" {
			t.Errorf("%s format with page count %q claimed a spine of %q", f.Kind, f.PageCount, got)
		}
	}
}

func TestAnUnknownStockFallsBackToTheCommonestRatherThanToZero(t *testing.T) {
	f := EditionFormat{Kind: EditionKindPrint, PageCount: "300", PaperStock: "Recycled, 70#", Binding: "Saddle stitch"}
	// 300 x 0.0025: the cream-paper default, not 0.
	if got := SpineWidthLabel(f); got != "0.750 in" {
		t.Fatalf("unknown stock and binding: got %q, want %q", got, "0.750 in")
	}
}

func TestAnOddPrintPageCountRoundsUpForTheSpine(t *testing.T) {
	f := EditionFormat{Kind: EditionKindPrint, PageCount: "301", PaperStock: "Cream, 55#", Binding: "Perfect bound"}
	if got := SpineWidthLabel(f); got != "0.755 in" {
		t.Fatalf("odd page count spine: got %q, want %q", got, "0.755 in")
	}
}

func TestFindingAFormatNamesTheEditionItBelongsTo(t *testing.T) {
	index := &EditionIndex{Version: 1, Editions: []Edition{
		{ID: "ed-1", Label: "First edition", Formats: []EditionFormat{{ID: "ed-1-pb", Kind: EditionKindPrint}}},
		{ID: "ed-2", Label: "Second edition", Formats: []EditionFormat{{ID: "ed-2-eb", Kind: EditionKindEbook}}},
	}}
	edition, format, ok := index.FindFormat("ed-2-eb")
	if !ok || edition.Label != "Second edition" || format.Kind != EditionKindEbook {
		t.Fatalf("got %v %q %q", ok, edition.Label, format.Kind)
	}
	if _, _, ok := index.FindFormat("nothing"); ok {
		t.Fatal("found a format that does not exist")
	}
	var missing *EditionIndex
	if _, ok := missing.FindEdition("ed-1"); ok {
		t.Fatal("a book with no publishing record answered a lookup")
	}
}

func TestAnEmptyFormatSerialisesToAlmostNothing(t *testing.T) {
	// Every format of every edition rides the bridge as JSON on each save,
	// five seconds after every edit. A record whose fields are blank must not
	// cost twenty-five empty strings each time.
	raw, err := json.Marshal(EditionFormat{ID: "ed-1-eb", Kind: EditionKindEbook})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"id":"ed-1-eb","kind":"ebook"}` {
		t.Fatalf("an unfilled format serialised as %s", raw)
	}
}
