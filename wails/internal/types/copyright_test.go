package types

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// copyrightCaseFile is the shared table of worked examples. The TypeScript
// preview is checked against the same file, which is the only thing keeping
// the page an author sees while typing and the page printed into the exported
// book from drifting apart.
const copyrightCaseFile = "testdata/copyright_cases.json"

type copyrightCase struct {
	Name       string        `json:"name"`
	Metadata   Metadata      `json:"metadata"`
	Edition    Edition       `json:"edition"`
	Format     EditionFormat `json:"format"`
	PriorYears []string      `json:"prior_years"`
	Lines      []string      `json:"lines"`
}

func loadCopyrightCases(t *testing.T) []copyrightCase {
	t.Helper()
	raw, err := os.ReadFile(copyrightCaseFile)
	if err != nil {
		t.Fatalf("shared copyright fixture: %v", err)
	}
	var file struct {
		Cases []copyrightCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("shared copyright fixture is not readable: %v", err)
	}
	if len(file.Cases) == 0 {
		t.Fatal("shared copyright fixture holds no cases")
	}
	return file.Cases
}

func TestCopyrightPageMatchesTheSharedExamples(t *testing.T) {
	for _, tc := range loadCopyrightCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			got := CopyrightLines(tc.Metadata, tc.Edition, tc.Format, tc.PriorYears)
			if len(got) != len(tc.Lines) {
				t.Fatalf("copyright page has %d lines, expected %d:\ngot  %q\nwant %q", len(got), len(tc.Lines), got, tc.Lines)
			}
			for i := range got {
				if got[i] != tc.Lines[i] {
					t.Errorf("line %d:\ngot  %q\nwant %q", i+1, got[i], tc.Lines[i])
				}
			}
		})
	}
}

func TestCopyrightYearsAreCumulativeAndNeverRepeat(t *testing.T) {
	index := &EditionIndex{Version: 1, Editions: []Edition{
		{ID: "a", Label: "First edition", Year: "2019"},
		{ID: "b", Label: "Second edition", Year: "2024", PreviousEditionID: "a"},
		{ID: "c", Label: "Third edition", Year: "2024", PreviousEditionID: "b"},
	}}
	meta := Metadata{Title: "Harbour Lights", Author: "Ines Marrow", Publisher: "Windlass and Co."}
	format := EditionFormat{ID: "c-pb", Kind: EditionKindPrint, Format: "Paperback", PublicationDate: "2024-02-03"}

	lines := CopyrightLines(meta, index.Editions[2], format, index.PriorYears("c"))
	if lines[1] != "Copyright © 2019, 2024 by Ines Marrow" {
		t.Fatalf("a reissue in the same year as the edition before it should not print the year twice: %q", lines[1])
	}
}

func TestCopyrightYearWalkSurvivesARecordThatPointsAtItself(t *testing.T) {
	// A hand-edited file can name an edition as its own predecessor. The walk
	// has to end; an autosave five seconds after an edit must not hang.
	index := &EditionIndex{Version: 1, Editions: []Edition{
		{ID: "a", Year: "2026", PreviousEditionID: "b"},
		{ID: "b", Year: "2027", PreviousEditionID: "a"},
	}}
	done := make(chan []string, 1)
	go func() { done <- index.PriorYears("a") }()
	select {
	case years := <-done:
		if len(years) != 1 || years[0] != "2027" {
			t.Fatalf("expected the one reachable prior year, got %q", years)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("walking a self-referencing edition chain did not finish")
	}
}

func TestRevisionNoteOnlyPrintsOnALaterEdition(t *testing.T) {
	meta := Metadata{Title: "Harbour Lights", Author: "Ines Marrow", Publisher: "Windlass and Co."}
	first := Edition{ID: "a", Label: "First edition", Year: "2026", RevisionNote: "Original release."}
	format := EditionFormat{ID: "a-pb", Kind: EditionKindPrint, Format: "Paperback", PublicationDate: "2026-05-01"}

	for _, line := range CopyrightLines(meta, first, format, nil) {
		if line == "Original release." {
			t.Fatal("a first edition printed a revision note")
		}
	}

	second := Edition{ID: "b", Label: "Second edition", Year: "2031", PreviousEditionID: "a", RevisionNote: "Chapters nine to eleven condensed."}
	lines := CopyrightLines(meta, second, format, []string{"2026"})
	if lines[len(lines)-1] != "Chapters nine to eleven condensed." {
		t.Fatalf("a later edition should end on its revision note, got %q", lines[len(lines)-1])
	}
}
