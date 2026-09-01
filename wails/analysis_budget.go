package main

import (
	"runtime"
	"strings"

	"draftline/internal/types"
)

const (
	analysisBusyMessage = "story analysis is already running"
	largeManuscriptSize = 750_000
)

// beginAnalysis serializes manuscript-scale analysis and temporarily limits
// the Go scheduler. prose/v3 is in-process Go code, so GOMAXPROCS is the one
// cross-platform ceiling that applies to every current analysis stage rather
// than only one worker pool. It is a concurrency budget, not an exact OS CPU
// percentage; leaving runnable cores free keeps the editor and machine usable.
func (a *App) beginAnalysis(book types.BookData) (done func(), ok bool) {
	if !a.analysisMu.TryLock() {
		return nil, false
	}
	profile := a.getSettings().AnalysisCPUProfile
	cores := analysisCoreBudget(profile, runtime.NumCPU(), manuscriptSize(book))
	previous := runtime.GOMAXPROCS(cores)
	return func() {
		runtime.GOMAXPROCS(previous)
		a.analysisMu.Unlock()
	}, true
}

func analysisCoreBudget(profile string, available, manuscriptBytes int) int {
	if available < 1 {
		available = 1
	}
	gentle := max(1, min(2, (available+3)/4))
	balanced := max(1, min(4, (available+1)/2))
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "gentle":
		return gentle
	case "balanced":
		return balanced
	case "fast":
		return available
	default: // adaptive, empty, and unknown values are backwards-safe.
		if manuscriptBytes >= largeManuscriptSize {
			return gentle
		}
		return balanced
	}
}

func manuscriptSize(book types.BookData) int {
	total := 0
	for _, chapter := range append(append(append([]types.ChapterItem{}, book.FrontMatter...), book.Body...), book.BackMatter...) {
		total += len(chapter.Content)
	}
	return total
}
