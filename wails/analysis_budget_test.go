package main

import (
	"testing"

	"draftline/internal/types"
)

func TestAnalysisCoreBudget(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		available int
		bytes     int
		want      int
	}{
		{"adaptive ordinary", "adaptive", 8, largeManuscriptSize - 1, 4},
		{"adaptive large", "adaptive", 8, largeManuscriptSize, 2},
		{"gentle", "gentle", 8, 1, 2},
		{"balanced", "balanced", 8, 1, 4},
		{"fast", "fast", 8, largeManuscriptSize, 8},
		{"four core gentle", "gentle", 4, 1, 1},
		{"four core balanced", "balanced", 4, 1, 2},
		{"old setting defaults adaptive", "", 4, largeManuscriptSize, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := analysisCoreBudget(test.profile, test.available, test.bytes); got != test.want {
				t.Fatalf("analysisCoreBudget(%q, %d, %d) = %d, want %d", test.profile, test.available, test.bytes, got, test.want)
			}
		})
	}
}

func TestAnalysisSingleFlight(t *testing.T) {
	app := NewApp()
	app.setSettings(types.AppSettings{AnalysisCPUProfile: "gentle"})
	book := types.BookData{Body: []types.ChapterItem{{Content: "A short chapter."}}}

	done, ok := app.beginAnalysis(book)
	if !ok {
		t.Fatal("first analysis did not acquire the slot")
	}
	if secondDone, secondOK := app.beginAnalysis(book); secondOK || secondDone != nil {
		t.Fatal("concurrent analysis acquired the single-flight slot")
	}
	done()

	thirdDone, thirdOK := app.beginAnalysis(book)
	if !thirdOK {
		t.Fatal("analysis slot was not released")
	}
	thirdDone()
}

func TestManuscriptSizeIncludesEverySection(t *testing.T) {
	book := types.BookData{
		FrontMatter: []types.ChapterItem{{Content: "123"}},
		Body:        []types.ChapterItem{{Content: "4567"}},
		BackMatter:  []types.ChapterItem{{Content: "89"}},
	}
	if got := manuscriptSize(book); got != 9 {
		t.Fatalf("manuscriptSize = %d, want 9", got)
	}
}
