package main

import (
	"runtime"
	"strings"

	"draftline/internal/types"
)

const (
	analysisBusyMessage       = "story analysis is already running"
	largeManuscriptSize       = 750_000
	largeAnalysisMemoryBudget = 4 * 1024 * 1024
	largeAnalysisBatchBytes   = 512 * 1024
)

type analysisRunBudget struct {
	Workers          int
	MaxInFlightBytes int
	MaxBatchBytes    int
}

// beginAnalysis serializes manuscript-scale analysis and returns a budget for
// the analysis pools only. It deliberately leaves the process-wide Go
// scheduler untouched so bindings, file I/O, the asset server, and unrelated
// work retain the machine's normal scheduling capacity.
func (a *App) beginAnalysis(book types.BookData) (budget analysisRunBudget, done func(), ok bool) {
	if !a.analysisMu.TryLock() {
		return analysisRunBudget{}, nil, false
	}
	profile := a.getSettings().AnalysisCPUProfile
	budget = analysisPoolBudget(profile, runtime.NumCPU(), manuscriptSize(book))
	return budget, func() {
		a.analysisMu.Unlock()
	}, true
}

func analysisPoolBudget(profile string, available, manuscriptBytes int) analysisRunBudget {
	if available < 1 {
		available = 1
	}
	gentle := max(1, min(2, (available+3)/4))
	balanced := max(1, min(4, (available+1)/2))
	workers := balanced
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "gentle":
		workers = gentle
	case "balanced":
		workers = balanced
	case "fast":
		workers = available
	default: // adaptive, empty, and unknown values are backwards-safe.
		workers = balanced
	}
	budget := analysisRunBudget{Workers: workers}
	if manuscriptBytes >= largeManuscriptSize {
		// The large-manuscript tier protects against several prose documents
		// being resident together. It limits queued/in-flight source bytes and
		// batch size rather than reducing scheduler or worker parallelism.
		budget.MaxInFlightBytes = largeAnalysisMemoryBudget
		budget.MaxBatchBytes = largeAnalysisBatchBytes
	}
	return budget
}

func manuscriptSize(book types.BookData) int {
	total := 0
	for _, chapter := range append(append(append([]types.ChapterItem{}, book.FrontMatter...), book.Body...), book.BackMatter...) {
		total += len(chapter.Content)
	}
	return total
}
