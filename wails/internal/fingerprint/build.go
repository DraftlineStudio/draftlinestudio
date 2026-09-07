// Package fingerprint builds Draftline's deterministic manuscript-memory
// model (v5): typed frames → canonical state ledgers → event identities →
// narrative developments → continuity inspections. Extraction never
// generates propositions — every free-text payload is a verbatim slice of a
// source sentence, and a frame abstains when a slot cannot be filled
// confidently.
package fingerprint

import (
	"sort"
	"time"

	"draftline/internal/types"
)

const (
	engine = "draftline-manuscript-memory-v5"
	// Schema 6 is the typed-frame corpus; schema 5 (the scrapped SPO corpus)
	// is silently discarded on the next analysis pass.
	version = 6
)

// Build reconstructs all derived manuscript-memory data while preserving
// explicit author intent (contexts, corrections, voice notes).
func Build(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.StoryFingerprint {
	prior := book.Analysis.Fingerprint
	result := &types.StoryFingerprint{
		Engine: engine, Version: version, LastAnalyzed: time.Now().UTC().Format(time.RFC3339),
		Contexts: []types.StoryContext{}, TemporalConstraints: []types.TemporalConstraint{},
		Frames: []types.NarrativeFrame{},
	}
	if book.Analysis.Evidence == nil {
		return result
	}
	result.ContentHash = book.Analysis.Evidence.ContentHash
	if prior != nil {
		result.AuthorModel = prior.AuthorModel
	}
	if progress != nil {
		progress(types.StoryAnalysisProgress{Phase: "memory", Message: "Building manuscript memory", Current: 0, Total: len(book.Analysis.Evidence.Records), Percent: 78})
	}
	records := eligibleEvidence(book.Analysis.Evidence.Records)
	contexts, contextByEvidence := inferContexts(records, result.AuthorModel.Contexts)
	result.Contexts = contexts
	applyEvidenceContextCorrections(result.AuthorModel.Corrections, contextByEvidence)
	result.TemporalConstraints = inferTemporalConstraints(records, contextByEvidence)
	points, _ := solveTemporal(records, contextByEvidence, result.TemporalConstraints)

	scopes := scopesFromContexts(result.Contexts)
	result.Frames = extractFrames(book, records, contextByEvidence, points, scopes)
	result.Ledgers = buildLedgers(result.Frames)
	result.EventIdentities = resolveEventIdentities(result.Frames)
	result.Developments = synthesizeDevelopments(book, result.Frames, result.Ledgers, result.EventIdentities)
	result.Inspections = buildInspections(result.Frames, result.Ledgers, result.EventIdentities, scopes)

	result.Profiles = deriveProfiles(records, result.AuthorModel.Profiles)
	result.Voices = buildVoiceProfiles(book, result.AuthorModel.VoiceNotes)

	result.CorpusStats = types.FingerprintCorpusStats{
		EvidenceAtoms: len(records), Frames: len(result.Frames), Abstained: abstainedCount(result.Frames),
		Ledgers: len(result.Ledgers), EventIdentities: len(result.EventIdentities),
		Developments: len(result.Developments), Inspections: len(result.Inspections),
	}
	result.FrameDiagnostic = buildFrameReport(result)
	result.DevelopmentDiagnostic = buildDevelopmentReport(result)
	result.InspectionDiagnostic = buildInspectionReport(result)
	if progress != nil {
		progress(types.StoryAnalysisProgress{Phase: "memory", Message: "Manuscript memory current", Current: len(records), Total: len(records), Percent: 82})
	}
	return result
}

func abstainedCount(frames []types.NarrativeFrame) int {
	count := 0
	for _, frame := range frames {
		if len(frame.Abstentions) > 0 {
			count++
		}
	}
	return count
}

// scopesFromContexts projects story contexts into reality scopes so frames
// can carry a scope without re-deriving context inference.
func scopesFromContexts(contexts []types.StoryContext) map[string]types.NarrativeRealityScope {
	scopes := map[string]types.NarrativeRealityScope{}
	for _, context := range contexts {
		kind := context.Kind
		switch kind {
		case "primary":
			kind = "current"
		case "memory":
			kind = "remembered"
		case "past":
			kind = "flashback"
		case "":
			kind = "uncertain"
		}
		scopes[context.ID] = types.NarrativeRealityScope{
			ID: context.ID, Kind: kind, Label: context.Label,
			ParentID: context.ParentID, Confidence: context.Confidence,
		}
	}
	return scopes
}

func eligibleEvidence(records []types.EvidenceRecord) []types.EvidenceRecord {
	result := make([]types.EvidenceRecord, 0, len(records))
	for _, record := range records {
		if record.Status != "rejected" {
			result = append(result, record)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ChapterIndex != result[j].ChapterIndex {
			return result[i].ChapterIndex < result[j].ChapterIndex
		}
		if result[i].ParagraphIndex != result[j].ParagraphIndex {
			return result[i].ParagraphIndex < result[j].ParagraphIndex
		}
		return result[i].StartOffset < result[j].StartOffset
	})
	return result
}

func chapterTitle(book *types.BookData, record types.EvidenceRecord) string {
	sections := [][]types.ChapterItem{book.FrontMatter, book.Body, book.BackMatter}
	for _, items := range sections {
		for _, chapter := range items {
			if chapter.ID != "" && chapter.ID == record.ChapterID {
				return chapter.Title
			}
		}
	}
	return "Chapter"
}

func clone(values []string) []string { return append([]string(nil), values...) }
