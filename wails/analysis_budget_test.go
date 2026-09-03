package main

import (
	"runtime"
	"testing"

	"draftline/internal/types"
)

func TestAnalysisPoolBudget(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		available   int
		bytes       int
		wantWorkers int
		wantMemory  int
	}{
		{"adaptive ordinary", "adaptive", 8, largeManuscriptSize - 1, 4, 0},
		{"adaptive large", "adaptive", 8, largeManuscriptSize, 4, largeAnalysisMemoryBudget},
		{"gentle", "gentle", 8, 1, 2, 0},
		{"balanced", "balanced", 8, 1, 4, 0},
		{"fast large", "fast", 8, largeManuscriptSize, 8, largeAnalysisMemoryBudget},
		{"four core gentle", "gentle", 4, 1, 1, 0},
		{"four core balanced", "balanced", 4, 1, 2, 0},
		{"old setting defaults adaptive", "", 4, largeManuscriptSize, 2, largeAnalysisMemoryBudget},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := analysisPoolBudget(test.profile, test.available, test.bytes)
			if got.Workers != test.wantWorkers || got.MaxInFlightBytes != test.wantMemory {
				t.Fatalf("analysisPoolBudget(%q, %d, %d) = %#v, want workers=%d memory=%d", test.profile, test.available, test.bytes, got, test.wantWorkers, test.wantMemory)
			}
		})
	}
}

func TestBeginAnalysisLeavesProcessSchedulerUntouched(t *testing.T) {
	before := runtime.GOMAXPROCS(0)
	app := &App{}
	app.setSettings(types.AppSettings{AnalysisCPUProfile: "gentle"})

	budget, done, ok := app.beginAnalysis(types.BookData{Body: []types.ChapterItem{{Content: "Test."}}})
	if !ok {
		t.Fatal("analysis did not acquire the slot")
	}
	if during := runtime.GOMAXPROCS(0); during != before {
		done()
		t.Fatalf("beginAnalysis changed GOMAXPROCS from %d to %d", before, during)
	}
	done()
	if after := runtime.GOMAXPROCS(0); after != before {
		t.Fatalf("finishing analysis changed GOMAXPROCS from %d to %d", before, after)
	}
	if budget.Workers >= before && before > 1 {
		t.Fatalf("gentle pool should be locally bounded below scheduler capacity: budget=%d scheduler=%d", budget.Workers, before)
	}
}

func TestAnalysisSingleFlight(t *testing.T) {
	app := NewApp()
	app.setSettings(types.AppSettings{AnalysisCPUProfile: "gentle"})
	book := types.BookData{Body: []types.ChapterItem{{Content: "A short chapter."}}}

	_, done, ok := app.beginAnalysis(book)
	if !ok {
		t.Fatal("first analysis did not acquire the slot")
	}
	if _, secondDone, secondOK := app.beginAnalysis(book); secondOK || secondDone != nil {
		t.Fatal("concurrent analysis acquired the single-flight slot")
	}
	done()

	_, thirdDone, thirdOK := app.beginAnalysis(book)
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
