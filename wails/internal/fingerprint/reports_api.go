package fingerprint

import "draftline/internal/types"

// TextDiagnostics rebuilds all three human-readable diagnostics from the
// archive's persisted evidence. It performs no I/O and mutates no manuscript.
func TextDiagnostics(book *types.BookData) types.FingerprintTextDiagnostics {
	if book == nil || book.Analysis.Evidence == nil {
		unavailable := "Manuscript-memory analysis has not run.\n"
		return types.FingerprintTextDiagnostics{Frames: unavailable, Developments: unavailable, Inspections: unavailable}
	}
	model := Build(book, nil)
	return types.FingerprintTextDiagnostics{
		Frames: model.FrameDiagnostic, Developments: model.DevelopmentDiagnostic, Inspections: model.InspectionDiagnostic,
	}
}
